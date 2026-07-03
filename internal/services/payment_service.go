package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"vk_backend/internal/config"
	"vk_backend/internal/logger"
	"vk_backend/internal/models"
	"vk_backend/internal/outbox"
	"vk_backend/internal/payments"

	"gorm.io/gorm"
)

type PaymentService struct {
	db            *gorm.DB
	cfg           *config.Config
	outboxService *outbox.Service
	provider      payments.Provider
	logger        *logger.Logger
}

func NewPaymentService(db *gorm.DB, cfg *config.Config, outboxService *outbox.Service, provider payments.Provider) *PaymentService {
	return &PaymentService{
		db:            db,
		cfg:           cfg,
		outboxService: outboxService,
		provider:      provider,
		logger:        logger.NewLogger("PaymentService", logger.INFO),
	}
}

// InitiatePayment computes the charge amount server-side from the
// advocate's own rate (never trusting a client-supplied amount), persists a
// requires_payment row in the same DB transaction as its outbox event, then
// asks the gateway to create a payment intent. The gateway call happens
// after that transaction commits, since it's an external network call that
// shouldn't hold a DB transaction open.
func (s *PaymentService) InitiatePayment(ctx context.Context, clientID, advocateID uint, durationMinutes int) (*models.Payment, error) {
	s.logger.Finer("InitiatePayment called: clientID=%d, advocateID=%d, durationMinutes=%d", clientID, advocateID, durationMinutes)

	if durationMinutes <= 0 {
		s.logger.Info("InitiatePayment failed: invalid duration: %d", durationMinutes)
		return nil, errors.New("duration_minutes must be positive")
	}

	var advocate models.Advocate
	if err := s.db.First(&advocate, advocateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("InitiatePayment failed: advocate not found: %d", advocateID)
			return nil, errors.New("advocate not found")
		}
		s.logger.Severe("InitiatePayment failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	amountCents := ratePerMinuteCents(advocate, s.cfg.UserBRatePerMinute) * int64(durationMinutes)
	if amountCents <= 0 {
		s.logger.Info("InitiatePayment failed: computed amount is non-positive: %d", amountCents)
		return nil, errors.New("computed amount must be positive")
	}
	platformFeeCents := int64(math.Round(float64(amountCents) * s.cfg.PlatformCommission / 100.0))
	advocateCommissionCents := amountCents - platformFeeCents

	txnID, err := generateTransactionID()
	if err != nil {
		s.logger.Severe("InitiatePayment failed: transaction id generation error: %v", err)
		return nil, errors.New("failed to generate transaction ID")
	}

	payment := &models.Payment{
		ClientID:                clientID,
		AdvocateID:              advocateID,
		AmountCents:             amountCents,
		Currency:                s.cfg.PaymentCurrency,
		DurationMinutes:         durationMinutes,
		Status:                  "requires_payment",
		TransactionID:           txnID,
		PaymentGateway:          s.provider.Name(),
		AdvocateCommissionCents: advocateCommissionCents,
		PlatformFeeCents:        platformFeeCents,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(payment).Error; err != nil {
			return err
		}
		if s.outboxService != nil {
			return s.outboxService.EnqueueTx(tx, "payment", fmt.Sprintf("%d", payment.ID), "payment.initiated", map[string]interface{}{
				"payment_id":  payment.ID,
				"client_id":   payment.ClientID,
				"advocate_id": payment.AdvocateID,
				"amount_cents": payment.AmountCents,
				"currency":    payment.Currency,
			})
		}
		return nil
	})
	if err != nil {
		s.logger.Severe("InitiatePayment failed: database error: %v", err)
		return nil, errors.New("failed to create payment")
	}

	intent, err := s.provider.CreateIntent(ctx, payments.CreateIntentParams{
		AmountCents: amountCents,
		Currency:    payment.Currency,
		Reference:   txnID,
		Metadata:    map[string]string{"payment_id": fmt.Sprintf("%d", payment.ID)},
	})
	if err != nil {
		s.logger.Severe("InitiatePayment failed: gateway CreateIntent error: %v", err)
		if updateErr := s.db.Model(&models.Payment{}).Where("id = ?", payment.ID).Update("status", "failed").Error; updateErr != nil {
			s.logger.Severe("InitiatePayment: also failed to mark payment failed: %v", updateErr)
		}
		return nil, errors.New("failed to initiate payment with gateway")
	}

	payment.GatewayPaymentID = intent.GatewayPaymentID
	payment.Status = "processing"
	if err := s.db.Model(&models.Payment{}).Where("id = ?", payment.ID).
		Updates(map[string]interface{}{
			"gateway_payment_id": payment.GatewayPaymentID,
			"status":             payment.Status,
		}).Error; err != nil {
		s.logger.Severe("InitiatePayment failed: failed to record gateway payment id: %v", err)
		return nil, errors.New("failed to record gateway payment id")
	}

	payment.ClientSecret = intent.ClientSecret

	s.logger.Info("Payment initiated successfully: ID=%d, TransactionID=%s, AmountCents=%d", payment.ID, payment.TransactionID, payment.AmountCents)
	return payment, nil
}

