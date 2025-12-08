package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"enterprise-api/internal/models"

	"gorm.io/gorm"
)

type PaymentService struct {
	db *gorm.DB
}

func NewPaymentService(db *gorm.DB) *PaymentService {
	return &PaymentService{db: db}
}

func (s *PaymentService) InitiatePayment(clientID, advocateID uint, amount float64) (*models.Payment, error) {
	if amount <= 0 {
		return nil, errors.New("invalid amount")
	}

	// Generate unique transaction ID
	b := make([]byte, 16)
	rand.Read(b)
	txnID := fmt.Sprintf("txn_%s", hex.EncodeToString(b))

	payment := &models.Payment{
		ClientID:       clientID,
		AdvocateID:     advocateID,
		Amount:         amount,
		Status:         "pending",
		TransactionID:  txnID,
		PaymentGateway: "stripe",
	}

	if err := s.db.Create(payment).Error; err != nil {
		return nil, errors.New("failed to create payment")
	}

	return payment, nil
}

func (s *PaymentService) VerifyPayment(txnID string) (*models.Payment, error) {
	if txnID == "" {
		return nil, errors.New("invalid transaction ID")
	}

	var payment models.Payment
	if err := s.db.Where("transaction_id = ?", txnID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("transaction not found")
		}
		return nil, errors.New("database error")
	}

	payment.Status = "completed"
	if err := s.db.Save(&payment).Error; err != nil {
		return nil, errors.New("failed to update payment")
	}

	return &payment, nil
}
