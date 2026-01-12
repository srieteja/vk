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
	StartedAt    *time.Time `json:"started_at,omitempty"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (Call) TableName() string {
	return "calls"
}

type Payment struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ClientID        uint      `json:"client_id"`
	AdvocateID      uint      `json:"advocate_id"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"`
	TransactionID   string    `gorm:"uniqueIndex" json:"transaction_id"`
	PaymentGateway  string    `json:"payment_gateway"`
	UserACommission float64   `json:"advocate_commission"`
	PlatformFee     float64   `json:"platform_fee"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
