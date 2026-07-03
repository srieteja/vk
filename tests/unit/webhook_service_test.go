package unit

import (
	"encoding/json"
	"net/http"
	"testing"

	"vk_backend/internal/config"
	"vk_backend/internal/models"
	"vk_backend/internal/outbox"
	"vk_backend/internal/payments"
	"vk_backend/internal/services"

	"gorm.io/gorm"
)

func setupWebhookTest(t *testing.T) (*gorm.DB, *services.PaymentService, *services.WebhookService, *payments.FakeProvider, models.Advocate, models.Client) {
	t.Helper()
	db := setupTestDB()
	provider := payments.NewFakeProvider()
	outboxService := outbox.NewService(db)
	cfg := &config.Config{PlatformCommission: 20.0, UserBRatePerMinute: 5.0, PaymentCurrency: "usd"}

	paymentService := services.NewPaymentService(db, cfg, outboxService, provider)
	webhookService := services.NewWebhookService(db, outboxService, provider)

	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60} // $1/min
	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	db.Create(&advocate)
	db.Create(&client)

	return db, paymentService, webhookService, provider, advocate, client
}

func fakeWebhookBody(t *testing.T, payload payments.FakeWebhookPayload) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal fake webhook payload: %v", err)
	}
	return data
}

func TestWebhookCompletesPaymentAndCreditsEarningsOnce(t *testing.T) {
	db, paymentService, webhookService, _, advocate, client := setupWebhookTest(t)

	payment, err := paymentService.InitiatePayment(t.Context(), client.ID, advocate.ID, 10) // 10 min @ $1/min = 1000 cents
	if err != nil {
		t.Fatalf("InitiatePayment failed: %v", err)
	}

	body := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID:          "evt_1",
		EventType:        "payment.succeeded",
		GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents:      payment.AmountCents,
		Currency:         payment.Currency,
		Status:           payments.StatusCompleted,
	})

	if err := webhookService.HandleWebhook(t.Context(), body, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	var updated models.Payment
	db.First(&updated, payment.ID)
	if updated.Status != "completed" {
		t.Errorf("expected status completed, got %s", updated.Status)
	}

	var updatedAdvocate models.Advocate
	db.First(&updatedAdvocate, advocate.ID)
	expectedEarnings := 8.0 // 1000 cents * 80% = 800 cents = $8
	if updatedAdvocate.Earnings != expectedEarnings {
		t.Errorf("expected earnings %.2f, got %.2f", expectedEarnings, updatedAdvocate.Earnings)
	}

	// Redeliver the same event: must not double-credit.
	if err := webhookService.HandleWebhook(t.Context(), body, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook (redelivery) failed: %v", err)
	}
	db.First(&updatedAdvocate, advocate.ID)
	if updatedAdvocate.Earnings != expectedEarnings {
		t.Errorf("expected earnings to remain %.2f after redelivery, got %.2f", expectedEarnings, updatedAdvocate.Earnings)
	}
}

func TestWebhookAmountMismatchFlagsInsteadOfCrediting(t *testing.T) {
	db, paymentService, webhookService, _, advocate, client := setupWebhookTest(t)

	payment, err := paymentService.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("InitiatePayment failed: %v", err)
	}

	body := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID:          "evt_2",
		EventType:        "payment.succeeded",
		GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents:      payment.AmountCents + 1, // tampered
		Currency:         payment.Currency,
		Status:           payments.StatusCompleted,
	})

	if err := webhookService.HandleWebhook(t.Context(), body, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	var updated models.Payment
	db.First(&updated, payment.ID)
	if updated.Status != "flagged" {
		t.Errorf("expected status flagged, got %s", updated.Status)
	}

	var updatedAdvocate models.Advocate
	db.First(&updatedAdvocate, advocate.ID)
	if updatedAdvocate.Earnings != 0 {
		t.Errorf("expected no earnings credited on mismatch, got %.2f", updatedAdvocate.Earnings)
	}
}

