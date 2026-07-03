// Package payments abstracts payment gateways (Stripe, Razorpay, a fake for
// tests) behind a single Provider interface, so the rest of the app never
// branches on which gateway is configured.
package payments

import (
	"context"
	"errors"
	"net/http"
)

// ErrSignatureInvalid is returned by ParseWebhook when the request's
// signature doesn't verify against the configured webhook secret.
var ErrSignatureInvalid = errors.New("payments: invalid webhook signature")

// PaymentStatus is a normalized status derived from a provider-specific
// webhook event type.
type PaymentStatus string

const (
	StatusCompleted PaymentStatus = "completed"
	StatusFailed    PaymentStatus = "failed"
	StatusRefunded  PaymentStatus = "refunded"
	// StatusDisputed means the cardholder's bank has filed a chargeback.
	// v1 handling is manual: the payment is flagged for a human to review,
	// with no automatic earnings adjustment (unlike a clean refund).
	StatusDisputed PaymentStatus = "disputed"
	// StatusPending means the event doesn't represent a state this app acts
	// on (e.g. a provider event outside the set we handle); the caller
	// should acknowledge it (2xx) without applying a state transition.
	StatusPending PaymentStatus = "pending"
)

type CreateIntentParams struct {
	AmountCents int64
	// Currency is a lowercase ISO 4217 code (e.g. "usd", "inr").
	Currency string
	// Reference is our internal transaction ID, attached as gateway
	// metadata so a payment can be traced back to our records independent
	// of the gateway's own ID.
	Reference string
	Metadata  map[string]string
}

// Intent is the result of starting a payment with the gateway.
type Intent struct {
	// GatewayPaymentID correlates this payment at the provider. Once the
	// payment completes, it's also the ID used to issue a refund.
	GatewayPaymentID string
	// ClientSecret is returned to the caller for client-side confirmation.
	// Empty for providers that don't use one (e.g. Razorpay, which instead
	// hands the frontend an order ID + the publishable key for Checkout.js).
	ClientSecret string
	// Status is the provider's raw initial status, for logging/debugging
	// only -- callers should not branch on it.
	Status string
}

// WebhookEvent is a normalized view of a provider webhook notification.
type WebhookEvent struct {
	// EventID is the provider's event ID, used for delivery deduplication.
	EventID string
	// EventType is the provider's raw event type, for logging only.
	EventType string
	// GatewayPaymentID correlates back to the payment this event is about.
	GatewayPaymentID string
	// AmountCents is the amount charged (Status == StatusCompleted) or
	// refunded (Status == StatusRefunded) by this specific event.
	AmountCents int64
	Currency    string
	Status      PaymentStatus
}

type RefundParams struct {
	GatewayPaymentID string
	// AmountCents is the amount to refund. v1 only issues full refunds, but
	// the field is amount-based (not a bool) so a provider implementation
	// isn't hard-coded to "refund everything."
	AmountCents int64
	Reason      string
}

type Refund struct {
	GatewayRefundID string
	// Status is the provider's raw status, for logging/debugging only --
	// the refund isn't considered final until confirmed by webhook.
	Status string
}

// Provider is implemented by each supported payment gateway.
type Provider interface {
	Name() string
	CreateIntent(ctx context.Context, params CreateIntentParams) (*Intent, error)
	// ParseWebhook verifies the request signature and normalizes the body
	// into a WebhookEvent. headers must include whatever signature header
	// the provider uses (e.g. Stripe-Signature, X-Razorpay-Signature).
	ParseWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookEvent, error)
	CreateRefund(ctx context.Context, params RefundParams) (*Refund, error)
}
