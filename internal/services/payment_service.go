package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"vk_backend/internal/config"
	"vk_backend/internal/logger"
	"vk_backend/internal/models"
	"vk_backend/internal/outbox"

	"gorm.io/gorm"
)

type PaymentService struct {
	db            *gorm.DB
	cfg           *config.Config
	outboxService *outbox.Service
	logger        *logger.Logger
}

func NewPaymentService(db *gorm.DB, cfg *config.Config, outboxService *outbox.Service) *PaymentService {
	return &PaymentService{
		db:            db,
		cfg:           cfg,
		outboxService: outboxService,
		logger:        logger.NewLogger("PaymentService", logger.INFO),
	}
}

func (s *PaymentService) InitiatePayment(clientID, advocateID uint, amount float64) (*models.Payment, error) {
	s.logger.Finer("InitiatePayment called: clientID=%d, advocateID=%d, amount=%.2f", clientID, advocateID, amount)

	if amount <= 0 {
		s.logger.Info("InitiatePayment failed: invalid amount: %.2f", amount)
		return nil, errors.New("invalid amount")
	}

	// Calculate commission and fees
	platformFee := amount * (s.cfg.PlatformCommission / 100.0)
	advocateCommission := amount - platformFee

	// Generate unique transaction ID
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		s.logger.Severe("InitiatePayment failed: random generation error: %v", err)
		return nil, errors.New("failed to generate transaction ID")
	}
	txnID := fmt.Sprintf("txn_%s", hex.EncodeToString(b))

	payment := &models.Payment{
		ClientID:           clientID,
		AdvocateID:         advocateID,
		Amount:             amount,
		Status:             "pending",
		TransactionID:      txnID,
		PaymentGateway:     "stripe",
		AdvocateCommission: advocateCommission,
		PlatformFee:        platformFee,
	}

	if err := s.db.Create(payment).Error; err != nil {
		s.logger.Severe("InitiatePayment failed: database error: %v", err)
		return nil, errors.New("failed to create payment")
	}

	if s.outboxService != nil {
		_ = s.outboxService.Enqueue(context.Background(), "payment", fmt.Sprintf("%d", payment.ID), "payment.initiated", map[string]interface{}{
			"payment_id":  payment.ID,
			"client_id":   payment.ClientID,
			"advocate_id": payment.AdvocateID,
			"amount":      payment.Amount,
		})
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

	if payment.Status == "completed" {
		s.logger.Info("VerifyPayment: transaction already completed: %s", txnID)
		return &payment, nil
	}

	// Use transaction to ensure data consistency
	err := s.db.Transaction(func(tx *gorm.DB) error {
		payment.Status = "completed"
		if err := tx.Save(&payment).Error; err != nil {
			return err
		}

		// Update advocate earnings
		if err := tx.Model(&models.Advocate{}).Where("id = ?", payment.AdvocateID).
			UpdateColumn("earnings", gorm.Expr("earnings + ?", payment.AdvocateCommission)).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		s.logger.Severe("VerifyPayment failed: transaction error: %v", err)
		return nil, errors.New("failed to complete payment")
	}

	if s.outboxService != nil {
		_ = s.outboxService.Enqueue(context.Background(), "payment", fmt.Sprintf("%d", payment.ID), "payment.completed", map[string]interface{}{
			"payment_id":  payment.ID,
			"client_id":   payment.ClientID,
			"advocate_id": payment.AdvocateID,
			"amount":      payment.Amount,
			"status":      payment.Status,
		})
	}

	s.logger.Info("Payment verified successfully: ID=%d, TransactionID=%s", payment.ID, payment.TransactionID)
	return &payment, nil
}
