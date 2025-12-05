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
	// Load environment variables
	godotenv.Load()

	cfg := config.LoadConfig()
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize DB
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "enterprise-api"})
	})

	// API routes
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			// ---------------------------
			// Advocate routes
			// ---------------------------
			auth.POST("/advocate/register", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Advocate register endpoint"})
			})
			auth.POST("/advocate/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Advocate login endpoint"})
			})

			// ---------------------------
			// Client routes
			// ---------------------------
			auth.POST("/client/register", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Client register endpoint"})
			})
			auth.POST("/client/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Client login endpoint"})
			})
		}
	}

	// HTTP server
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
