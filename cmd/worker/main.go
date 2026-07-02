package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vk_backend/internal/config"
	"vk_backend/internal/database"
	"vk_backend/internal/logger"
	"vk_backend/internal/models"
	"vk_backend/internal/outbox"

	"github.com/joho/godotenv"
)

func main() {
	envErr := godotenv.Load()

	cfg := config.LoadConfig()

	logPriority := logger.ParsePriority(cfg.LogLevel)
	logger.Init("vk_backend_worker", logPriority)
	log := logger.GetLogger()
	if envErr != nil {
		log.Info("Failed to load .env: %v", envErr)
	}

	if err := cfg.Validate(); err != nil {
		log.Severe("Invalid configuration: %v", err)
		os.Exit(1)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Severe("Failed to connect to database: %v", err)
		os.Exit(1)
	}

	if cfg.RunMigrations {
		if err := database.ApplyMigrations(db, cfg.MigrationsDir); err != nil {
			log.Severe("Failed to apply migrations: %v", err)
			os.Exit(1)
		}
	}

	worker := outbox.NewWorker(db)
	worker.SetBatchSize(cfg.OutboxBatchSize)
	worker.SetMaxAttempts(cfg.OutboxMaxAttempts)
	worker.Register("payment.initiated", func(ctx context.Context, event *models.OutboxEvent) error {
		log.Info("Processed event: %s (%d)", event.EventType, event.ID)
		return nil
	})
	worker.Register("payment.completed", func(ctx context.Context, event *models.OutboxEvent) error {
		log.Info("Processed event: %s (%d)", event.EventType, event.ID)
		return nil
	})
	worker.Register("call.initiated", func(ctx context.Context, event *models.OutboxEvent) error {
		log.Info("Processed event: %s (%d)", event.EventType, event.ID)
		return nil
	})
	worker.Register("call.accepted", func(ctx context.Context, event *models.OutboxEvent) error {
		log.Info("Processed event: %s (%d)", event.EventType, event.ID)
		return nil
	})
	worker.Register("call.completed", func(ctx context.Context, event *models.OutboxEvent) error {
		log.Info("Processed event: %s (%d)", event.EventType, event.ID)
		return nil
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pollInterval := time.Duration(cfg.OutboxPollIntervalSeconds) * time.Second
	log.Info("Starting outbox worker with poll interval %s", pollInterval)
	worker.Run(ctx, pollInterval)
}
