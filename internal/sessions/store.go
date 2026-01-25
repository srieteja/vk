package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var ErrSessionNotFound = errors.New("session not found")

type Store interface {
	Create(ctx context.Context, session *models.Session) error
	GetByToken(ctx context.Context, token string) (*models.Session, error)
}

type dbStore struct {
	db *gorm.DB
}

func (s *dbStore) Create(ctx context.Context, session *models.Session) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *dbStore) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	var session models.Session
	err := s.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return &session, nil
}

type redisStore struct {
	client *redis.Client
	prefix string
}

type redisSession struct {
	UserID    uint   `json:"user_id"`
	UserType  string `json:"user_type"`
	ExpiresAt int64  `json:"expires_at"`
}

func (s *redisStore) key(token string) string {
	return fmt.Sprintf("%s%s", s.prefix, token)
}

func (s *redisStore) Create(ctx context.Context, session *models.Session) error {
	payload := redisSession{
		UserID:    session.UserID,
		UserType:  session.UserType,
		ExpiresAt: session.ExpiresAt.Unix(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return ErrSessionNotFound
	}

	return s.client.Set(ctx, s.key(session.Token), data, ttl).Err()
}

func (s *redisStore) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	value, err := s.client.Get(ctx, s.key(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	var payload redisSession
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, err
	}

	expiresAt := time.Unix(payload.ExpiresAt, 0)
	if time.Now().After(expiresAt) {
		return nil, ErrSessionNotFound
	}

	return &models.Session{
		UserID:    payload.UserID,
		UserType:  payload.UserType,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

type hybridStore struct {
	db     *gorm.DB
	redis  *redisStore
	logger *logger.Logger
}

func (s *hybridStore) Create(ctx context.Context, session *models.Session) error {
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return err
	}

	if s.redis != nil {
		if err := s.redis.Create(ctx, session); err != nil {
			s.logger.Info("Failed to cache session in redis: %v", err)
		}
	}

	return nil
}

func (s *hybridStore) GetByToken(ctx context.Context, token string) (*models.Session, error) {
	if s.redis != nil {
		session, err := s.redis.GetByToken(ctx, token)
		if err == nil {
			return session, nil
		}
		if err != ErrSessionNotFound {
			s.logger.Info("Redis session lookup failed: %v", err)
		}
	}

	var session models.Session
	err := s.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if s.redis != nil {
		if err := s.redis.Create(ctx, &session); err != nil {
			s.logger.Info("Failed to cache session in redis: %v", err)
		}
	}

	return &session, nil
}

func NewStore(db *gorm.DB, redisClient *redis.Client, mode string) (Store, string, error) {
	switch mode {
	case "redis":
		if redisClient == nil {
			return nil, "", errors.New("redis session store requested but redis is unavailable")
		}
		return &redisStore{client: redisClient, prefix: "session:"}, "redis", nil
	case "db":
		if db == nil {
			return nil, "", errors.New("db session store requested but db is unavailable")
		}
		return &dbStore{db: db}, "db", nil
	case "hybrid":
		if db == nil {
			return nil, "", errors.New("hybrid session store requested but db is unavailable")
		}
		if redisClient == nil {
			return &dbStore{db: db}, "db", nil
		}
		return &hybridStore{
			db:     db,
			redis:  &redisStore{client: redisClient, prefix: "session:"},
			logger: logger.NewLogger("SessionStore", logger.INFO),
		}, "hybrid", nil
	default:
		return NewStore(db, redisClient, "hybrid")
	}
}
