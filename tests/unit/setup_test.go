package unit

import (
	"vk_backend/internal/models"
	"vk_backend/internal/sessions"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
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
		&models.OutboxEvent{},
		&models.IdempotencyKey{},
		&models.WebhookEvent{},
	)
	if err != nil {
		panic("failed to migrate schema")
	}

	// SQLite ":memory:" is scoped to a single connection: a second pooled
	// connection would see a fresh, unmigrated database. Pin the pool to one
	// connection so goroutine-spawned queries (e.g. the WebSocket handler in
	// signaling tests) share the same in-memory database as the setup above.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	return db
}

func setupSessionStore(db *gorm.DB) sessions.Store {
	store, _, err := sessions.NewStore(db, nil, "db")
	if err != nil {
		panic("failed to initialize session store")
	}
	return store
}
