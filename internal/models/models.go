package models

import "time"

type Advocate struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	GoogleID     *string   `gorm:"uniqueIndex" json:"google_id,omitempty"`
	Password     string    `json:"-"`
	Name         string    `json:"name"`
	Availability string    `json:"availability"`
	UUID         string    `gorm:"type:uuid;uniqueIndex" json:"uuid"`
	Location     string    `json:"location,omitempty"` // City/Location
	ProfileImage string    `json:"profile_image,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	Earnings     float64   `json:"earnings"`
	HourlyRate   float64   `json:"hourly_rate"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Advocate) TableName() string {
	return "advocates"
}

type Client struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	GoogleID     *string   `gorm:"uniqueIndex" json:"google_id,omitempty"`
	Password     string    `json:"-"`
	Name         string    `json:"name"`
	UUID         string    `gorm:"type:uuid;uniqueIndex" json:"uuid"`
	ProfileImage string    `json:"profile_image,omitempty"`
	Balance      float64   `json:"balance"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Client) TableName() string {
	return "clients"
}

type Call struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	CallerID     uint       `json:"caller_id"`
	CallerType   string     `json:"caller_type"`
	ReceiverID   uint       `json:"receiver_id"`
	ReceiverType string     `json:"receiver_type"`
	Status       string     `json:"status"`
	Duration     int64      `json:"duration"`
	ChargeAmount float64    `json:"charge_amount"`
	PaymentID    *uint      `json:"payment_id,omitempty"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (Call) TableName() string {
	return "calls"
}

type Payment struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	ClientID                uint      `json:"client_id"`
	AdvocateID              uint      `json:"advocate_id"`
	AmountCents             int64     `json:"amount_cents"`
	Currency                string    `json:"currency"`
	DurationMinutes         int       `json:"duration_minutes"`
	Status                  string    `json:"status"`
	TransactionID           string    `gorm:"uniqueIndex" json:"transaction_id"`
	PaymentGateway          string    `json:"payment_gateway"`
	GatewayPaymentID        string    `json:"gateway_payment_id,omitempty"`
	AdvocateCommissionCents int64     `json:"advocate_commission_cents"`
	PlatformFeeCents        int64     `json:"platform_fee_cents"`
	RefundedAmountCents     int64     `json:"refunded_amount_cents"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	// ClientSecret is set only on the InitiatePayment response for
	// client-side gateway confirmation; never persisted (gateways don't
	// need it stored -- Stripe keeps it retrievable via the PaymentIntent
	// itself, and Razorpay doesn't use one at all).
	ClientSecret string `gorm:"-" json:"client_secret,omitempty"`
}

func (Payment) TableName() string {
	return "payments"
}

type Session struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	UserType  string    `json:"user_type"`
	Token     string    `gorm:"uniqueIndex" json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (Session) TableName() string {
	return "sessions"
}

type OutboxEvent struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	AggregateType string     `json:"aggregate_type"`
	AggregateID   string     `json:"aggregate_id"`
	EventType     string     `json:"event_type"`
	Payload       []byte     `gorm:"type:jsonb" json:"payload"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"last_error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

type IdempotencyKey struct {
	Key            string    `gorm:"primaryKey" json:"key"`
	UserID         uint      `json:"user_id"`
	UserType       string    `json:"user_type"`
	RequestHash    string    `json:"request_hash"`
	ResponseStatus int       `json:"response_status"`
	ResponseBody   string    `json:"response_body"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}

// WebhookEvent records every gateway webhook delivery we've processed, keyed
// by (provider, event_id), so a redelivered event is a no-op instead of
// double-crediting or double-refunding a payment.
type WebhookEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Provider  string    `gorm:"uniqueIndex:idx_webhook_provider_event" json:"provider"`
	EventID   string    `gorm:"uniqueIndex:idx_webhook_provider_event" json:"event_id"`
	EventType string    `json:"event_type"`
	Payload   []byte    `gorm:"type:jsonb" json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

func (WebhookEvent) TableName() string {
	return "webhook_events"
}