func TestWebhookMarksPaymentFailed(t *testing.T) {
	db, paymentService, webhookService, _, advocate, client := setupWebhookTest(t)

	payment, err := paymentService.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("InitiatePayment failed: %v", err)
	}

	body := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID:          "evt_3",
		EventType:        "payment.failed",
		GatewayPaymentID: payment.GatewayPaymentID,
		Status:           payments.StatusFailed,
	})

	if err := webhookService.HandleWebhook(t.Context(), body, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	var updated models.Payment
	db.First(&updated, payment.ID)
	if updated.Status != "failed" {
		t.Errorf("expected status failed, got %s", updated.Status)
	}
}

func TestWebhookRefundDebitsEarnings(t *testing.T) {
	db, paymentService, webhookService, _, advocate, client := setupWebhookTest(t)

	payment, err := paymentService.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("InitiatePayment failed: %v", err)
	}

	completeBody := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID: "evt_4a", EventType: "payment.succeeded", GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents: payment.AmountCents, Currency: payment.Currency, Status: payments.StatusCompleted,
	})
	if err := webhookService.HandleWebhook(t.Context(), completeBody, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook (complete) failed: %v", err)
	}

	refundBody := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID: "evt_4b", EventType: "refund.processed", GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents: payment.AmountCents, Currency: payment.Currency, Status: payments.StatusRefunded,
	})
	if err := webhookService.HandleWebhook(t.Context(), refundBody, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook (refund) failed: %v", err)
	}

	var updated models.Payment
	db.First(&updated, payment.ID)
	if updated.Status != "refunded" {
		t.Errorf("expected status refunded, got %s", updated.Status)
	}
	if updated.RefundedAmountCents != payment.AmountCents {
		t.Errorf("expected refunded_amount_cents %d, got %d", payment.AmountCents, updated.RefundedAmountCents)
	}

	var updatedAdvocate models.Advocate
	db.First(&updatedAdvocate, advocate.ID)
	if updatedAdvocate.Earnings != 0 {
		t.Errorf("expected earnings debited back to 0, got %.2f", updatedAdvocate.Earnings)
	}
}

func TestWebhookDisputeFlagsPaymentWithoutEarningsAdjustment(t *testing.T) {
	db, paymentService, webhookService, _, advocate, client := setupWebhookTest(t)

	payment, err := paymentService.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("InitiatePayment failed: %v", err)
	}

	completeBody := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID: "evt_6a", EventType: "payment.succeeded", GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents: payment.AmountCents, Currency: payment.Currency, Status: payments.StatusCompleted,
	})
	if err := webhookService.HandleWebhook(t.Context(), completeBody, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook (complete) failed: %v", err)
	}

	var updatedAdvocate models.Advocate
	db.First(&updatedAdvocate, advocate.ID)
	earningsAfterCompletion := updatedAdvocate.Earnings

	disputeBody := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID: "evt_6b", EventType: "charge.dispute.created", GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents: payment.AmountCents, Currency: payment.Currency, Status: payments.StatusDisputed,
	})
	if err := webhookService.HandleWebhook(t.Context(), disputeBody, http.Header{}); err != nil {
		t.Fatalf("HandleWebhook (dispute) failed: %v", err)
	}

	var updated models.Payment
	db.First(&updated, payment.ID)
	if updated.Status != "flagged" {
		t.Errorf("expected status flagged, got %s", updated.Status)
	}

	db.First(&updatedAdvocate, advocate.ID)
	if updatedAdvocate.Earnings != earningsAfterCompletion {
		t.Errorf("expected no automatic earnings adjustment on dispute (manual review), earnings changed from %.2f to %.2f",
			earningsAfterCompletion, updatedAdvocate.Earnings)
	}
}

func TestWebhookForUnknownPaymentIsAcked(t *testing.T) {
	_, _, webhookService, _, _, _ := setupWebhookTest(t)

	body := fakeWebhookBody(t, payments.FakeWebhookPayload{
		EventID: "evt_5", EventType: "payment.succeeded", GatewayPaymentID: "fake_pi_does_not_exist",
		AmountCents: 1000, Currency: "usd", Status: payments.StatusCompleted,
	})

	if err := webhookService.HandleWebhook(t.Context(), body, http.Header{}); err != nil {
		t.Errorf("expected webhook for an unknown payment to be acked (nil error), got %v", err)
	}
}
