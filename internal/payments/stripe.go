package payments

import (
	"context"
	"encoding/json"
	"net/http"

	stripego "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeProvider struct {
	client        *stripego.Client
	webhookSecret string
}

func NewStripeProvider(secretKey, webhookSecret string) *StripeProvider {
	return &StripeProvider{
		client:        stripego.NewClient(secretKey),
		webhookSecret: webhookSecret,
	}
}

func (p *StripeProvider) Name() string { return "stripe" }

func (p *StripeProvider) CreateIntent(ctx context.Context, params CreateIntentParams) (*Intent, error) {
	stripeParams := &stripego.PaymentIntentCreateParams{
		Amount:   stripego.Int64(params.AmountCents),
		Currency: stripego.String(params.Currency),
	}
	stripeParams.AddMetadata("reference", params.Reference)
	for k, v := range params.Metadata {
		stripeParams.AddMetadata(k, v)
	}

	pi, err := p.client.V1PaymentIntents.Create(ctx, stripeParams)
	if err != nil {
		return nil, err
	}

	return &Intent{
		GatewayPaymentID: pi.ID,
		ClientSecret:     pi.ClientSecret,
		Status:           string(pi.Status),
	}, nil
}

// stripePaymentIntentObject and stripeChargeObject deliberately re-declare
// only the fields we need, rather than unmarshaling into stripego's full
// generated types: Stripe's "expandable" fields (e.g. Charge.payment_intent)
// serialize as a plain string ID in webhook payloads unless expansion was
// requested, which doesn't unmarshal cleanly into stripego's *PaymentIntent
// pointer field.
type stripePaymentIntentObject struct {
	ID       string `json:"id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type stripeChargeObject struct {
	PaymentIntent  string `json:"payment_intent"`
	AmountRefunded int64  `json:"amount_refunded"`
	Currency       string `json:"currency"`
}

type stripeDisputeObject struct {
	PaymentIntent string `json:"payment_intent"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}

func (p *StripeProvider) ParseWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookEvent, error) {
	// IgnoreAPIVersionMismatch: a webhook endpoint's configured API version
	// can legitimately drift from the version this SDK was built against;
	// we only read a handful of stable fields, so we don't need to hard-fail
	// on that instead of just verifying the signature.
	event, err := webhook.ConstructEventWithOptions(payload, headers.Get("Stripe-Signature"), p.webhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		return nil, ErrSignatureInvalid
	}

	base := &WebhookEvent{
		EventID:   event.ID,
		EventType: string(event.Type),
		Status:    StatusPending,
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripePaymentIntentObject
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, err
		}
		base.GatewayPaymentID = pi.ID
		base.AmountCents = pi.Amount
		base.Currency = pi.Currency
		base.Status = StatusCompleted

	case "payment_intent.payment_failed":
		var pi stripePaymentIntentObject
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return nil, err
		}
		base.GatewayPaymentID = pi.ID
		base.AmountCents = pi.Amount
		base.Currency = pi.Currency
		base.Status = StatusFailed

	case "charge.refunded":
		var ch stripeChargeObject
		if err := json.Unmarshal(event.Data.Raw, &ch); err != nil {
			return nil, err
		}
		base.GatewayPaymentID = ch.PaymentIntent
		base.AmountCents = ch.AmountRefunded
		base.Currency = ch.Currency
		base.Status = StatusRefunded

	case "charge.dispute.created":
		var d stripeDisputeObject
		if err := json.Unmarshal(event.Data.Raw, &d); err != nil {
			return nil, err
		}
		base.GatewayPaymentID = d.PaymentIntent
		base.AmountCents = d.Amount
		base.Currency = d.Currency
		base.Status = StatusDisputed
	}

	return base, nil
}

func (p *StripeProvider) CreateRefund(ctx context.Context, params RefundParams) (*Refund, error) {
	refundParams := &stripego.RefundCreateParams{
		PaymentIntent: stripego.String(params.GatewayPaymentID),
	}
	if params.AmountCents > 0 {
		refundParams.Amount = stripego.Int64(params.AmountCents)
	}

	r, err := p.client.V1Refunds.Create(ctx, refundParams)
	if err != nil {
		return nil, err
	}

	return &Refund{GatewayRefundID: r.ID, Status: string(r.Status)}, nil
}
