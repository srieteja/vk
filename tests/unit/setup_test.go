package unit

import (
	"vk_backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database")
	}

	// Migrate the schema
	err = db.AutoMigrate(
		&models.Advocate{},
		&models.Client{},
		&models.Call{},
		&models.Payment{},
		&models.Session{},
	)
	if err != nil {
		panic("failed to migrate schema")
	}

	return db
}
