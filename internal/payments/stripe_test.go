package payments

import (
	"net/http"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v82/webhook"
)

func signStripeEvent(t *testing.T, secret string, body []byte) http.Header {
	t.Helper()
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload:   body,
		Secret:    secret,
		Timestamp: time.Now(),
	})
	h := http.Header{}
	h.Set("Stripe-Signature", signed.Header)
	return h
}

func TestStripeParseWebhookPaymentSucceeded(t *testing.T) {
	const secret = "whsec_test_secret"
	p := NewStripeProvider("sk_test_x", secret)

	body := []byte(`{
		"id": "evt_123",
		"type": "payment_intent.succeeded",
		"data": {"object": {"id": "pi_123", "amount": 5000, "currency": "usd"}}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signStripeEvent(t, secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.EventID != "evt_123" {
		t.Errorf("expected event id evt_123, got %s", event.EventID)
	}
	if event.GatewayPaymentID != "pi_123" {
		t.Errorf("expected gateway payment id pi_123, got %s", event.GatewayPaymentID)
	}
	if event.AmountCents != 5000 {
		t.Errorf("expected amount 5000, got %d", event.AmountCents)
	}
	if event.Currency != "usd" {
		t.Errorf("expected currency usd, got %s", event.Currency)
	}
	if event.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", event.Status)
	}
}

func TestStripeParseWebhookPaymentFailed(t *testing.T) {
	const secret = "whsec_test_secret"
	p := NewStripeProvider("sk_test_x", secret)

	body := []byte(`{
		"id": "evt_456",
		"type": "payment_intent.payment_failed",
		"data": {"object": {"id": "pi_456", "amount": 2500, "currency": "inr"}}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signStripeEvent(t, secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusFailed {
		t.Errorf("expected status failed, got %s", event.Status)
	}
}

func TestStripeParseWebhookChargeRefunded(t *testing.T) {
	const secret = "whsec_test_secret"
	p := NewStripeProvider("sk_test_x", secret)

	body := []byte(`{
		"id": "evt_789",
		"type": "charge.refunded",
		"data": {"object": {"payment_intent": "pi_123", "amount_refunded": 5000, "currency": "usd"}}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signStripeEvent(t, secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusRefunded {
		t.Errorf("expected status refunded, got %s", event.Status)
	}
	if event.GatewayPaymentID != "pi_123" {
		t.Errorf("expected gateway payment id pi_123, got %s", event.GatewayPaymentID)
	}
	if event.AmountCents != 5000 {
		t.Errorf("expected refunded amount 5000, got %d", event.AmountCents)
	}
}

func TestStripeParseWebhookChargeDisputeCreated(t *testing.T) {
	const secret = "whsec_test_secret"
	p := NewStripeProvider("sk_test_x", secret)

	body := []byte(`{
		"id": "evt_dispute_1",
		"type": "charge.dispute.created",
		"data": {"object": {"payment_intent": "pi_123", "amount": 5000, "currency": "usd"}}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signStripeEvent(t, secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusDisputed {
		t.Errorf("expected status disputed, got %s", event.Status)
	}
	if event.GatewayPaymentID != "pi_123" {
		t.Errorf("expected gateway payment id pi_123, got %s", event.GatewayPaymentID)
	}
}

func TestStripeParseWebhookUnrecognizedEventIsPending(t *testing.T) {
	const secret = "whsec_test_secret"
	p := NewStripeProvider("sk_test_x", secret)

	body := []byte(`{"id": "evt_999", "type": "customer.created", "data": {"object": {}}}`)

	event, err := p.ParseWebhook(t.Context(), body, signStripeEvent(t, secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusPending {
		t.Errorf("expected unrecognized event to normalize to pending, got %s", event.Status)
	}
}

func TestStripeParseWebhookRejectsBadSignature(t *testing.T) {
	p := NewStripeProvider("sk_test_x", "whsec_real_secret")

	body := []byte(`{"id": "evt_123", "type": "payment_intent.succeeded", "data": {"object": {}}}`)
	badHeaders := signStripeEvent(t, "whsec_wrong_secret", body)

	_, err := p.ParseWebhook(t.Context(), body, badHeaders)
	if err != ErrSignatureInvalid {
		t.Errorf("expected ErrSignatureInvalid, got %v", err)
	}
}
