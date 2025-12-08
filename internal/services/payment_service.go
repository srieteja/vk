package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"enterprise-api/internal/logger"
	"enterprise-api/internal/models"

	"gorm.io/gorm"
)

type PaymentService struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewPaymentService(db *gorm.DB) *PaymentService {
	return &PaymentService{
		db:     db,
		logger: logger.NewLogger("PaymentService", logger.INFO),
	}
}

func (s *PaymentService) InitiatePayment(clientID, advocateID uint, amount float64) (*models.Payment, error) {
	s.logger.Finer("InitiatePayment called: clientID=%d, advocateID=%d, amount=%.2f", clientID, advocateID, amount)

	if amount <= 0 {
		s.logger.Info("InitiatePayment failed: invalid amount: %.2f", amount)
		return nil, errors.New("invalid amount")
	}

	// Generate unique transaction ID
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		s.logger.Severe("InitiatePayment failed: random generation error: %v", err)
		return nil, errors.New("failed to generate transaction ID")
	}
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
		s.logger.Severe("InitiatePayment failed: database error: %v", err)
		return nil, errors.New("failed to create payment")
	}

	s.logger.Info("Payment initiated successfully: ID=%d, TransactionID=%s, Amount=%.2f", payment.ID, payment.TransactionID, payment.Amount)
	return payment, nil
}

func (s *PaymentService) VerifyPayment(txnID string) (*models.Payment, error) {
	s.logger.Finer("VerifyPayment called: transactionID=%s", txnID)

	if txnID == "" {
		s.logger.Info("VerifyPayment failed: empty transaction ID")
		return nil, errors.New("invalid transaction ID")
	}

	var payment models.Payment
	if err := s.db.Where("transaction_id = ?", txnID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("VerifyPayment failed: transaction not found: %s", txnID)
			return nil, errors.New("transaction not found")
		}
		s.logger.Severe("VerifyPayment failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	payment.Status = "completed"
	if err := s.db.Save(&payment).Error; err != nil {
		s.logger.Severe("VerifyPayment failed: failed to update payment: %v", err)
		return nil, errors.New("failed to update payment")
	}

	s.logger.Info("Payment verified successfully: ID=%d, TransactionID=%s", payment.ID, payment.TransactionID)
	return &payment, nil
}
