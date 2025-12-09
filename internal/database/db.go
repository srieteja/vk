package database

import (
	"fmt"

	"enterprise-api/internal/config"
	"enterprise-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
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
	)
}