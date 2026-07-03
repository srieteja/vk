package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	razorpaygo "github.com/razorpay/razorpay-go"
	razorpayutils "github.com/razorpay/razorpay-go/utils"
)

type RazorpayProvider struct {
	client        *razorpaygo.Client
	webhookSecret string
}

func NewRazorpayProvider(keyID, keySecret, webhookSecret string) *RazorpayProvider {
	return &RazorpayProvider{
		client:        razorpaygo.NewClient(keyID, keySecret),
		webhookSecret: webhookSecret,
	}
}

func (p *RazorpayProvider) Name() string { return "razorpay" }

func (p *RazorpayProvider) CreateIntent(ctx context.Context, params CreateIntentParams) (*Intent, error) {
	data := map[string]interface{}{
		"amount":   params.AmountCents, // Razorpay also bills in the smallest currency unit (e.g. paise for INR)
		"currency": strings.ToUpper(params.Currency),
		"receipt":  params.Reference,
	}
	if len(params.Metadata) > 0 {
		notes := make(map[string]interface{}, len(params.Metadata))
		for k, v := range params.Metadata {
			notes[k] = v
		}
		data["notes"] = notes
	}

	order, err := p.client.Order.Create(data, nil)
	if err != nil {
		return nil, err
	}

	orderID, _ := order["id"].(string)
	if orderID == "" {
		return nil, errors.New("razorpay: order create response missing id")
	}
	status, _ := order["status"].(string)

	return &Intent{
		// The order ID, not a payment ID, since Razorpay only creates the
		// actual payment once the customer completes checkout. The webhook
		// handler overwrites this with the real payment ID once captured --
		// by the time a refund is possible, this always holds a payment ID.
		GatewayPaymentID: orderID,
		// Razorpay Checkout.js takes the order ID + publishable key
		// directly; there's no client-secret concept to return here.
		ClientSecret: "",
		Status:       status,
	}, nil
}

type razorpayWebhookPayload struct {
	Event   string `json:"event"`
	Payload struct {
		Payment struct {
			Entity struct {
				ID       string `json:"id"`
				OrderID  string `json:"order_id"`
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
			} `json:"entity"`
		} `json:"payment"`
		Refund struct {
			Entity struct {
				ID        string `json:"id"`
				PaymentID string `json:"payment_id"`
				Amount    int64  `json:"amount"`
				Currency  string `json:"currency"`
			} `json:"entity"`
		} `json:"refund"`
		Dispute struct {
			Entity struct {
				ID        string `json:"id"`
				PaymentID string `json:"payment_id"`
				Amount    int64  `json:"amount"`
				Currency  string `json:"currency"`
			} `json:"entity"`
		} `json:"dispute"`
	} `json:"payload"`
}

func (p *RazorpayProvider) ParseWebhook(ctx context.Context, payload []byte, headers http.Header) (*WebhookEvent, error) {
	signature := headers.Get("X-Razorpay-Signature")
	if !razorpayutils.VerifyWebhookSignature(string(payload), signature, p.webhookSecret) {
		return nil, ErrSignatureInvalid
	}

	var wp razorpayWebhookPayload
	if err := json.Unmarshal(payload, &wp); err != nil {
		return nil, err
	}

	base := &WebhookEvent{EventType: wp.Event, EventID: wp.Event, Status: StatusPending}

	switch wp.Event {
	case "payment.captured":
		e := wp.Payload.Payment.Entity
		base.EventID = fmt.Sprintf("%s:%s", wp.Event, e.ID)
		base.GatewayPaymentID = e.ID
		base.AmountCents = e.Amount
		base.Currency = strings.ToLower(e.Currency)
		base.Status = StatusCompleted

	case "payment.failed":
		e := wp.Payload.Payment.Entity
		base.EventID = fmt.Sprintf("%s:%s", wp.Event, e.ID)
		base.GatewayPaymentID = e.ID
		base.AmountCents = e.Amount
		base.Currency = strings.ToLower(e.Currency)
		base.Status = StatusFailed

	case "refund.processed":
		e := wp.Payload.Refund.Entity
		base.EventID = fmt.Sprintf("%s:%s", wp.Event, e.ID)
		base.GatewayPaymentID = e.PaymentID
		base.AmountCents = e.Amount
		base.Currency = strings.ToLower(e.Currency)
		base.Status = StatusRefunded

	case "payment.dispute.created":
		e := wp.Payload.Dispute.Entity
		base.EventID = fmt.Sprintf("%s:%s", wp.Event, e.ID)
		base.GatewayPaymentID = e.PaymentID
		base.AmountCents = e.Amount
		base.Currency = strings.ToLower(e.Currency)
		base.Status = StatusDisputed
	}

	return base, nil
}

func (p *RazorpayProvider) CreateRefund(ctx context.Context, params RefundParams) (*Refund, error) {
	refund, err := p.client.Payment.Refund(params.GatewayPaymentID, int(params.AmountCents), nil, nil)
	if err != nil {
		return nil, err
	}

	id, _ := refund["id"].(string)
	if id == "" {
		return nil, errors.New("razorpay: refund create response missing id")
	}
	status, _ := refund["status"].(string)

	return &Refund{GatewayRefundID: id, Status: status}, nil
}
