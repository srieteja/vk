package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"enterprise-api/internal/config"
	"enterprise-api/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	cfg := config.LoadConfig()
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "enterprise-api"})
	})

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/userA/register", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "UserA register endpoint"})
			})
			auth.POST("/userA/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "UserA login endpoint"})
			})
			auth.POST("/userB/register", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "UserB register endpoint"})
			})
			auth.POST("/userB/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "UserB login endpoint"})
			})
		}
	}

	server := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}