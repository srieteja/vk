package payments

import "testing"

// Compile-time checks that every provider satisfies Provider.
var (
	_ Provider = (*FakeProvider)(nil)
	_ Provider = (*StripeProvider)(nil)
	_ Provider = (*RazorpayProvider)(nil)
)

func TestFakeProviderCreateIntent(t *testing.T) {
	p := NewFakeProvider()

	intent, err := p.CreateIntent(t.Context(), CreateIntentParams{AmountCents: 5000, Currency: "usd", Reference: "txn_1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if intent.GatewayPaymentID == "" {
		t.Error("expected a non-empty gateway payment id")
	}
	if intent.ClientSecret == "" {
		t.Error("expected a non-empty client secret")
	}

	second, err := p.CreateIntent(t.Context(), CreateIntentParams{AmountCents: 5000, Currency: "usd", Reference: "txn_2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.GatewayPaymentID == intent.GatewayPaymentID {
		t.Error("expected distinct gateway payment ids across calls")
	}
}

func TestFakeProviderParseWebhook(t *testing.T) {
	p := NewFakeProvider()

	payload := []byte(`{"event_id":"evt_1","event_type":"payment.succeeded","gateway_payment_id":"fake_pi_1","amount_cents":5000,"currency":"usd","status":"completed"}`)

	event, err := p.ParseWebhook(t.Context(), payload, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", event.Status)
	}
	if event.GatewayPaymentID != "fake_pi_1" {
		t.Errorf("expected gateway payment id fake_pi_1, got %s", event.GatewayPaymentID)
	}
}

func TestFakeProviderParseWebhookRejectsMissingEventID(t *testing.T) {
	p := NewFakeProvider()

	_, err := p.ParseWebhook(t.Context(), []byte(`{"status":"completed"}`), nil)
	if err == nil {
		t.Error("expected an error for a payload missing event_id")
	}
}

func TestFakeProviderCreateRefund(t *testing.T) {
	p := NewFakeProvider()

	refund, err := p.CreateRefund(t.Context(), RefundParams{GatewayPaymentID: "fake_pi_1", AmountCents: 5000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refund.GatewayRefundID == "" {
		t.Error("expected a non-empty gateway refund id")
	}
}
