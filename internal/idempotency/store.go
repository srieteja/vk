package idempotency

import (
	"context"
	"errors"
	"time"

	"vk_backend/internal/models"

	"gorm.io/gorm"
)

var ErrKeyNotFound = errors.New("idempotency key not found")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Get(ctx context.Context, key string) (*models.IdempotencyKey, error) {
	var record models.IdempotencyKey
	if err := s.db.WithContext(ctx).First(&record, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (s *Store) Save(ctx context.Context, record *models.IdempotencyKey) error {
	return s.db.WithContext(ctx).Create(record).Error
}

func (s *Store) DeleteExpired(ctx context.Context) error {
	return s.db.WithContext(ctx).
		Where("expires_at <= ?", time.Now()).
		Delete(&models.IdempotencyKey{}).Error
}
