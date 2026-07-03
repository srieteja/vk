package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"vk_backend/internal/config"
	"vk_backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	// ParameterizedQueries prevents query logs from interpolating bound
	// values (emails, tokens, etc.) into the printed SQL. IgnoreRecordNotFoundError
	// stops routine lookups (e.g. a failed login) from being logged as errors.
	gormLog := gormlogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
	})

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{Logger: gormLog, TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql db: %w", err)
	}

	if cfg.DBMaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetimeSeconds > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxLifetimeSeconds) * time.Second)
	}

	return db, nil
}

func InitSchema(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Advocate{},
		&models.Client{},
		&models.Call{},
		&models.Payment{},
		&models.Session{},
		&models.OutboxEvent{},
		&models.IdempotencyKey{},
	)
}