// GetPaymentStatus is a read-only lookup -- it never mutates payment state
// or credits earnings. Completion is exclusively webhook-driven (see
// WebhookService); this used to be VerifyPayment, which trusted an
// unauthenticated client call to mark a payment complete and credit the
// advocate, with no gateway confirmation at all.
func (s *PaymentService) GetPaymentStatus(txnID string, requestingClientID uint) (*models.Payment, error) {
	s.logger.Finer("GetPaymentStatus called: transactionID=%s", txnID)

	if txnID == "" {
		s.logger.Info("GetPaymentStatus failed: empty transaction ID")
		return nil, errors.New("invalid transaction ID")
	}

	var payment models.Payment
	if err := s.db.Where("transaction_id = ?", txnID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			s.logger.Info("GetPaymentStatus failed: transaction not found: %s", txnID)
			return nil, errors.New("transaction not found")
		}
		s.logger.Severe("GetPaymentStatus failed: database error: %v", err)
		return nil, errors.New("database error")
	}

	// Same "not found" error for a real-but-not-yours transaction as for a
	// nonexistent one -- avoids leaking existence via a distinct response.
	if payment.ClientID != requestingClientID {
		s.logger.Info("GetPaymentStatus failed: transaction not owned by requester: txnID=%s, ownerID=%d, requesterID=%d", txnID, payment.ClientID, requestingClientID)
		return nil, errors.New("transaction not found")
	}

	return &payment, nil
}

// RefundPayment triggers a full refund at the gateway for a completed
// payment. It does not mark the payment refunded or debit the advocate's
// earnings itself -- that's exclusively done by WebhookService once the
// gateway confirms the refund, matching how completion works.
func (s *PaymentService) RefundPayment(ctx context.Context, txnID string) error {
	s.logger.Finer("RefundPayment called: transactionID=%s", txnID)

	var payment models.Payment
	if err := s.db.WithContext(ctx).Where("transaction_id = ?", txnID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("transaction not found")
		}
		s.logger.Severe("RefundPayment failed: database error: %v", err)
		return errors.New("database error")
	}

	if payment.Status != "completed" {
		return fmt.Errorf("only completed payments can be refunded (status is %q)", payment.Status)
	}

	// v1 only supports refunding through whichever provider is currently
	// active. A payment made through a since-deactivated provider can't be
	// refunded here -- fail clearly rather than misrouting the refund call.
	if payment.PaymentGateway != s.provider.Name() {
		return fmt.Errorf("payment was made via %q, but the active payment provider is %q; cannot refund", payment.PaymentGateway, s.provider.Name())
	}

	if _, err := s.provider.CreateRefund(ctx, payments.RefundParams{
		GatewayPaymentID: payment.GatewayPaymentID,
		AmountCents:      payment.AmountCents, // v1: full refunds only
		Reason:           "admin requested",
	}); err != nil {
		s.logger.Severe("RefundPayment failed: gateway CreateRefund error: %v", err)
		return errors.New("failed to create refund with gateway")
	}

	s.logger.Info("Refund requested at gateway: paymentID=%d, transactionID=%s", payment.ID, txnID)
	return nil
}

// ratePerMinuteCents derives a per-minute rate in integer cents from the
// advocate's own hourly rate, falling back to the platform default when the
// advocate hasn't set one.
func ratePerMinuteCents(advocate models.Advocate, fallbackPerMinute float64) int64 {
	if advocate.HourlyRate > 0 {
		return int64(math.Round(advocate.HourlyRate / 60.0 * 100))
	}
	return int64(math.Round(fallbackPerMinute * 100))
}

func generateTransactionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("txn_%s", hex.EncodeToString(b)), nil
}
