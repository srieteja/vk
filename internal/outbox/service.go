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

	if err := s.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to enqueue outbox event: %w", err)
	}

	return nil
}
