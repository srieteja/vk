package models

import "time"

type UserA struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	Password     string    `json:"-"`
	Name         string    `json:"name"`
	Availability string    `json:"availability"`
	GoogleID     string    `json:"google_id,omitempty"`
	ProfileImage string    `json:"profile_image,omitempty"`
	Bio          string    `json:"bio,omitempty"`
	Earnings     float64   `json:"earnings"`
	HourlyRate   float64   `json:"hourly_rate"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserA) TableName() string {
	return "users_a"
}

type UserB struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	Password     string    `json:"-"`
	Name         string    `json:"name"`
	GoogleID     string    `json:"google_id,omitempty"`
	ProfileImage string    `json:"profile_image,omitempty"`
	Balance      float64   `json:"balance"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserB) TableName() string {
	return "users_b"
}

type Call struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	CallerID      uint       `json:"caller_id"`
	CallerType    string     `json:"caller_type"`
	ReceiverID    uint       `json:"receiver_id"`
	ReceiverType  string     `json:"receiver_type"`
	Status        string     `json:"status"`
	Duration      int64      `json:"duration"`
	ChargeAmount  float64    `json:"charge_amount"`
	PaymentID     *uint      `json:"payment_id,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (Call) TableName() string {
	return "calls"
}

type Payment struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserBID         uint      `json:"user_b_id"`
	UserAID         uint      `json:"user_a_id"`
	Amount          float64   `json:"amount"`
	Status          string    `json:"status"`
	TransactionID   string    `gorm:"uniqueIndex" json:"transaction_id"`
	PaymentGateway  string    `json:"payment_gateway"`
	UserACommission float64   `json:"user_a_commission"`
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