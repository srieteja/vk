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
	// Load .env file (optional)
	godotenv.Load()

	// Load config
	cfg := config.LoadConfig()

	// Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Router
	router := gin.New()
	router.Use(gin.Recovery())

	// 🔥 Log every incoming request
	router.Use(func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()

		c.Next()

		status := c.Writer.Status()
		log.Printf("%s %s | %d | %s | %v", method, path, status, ip, time.Since(start))
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "enterprise-api"})
	})

	// API routes
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/userA/register", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Advocate register endpoint"})
			})
			auth.POST("/userA/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Advocate login endpoint"})
			})
			auth.POST("/userB/register", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Client register endpoint"})
			})
			auth.POST("/userB/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "Client login endpoint"})
			})
		}
	}

	// 🔥 Make sure we bind to ALL interfaces (required for Oracle VM)
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: router,
	}

	// Start server
	go func() {
		log.Printf("🚀 Server running on http://0.0.0.0:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✔ Server exited properly.")
}
