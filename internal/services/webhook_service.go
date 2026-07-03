package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"
	"vk_backend/internal/outbox"
	"vk_backend/internal/payments"

	"gorm.io/gorm"
)

// WebhookService is the sole place a payment transitions to completed,
// failed, or refunded: gateway webhooks are the only source of truth for
// payment state, never a client-initiated call.
type WebhookService struct {
	db            *gorm.DB
	outboxService *outbox.Service
	provider      payments.Provider
	logger        *logger.Logger
}

func NewWebhookService(db *gorm.DB, outboxService *outbox.Service, provider payments.Provider) *WebhookService {
	return &WebhookService{
		db:            db,
		outboxService: outboxService,
		provider:      provider,
		logger:        logger.NewLogger("WebhookService", logger.INFO),
	}
}

// HandleWebhook verifies, dedupes, and applies a single gateway webhook
// delivery. A nil return means "acknowledge with 2xx" -- including cases
// where there's deliberately nothing to do (a duplicate delivery, an event
// type we don't act on, or an event for a payment we have no record of).
func (s *WebhookService) HandleWebhook(ctx context.Context, rawBody []byte, headers http.Header) error {
	event, err := s.provider.ParseWebhook(ctx, rawBody, headers)
	if err != nil {
		return err
	}

	record := &models.WebhookEvent{
		Provider:  s.provider.Name(),
		EventID:   event.EventID,
		EventType: event.EventType,
		Payload:   rawBody,
	}
	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			s.logger.Info("Duplicate webhook event ignored: provider=%s eventID=%s", s.provider.Name(), event.EventID)
			return nil
		}
		return fmt.Errorf("failed to record webhook event: %w", err)
	}

	if event.Status == payments.StatusPending {
		s.logger.Finer("Webhook event type not acted on: provider=%s type=%s", s.provider.Name(), event.EventType)
		return nil
	}

	var payment models.Payment
	if err := s.db.WithContext(ctx).
		Where("payment_gateway = ? AND gateway_payment_id = ?", s.provider.Name(), event.GatewayPaymentID).
		First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Severe("Webhook event for unknown payment: provider=%s gatewayPaymentID=%s", s.provider.Name(), event.GatewayPaymentID)
			return nil
		}
		return fmt.Errorf("database error looking up payment: %w", err)
	}

	switch event.Status {
	case payments.StatusCompleted:
		return s.handleCompleted(ctx, &payment, event)
	case payments.StatusFailed:
		return s.handleFailed(ctx, &payment)
	case payments.StatusRefunded:
		return s.handleRefunded(ctx, &payment, event)
	case payments.StatusDisputed:
		return s.handleDisputed(ctx, &payment)
	}
	return nil
}

// handleDisputed flags a payment for manual review on a chargeback. v1
// deliberately does not automate an earnings adjustment here (unlike a
// clean refund) -- dispute outcomes are decided by the gateway/card network
// and can go either way, so a human needs to resolve it.
func (s *WebhookService) handleDisputed(ctx context.Context, payment *models.Payment) error {
	s.logger.Severe("Payment disputed, flagging for manual review: paymentID=%d", payment.ID)
	return s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("id = ? AND status NOT IN ?", payment.ID, []string{"refunded", "flagged"}).
		Update("status", "flagged").Error
}

func (s *WebhookService) handleCompleted(ctx context.Context, payment *models.Payment, event *payments.WebhookEvent) error {
	if payment.AmountCents != event.AmountCents || payment.Currency != event.Currency {
		s.logger.Severe("Webhook amount/currency mismatch, flagging: paymentID=%d expected=%d %s got=%d %s",
			payment.ID, payment.AmountCents, payment.Currency, event.AmountCents, event.Currency)
		return s.db.WithContext(ctx).Model(&models.Payment{}).
			Where("id = ? AND status <> ?", payment.ID, "completed").
			Update("status", "flagged").Error
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Payment{}).
			Where("id = ? AND status <> ?", payment.ID, "completed").
			Update("status", "completed")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Already completed -- idempotent no-op.
			return nil
		}

		if err := tx.Model(&models.Advocate{}).Where("id = ?", payment.AdvocateID).
			UpdateColumn("earnings", gorm.Expr("earnings + ?", centsToUnits(payment.AdvocateCommissionCents))).Error; err != nil {
			return err
		}

		if s.outboxService != nil {
			return s.outboxService.EnqueueTx(tx, "payment", fmt.Sprintf("%d", payment.ID), "payment.completed", map[string]interface{}{
				"payment_id":  payment.ID,
				"client_id":   payment.ClientID,
				"advocate_id": payment.AdvocateID,
				"amount_cents": payment.AmountCents,
			})
		}
		return nil
	})
}

func (s *WebhookService) handleFailed(ctx context.Context, payment *models.Payment) error {
	return s.db.WithContext(ctx).Model(&models.Payment{}).
		Where("id = ? AND status NOT IN ?", payment.ID, []string{"completed", "failed", "refunded"}).
		Update("status", "failed").Error
}

func (s *WebhookService) handleRefunded(ctx context.Context, payment *models.Payment, event *payments.WebhookEvent) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Payment{}).
			Where("id = ? AND status = ?", payment.ID, "completed").
			Updates(map[string]interface{}{"status": "refunded", "refunded_amount_cents": event.AmountCents})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Not in a refundable state (already refunded, or never
			// completed) -- idempotent no-op.
			return nil
		}

		if err := tx.Model(&models.Advocate{}).Where("id = ?", payment.AdvocateID).
			UpdateColumn("earnings", gorm.Expr("earnings - ?", centsToUnits(payment.AdvocateCommissionCents))).Error; err != nil {
			return err
		}

		if s.outboxService != nil {
			return s.outboxService.EnqueueTx(tx, "payment", fmt.Sprintf("%d", payment.ID), "payment.refunded", map[string]interface{}{
				"payment_id":            payment.ID,
				"advocate_id":           payment.AdvocateID,
				"refunded_amount_cents": event.AmountCents,
			})
		}
		return nil
	})
}

// centsToUnits converts integer cents to the Advocate.Earnings float64
// column's major-unit representation (e.g. dollars).
func centsToUnits(cents int64) float64 {
	return float64(cents) / 100.0
}
