package payments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
)

// FakeProvider is an in-memory Provider for tests. It has no real signature
// scheme: tests construct a FakeWebhookPayload directly and marshal it as
// the request body rather than receiving one from a real gateway.
type FakeProvider struct{}

func NewFakeProvider() *FakeProvider {
	return &FakeProvider{}
}

func (p *FakeProvider) Name() string { return "fake" }

func (p *FakeProvider) CreateIntent(ctx context.Context, params CreateIntentParams) (*Intent, error) {
	id, err := randomID("fake_pi_")
	if err != nil {
		return nil, err
	}
	return &Intent{GatewayPaymentID: id, ClientSecret: id + "_secret", Status: "requires_payment"}, nil
}

// FakeWebhookPayload is the plain-JSON body FakeProvider.ParseWebhook
// expects; tests build one directly to simulate a gateway webhook.
type FakeWebhookPayload struct {
	EventID          string        `json:"event_id"`
	EventType        string        `json:"event_type"`
	GatewayPaymentID string        `json:"gateway_payment_id"`
	AmountCents      int64         `json:"amount_cents"`
	Currency         string        `json:"currency"`
	Status           PaymentStatus `json:"status"`
}

func (p *FakeProvider) ParseWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookEvent, error) {
	var fw FakeWebhookPayload
	if err := json.Unmarshal(payload, &fw); err != nil {
		return nil, err
	}
	if fw.EventID == "" {
		return nil, errors.New("fake webhook payload missing event_id")
	}
	return &WebhookEvent{
		EventID:          fw.EventID,
		EventType:        fw.EventType,
		GatewayPaymentID: fw.GatewayPaymentID,
		AmountCents:      fw.AmountCents,
		Currency:         fw.Currency,
		Status:           fw.Status,
	}, nil
}

func (p *FakeProvider) CreateRefund(ctx context.Context, params RefundParams) (*Refund, error) {
	id, err := randomID("fake_re_")
	if err != nil {
		return nil, err
	}
	return &Refund{GatewayRefundID: id, Status: "succeeded"}, nil
}

func randomID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}
