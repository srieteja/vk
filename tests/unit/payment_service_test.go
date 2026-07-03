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

func setupTestConfig() *config.Config {
	return &config.Config{
		PlatformCommission: 20.0, // 20% commission
		UserBRatePerMinute: 5.0,
		PaymentCurrency:    "usd",
	}
}

func newTestPaymentService(db *gorm.DB) *services.PaymentService {
	return services.NewPaymentService(db, setupTestConfig(), nil, payments.NewFakeProvider())
}

func TestInitiatePayment(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid", Balance: 100}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60} // $1/min = 100 cents/min
	db.Create(&client)
	db.Create(&advocate)

	payment, err := service.InitiatePayment(t.Context(), client.ID, advocate.ID, 10) // 10 min

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedAmountCents := int64(1000) // 10 min * 100 cents/min
	if payment.AmountCents != expectedAmountCents {
		t.Errorf("expected amount_cents %d, got %d", expectedAmountCents, payment.AmountCents)
	}

	expectedFee := int64(200) // 20% of 1000
	if payment.PlatformFeeCents != expectedFee {
		t.Errorf("expected platform fee cents %d, got %d", expectedFee, payment.PlatformFeeCents)
	}

	expectedCommission := expectedAmountCents - expectedFee
	if payment.AdvocateCommissionCents != expectedCommission {
		t.Errorf("expected advocate commission cents %d, got %d", expectedCommission, payment.AdvocateCommissionCents)
	}

	if payment.Status != "processing" {
		t.Errorf("expected status processing, got %s", payment.Status)
	}

	if payment.GatewayPaymentID == "" {
		t.Error("expected a gateway payment id to be recorded")
	}

	if payment.ClientSecret == "" {
		t.Error("expected a client secret to be returned")
	}
}

func TestInitiatePaymentFallsBackToPlatformRateWhenAdvocateHasNoHourlyRate(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 0}
	db.Create(&client)
	db.Create(&advocate)

	payment, err := service.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedAmountCents := int64(5000) // 10 min * $5/min fallback = 5000 cents
	if payment.AmountCents != expectedAmountCents {
		t.Errorf("expected amount_cents %d, got %d", expectedAmountCents, payment.AmountCents)
	}
}

func TestInitiatePayment_InvalidDuration(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60}
	db.Create(&advocate)

	_, err := service.InitiatePayment(t.Context(), 1, advocate.ID, 0)
	if err == nil {
		t.Fatal("expected error for zero duration")
	}
}

func TestInitiatePayment_UnknownAdvocate(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	_, err := service.InitiatePayment(t.Context(), 1, 999, 10)
	if err == nil {
		t.Fatal("expected error for unknown advocate")
	}
}

func TestGetPaymentStatusDoesNotMutateOrCreditEarnings(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60, Earnings: 0}
	db.Create(&client)
	db.Create(&advocate)

	p, err := service.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		payment, err := service.GetPaymentStatus(p.TransactionID, client.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if payment.Status != "processing" {
			t.Errorf("expected status to remain processing (read-only lookup), got %s", payment.Status)
		}
	}

	var updatedAdvocate models.Advocate
	db.First(&updatedAdvocate, advocate.ID)
	if updatedAdvocate.Earnings != 0 {
		t.Errorf("expected earnings to remain 0 (only a webhook can credit), got %f", updatedAdvocate.Earnings)
	}
}

func TestGetPaymentStatusWrongOwnerReturnsSameErrorAsNotFound(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	otherClient := models.Client{Name: "Other", Email: "other@test.com", UUID: "other-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60}
	db.Create(&client)
	db.Create(&otherClient)
	db.Create(&advocate)

	p, err := service.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, wrongOwnerErr := service.GetPaymentStatus(p.TransactionID, otherClient.ID)
	_, notFoundErr := service.GetPaymentStatus("txn_does_not_exist", otherClient.ID)

	if wrongOwnerErr == nil {
		t.Fatal("expected an error when checking someone else's transaction")
	}
	if wrongOwnerErr.Error() != notFoundErr.Error() {
		t.Errorf("wrong-owner and not-found errors must be identical to avoid enumeration, got %q vs %q",
			wrongOwnerErr.Error(), notFoundErr.Error())
	}
}

func TestRefundPaymentRejectsNonCompletedPayment(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60}
	db.Create(&client)
	db.Create(&advocate)

	p, err := service.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Payment is "processing", not "completed" -- refund must be rejected.
	if err := service.RefundPayment(t.Context(), p.TransactionID); err == nil {
		t.Error("expected an error refunding a non-completed payment")
	}
}

func TestRefundPaymentSucceedsForCompletedPayment(t *testing.T) {
	db := setupTestDB()
	provider := payments.NewFakeProvider()
	outboxService := outbox.NewService(db)
	cfg := setupTestConfig()
	service := services.NewPaymentService(db, cfg, outboxService, provider)
	webhookService := services.NewWebhookService(db, outboxService, provider)

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60}
	db.Create(&client)
	db.Create(&advocate)

	p, err := service.InitiatePayment(t.Context(), client.ID, advocate.ID, 10)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	body, err := json.Marshal(payments.FakeWebhookPayload{
		EventID: "evt_refund_setup", EventType: "payment.succeeded", GatewayPaymentID: p.GatewayPaymentID,
		AmountCents: p.AmountCents, Currency: p.Currency, Status: payments.StatusCompleted,
	})
	if err != nil {
		t.Fatalf("failed to marshal webhook payload: %v", err)
	}
	if err := webhookService.HandleWebhook(t.Context(), body, http.Header{}); err != nil {
		t.Fatalf("failed to complete payment via webhook: %v", err)
	}

	if err := service.RefundPayment(t.Context(), p.TransactionID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// RefundPayment only triggers the gateway call; status stays
	// "completed" until the refund webhook confirms it.
	var payment models.Payment
	db.First(&payment, p.ID)
	if payment.Status != "completed" {
		t.Errorf("expected status to remain completed until refund webhook arrives, got %s", payment.Status)
	}
}

func TestRefundPaymentRejectsMismatchedProvider(t *testing.T) {
	db := setupTestDB()
	service := newTestPaymentService(db) // active provider is "fake"

	client := models.Client{Name: "Client", Email: "c@test.com", UUID: "c-uuid"}
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid", HourlyRate: 60}
	db.Create(&client)
	db.Create(&advocate)

	// A payment recorded against a different (e.g. since-deactivated)
	// provider than the one currently active.
	payment := models.Payment{
		ClientID: client.ID, AdvocateID: advocate.ID, AmountCents: 1000, Currency: "usd",
		Status: "completed", TransactionID: "txn_other_provider", PaymentGateway: "stripe",
		GatewayPaymentID: "pi_123",
	}
	db.Create(&payment)

	if err := service.RefundPayment(t.Context(), payment.TransactionID); err == nil {
		t.Error("expected an error refunding a payment made via a different provider than the active one")
	}
}
