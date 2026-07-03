package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

func signRazorpayPayload(secret string, body []byte) http.Header {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	h := http.Header{}
	h.Set("X-Razorpay-Signature", sig)
	return h
}

func TestRazorpayParseWebhookPaymentCaptured(t *testing.T) {
	const secret = "razorpay_webhook_secret"
	p := NewRazorpayProvider("rzp_key", "rzp_secret", secret)

	body := []byte(`{
		"event": "payment.captured",
		"payload": {
			"payment": {
				"entity": {"id": "pay_123", "order_id": "order_123", "amount": 5000, "currency": "INR"}
			}
		}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signRazorpayPayload(secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.GatewayPaymentID != "pay_123" {
		t.Errorf("expected gateway payment id pay_123, got %s", event.GatewayPaymentID)
	}
	if event.AmountCents != 5000 {
		t.Errorf("expected amount 5000, got %d", event.AmountCents)
	}
	if event.Currency != "inr" {
		t.Errorf("expected currency inr, got %s", event.Currency)
	}
	if event.Status != StatusCompleted {
		t.Errorf("expected status completed, got %s", event.Status)
	}
	if event.EventID != "payment.captured:pay_123" {
		t.Errorf("expected event id payment.captured:pay_123, got %s", event.EventID)
	}
}

func TestRazorpayParseWebhookPaymentFailed(t *testing.T) {
	const secret = "razorpay_webhook_secret"
	p := NewRazorpayProvider("rzp_key", "rzp_secret", secret)

	body := []byte(`{
		"event": "payment.failed",
		"payload": {
			"payment": {"entity": {"id": "pay_456", "amount": 1000, "currency": "INR"}}
		}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signRazorpayPayload(secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusFailed {
		t.Errorf("expected status failed, got %s", event.Status)
	}
}

func TestRazorpayParseWebhookRefundProcessed(t *testing.T) {
	const secret = "razorpay_webhook_secret"
	p := NewRazorpayProvider("rzp_key", "rzp_secret", secret)

	body := []byte(`{
		"event": "refund.processed",
		"payload": {
			"refund": {"entity": {"id": "rfnd_1", "payment_id": "pay_123", "amount": 5000, "currency": "INR"}}
		}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signRazorpayPayload(secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusRefunded {
		t.Errorf("expected status refunded, got %s", event.Status)
	}
	if event.GatewayPaymentID != "pay_123" {
		t.Errorf("expected gateway payment id pay_123 (not the refund id), got %s", event.GatewayPaymentID)
	}
}

func TestRazorpayParseWebhookDisputeCreated(t *testing.T) {
	const secret = "razorpay_webhook_secret"
	p := NewRazorpayProvider("rzp_key", "rzp_secret", secret)

	body := []byte(`{
		"event": "payment.dispute.created",
		"payload": {
			"dispute": {"entity": {"id": "disp_1", "payment_id": "pay_123", "amount": 5000, "currency": "INR"}}
		}
	}`)

	event, err := p.ParseWebhook(t.Context(), body, signRazorpayPayload(secret, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Status != StatusDisputed {
		t.Errorf("expected status disputed, got %s", event.Status)
	}
	if event.GatewayPaymentID != "pay_123" {
		t.Errorf("expected gateway payment id pay_123, got %s", event.GatewayPaymentID)
	}
}

func TestRazorpayParseWebhookRejectsBadSignature(t *testing.T) {
	p := NewRazorpayProvider("rzp_key", "rzp_secret", "real_secret")

	body := []byte(`{"event": "payment.captured", "payload": {"payment": {"entity": {"id": "pay_1"}}}}`)
	badHeaders := signRazorpayPayload("wrong_secret", body)

	_, err := p.ParseWebhook(t.Context(), body, badHeaders)
	if err != ErrSignatureInvalid {
		t.Errorf("expected ErrSignatureInvalid, got %v", err)
	}
}
