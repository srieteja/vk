package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"

	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *logger.Logger
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db:     db,
		logger: logger.NewLogger("OutboxService", logger.INFO),
	}
}

func (s *Service) Enqueue(ctx context.Context, aggregateType string, aggregateID string, eventType string, payload interface{}) error {
	return enqueue(s.db.WithContext(ctx), aggregateType, aggregateID, eventType, payload)
}

// EnqueueTx enqueues an event using the given transaction, so the outbox
// insert commits atomically with whatever business-state change the caller
// is making in the same tx (e.g. creating a Payment row) -- avoiding the
// dual-write problem where one write could succeed without the other.
func (s *Service) EnqueueTx(tx *gorm.DB, aggregateType string, aggregateID string, eventType string, payload interface{}) error {
	return enqueue(tx, aggregateType, aggregateID, eventType, payload)
}

func enqueue(db *gorm.DB, aggregateType string, aggregateID string, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	event := &models.OutboxEvent{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       data,
		Status:        "pending",
		Attempts:      0,
	}

	if err := db.Create(event).Error; err != nil {
		return fmt.Errorf("failed to enqueue outbox event: %w", err)
	}

	return nil
}
