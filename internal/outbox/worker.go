package outbox

import (
	"context"
	"errors"
	"time"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Handler func(context.Context, *models.OutboxEvent) error

type Worker struct {
	db          *gorm.DB
	logger      *logger.Logger
	handlers    map[string]Handler
	batchSize   int
	maxAttempts int
}

func NewWorker(db *gorm.DB) *Worker {
	return &Worker{
		db:          db,
		logger:      logger.NewLogger("OutboxWorker", logger.INFO),
		handlers:    make(map[string]Handler),
		batchSize:   10,
		maxAttempts: 5,
	}
}

func (w *Worker) Register(eventType string, handler Handler) {
	w.handlers[eventType] = handler
}

func (w *Worker) SetBatchSize(size int) {
	if size > 0 {
		w.batchSize = size
	}
}

func (w *Worker) SetMaxAttempts(maxAttempts int) {
	if maxAttempts > 0 {
		w.maxAttempts = maxAttempts
	}
}

func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.logger.Info("Outbox worker error: %v", err)
			}
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) error {
	events, err := w.lockPending(ctx)
	if err != nil {
		return err
	}

	for _, event := range events {
		handler, ok := w.handlers[event.EventType]
		if !ok {
			w.logger.Info("No handler for event type: %s", event.EventType)
			w.markFailed(ctx, &event, errors.New("no handler registered"))
			continue
		}

		if err := handler(ctx, &event); err != nil {
			w.logger.Info("Outbox handler error: %v", err)
			w.markRetry(ctx, &event, err)
			continue
		}

		w.markCompleted(ctx, &event)
	}

	return nil
}

func (w *Worker) lockPending(ctx context.Context) ([]models.OutboxEvent, error) {
	var events []models.OutboxEvent
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", "pending").
			Order("id").
			Limit(w.batchSize).
			Find(&events).Error; err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		var ids []uint
		for i := range events {
			events[i].Attempts++
			ids = append(ids, events[i].ID)
		}

		return tx.Model(&models.OutboxEvent{}).
			Where("id IN ?", ids).
			UpdateColumns(map[string]interface{}{
				"status":   "processing",
				"attempts": gorm.Expr("attempts + 1"),
			}).Error
	})
	return events, err
}

func (w *Worker) markCompleted(ctx context.Context, event *models.OutboxEvent) {
	now := time.Now()
	_ = w.db.WithContext(ctx).Model(event).Updates(map[string]interface{}{
		"status":       "completed",
		"processed_at": &now,
		"last_error":   "",
	}).Error
}

func (w *Worker) markRetry(ctx context.Context, event *models.OutboxEvent, err error) {
	status := "pending"
	if event.Attempts >= w.maxAttempts {
		status = "failed"
	}
	_ = w.db.WithContext(ctx).Model(event).Updates(map[string]interface{}{
		"status":     status,
		"last_error": err.Error(),
	}).Error
}

func (w *Worker) markFailed(ctx context.Context, event *models.OutboxEvent, err error) {
	_ = w.db.WithContext(ctx).Model(event).Updates(map[string]interface{}{
		"status":     "failed",
		"last_error": err.Error(),
	}).Error
}
