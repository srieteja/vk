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
	"enterprise-api/internal/middleware"
	"enterprise-api/internal/models"
	"enterprise-api/internal/services"
	"strconv"

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

	// Initialize services
	authService := services.NewAuthService(db)
	advocateService := services.NewAdvocateService(db)
	callService := services.NewCallService(db)
	paymentService := services.NewPaymentService(db)
	webrtcService := services.NewWebRTCService()

	// Initialize middleware
	authMiddleware := middleware.AuthMiddleware(db)

	// API routes
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			// ---------------------------
			// Advocate (Provider) routes
			// ---------------------------
			auth.POST("/advocate/register", func(c *gin.Context) {
				var req models.RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := authService.RegisterAdvocate(&req)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, advocate)
			})

			auth.POST("/advocate/login", func(c *gin.Context) {
				var req models.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := authService.LoginAdvocate(req.Email, req.Password)
				if err != nil {
					c.JSON(401, gin.H{"error": err.Error()})
					return
				}

				// Create session
				session, err := authService.CreateSession(advocate.ID, "advocate")
				if err != nil {
					c.JSON(500, gin.H{"error": "failed to create session"})
					return
				}

				// Return advocate with token
				response := gin.H{
					"id":            advocate.ID,
					"email":         advocate.Email,
					"name":          advocate.Name,
					"availability":  advocate.Availability,
					"google_id":     advocate.GoogleID,
					"profile_image": advocate.ProfileImage,
					"bio":           advocate.Bio,
					"earnings":      advocate.Earnings,
					"hourly_rate":   advocate.HourlyRate,
					"created_at":    advocate.CreatedAt,
					"updated_at":    advocate.UpdatedAt,
					"token":         session.Token,
				}

				c.JSON(200, response)
			})

			// ---------------------------
			// Client (Consumer) routes
			// ---------------------------
			auth.POST("/client/register", func(c *gin.Context) {
				var req models.RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				client, err := authService.RegisterClient(&req)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, client)
			})

			auth.POST("/client/login", func(c *gin.Context) {
				var req models.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				client, err := authService.LoginClient(req.Email, req.Password)
				if err != nil {
					c.JSON(401, gin.H{"error": err.Error()})
					return
				}

				// Create session
				session, err := authService.CreateSession(client.ID, "client")
				if err != nil {
					c.JSON(500, gin.H{"error": "failed to create session"})
					return
				}

				// Return client with token
				response := gin.H{
					"id":            client.ID,
					"email":         client.Email,
					"name":          client.Name,
					"google_id":     client.GoogleID,
					"profile_image": client.ProfileImage,
					"balance":       client.Balance,
					"created_at":    client.CreatedAt,
					"updated_at":    client.UpdatedAt,
					"token":         session.Token,
				}

				c.JSON(200, response)
			})
		}

		// ---------------------------
		// Advocate (Provider) routes
		// ---------------------------
		advocate := api.Group("/advocate")
		advocate.Use(authMiddleware)
		{
			advocate.PUT("/availability", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userType, _ := c.Get("user_type")

				if userType != "advocate" {
					c.JSON(403, gin.H{"error": "access denied"})
					return
				}

				var req struct {
					Availability string `json:"availability" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := advocateService.UpdateAvailability(userID.(uint), req.Availability)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, advocate)
			})

			advocate.GET("/profile", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userType, _ := c.Get("user_type")

				if userType != "advocate" {
					c.JSON(403, gin.H{"error": "access denied"})
					return
				}

				advocate, err := advocateService.GetProfile(userID.(uint))
				if err != nil {
					c.JSON(404, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, advocate)
			})

			advocate.GET("/earnings", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userType, _ := c.Get("user_type")

				if userType != "advocate" {
					c.JSON(403, gin.H{"error": "access denied"})
					return
				}

				earnings, err := advocateService.GetEarnings(userID.(uint))
				if err != nil {
					c.JSON(404, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, earnings)
			})
		}

		// ---------------------------
		// Client (Consumer) routes
		// ---------------------------
		client := api.Group("/client")
		client.Use(authMiddleware)
		{
			client.GET("/available-users", func(c *gin.Context) {
				advocates, err := advocateService.GetAvailableAdvocates()
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}

				// Filter out sensitive data
				var users []gin.H
				for _, a := range advocates {
					users = append(users, gin.H{
						"id":            a.ID,
						"email":         a.Email,
						"name":          a.Name,
						"availability":  a.Availability,
						"profile_image": a.ProfileImage,
						"bio":           a.Bio,
						"hourly_rate":   a.HourlyRate,
					})
				}

				c.JSON(200, gin.H{
					"users": users,
					"count": len(users),
				})
			})

			payment := client.Group("/payment")
			{
				payment.POST("/initiate", func(c *gin.Context) {
					userID, _ := c.Get("user_id")
					userType, _ := c.Get("user_type")

					if userType != "client" {
						c.JSON(403, gin.H{"error": "access denied"})
						return
					}

					var req struct {
						AdvocateID uint    `json:"advocate_id" binding:"required"`
						Amount     float64 `json:"amount" binding:"required"`
					}
					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					payment, err := paymentService.InitiatePayment(userID.(uint), req.AdvocateID, req.Amount)
					if err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					c.JSON(200, payment)
				})

				payment.POST("/verify", func(c *gin.Context) {
					userID, _ := c.Get("user_id")
					userType, _ := c.Get("user_type")

					if userType != "client" {
						c.JSON(403, gin.H{"error": "access denied"})
						return
					}

					var req struct {
						TransactionID string `json:"transaction_id" binding:"required"`
					}
					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					payment, err := paymentService.VerifyPayment(req.TransactionID)
					if err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
						return
					}

					// Verify the payment belongs to this client
					if payment.ClientID != userID.(uint) {
						c.JSON(403, gin.H{"error": "access denied"})
						return
					}

					c.JSON(200, payment)
				})
			}
		}

		// ---------------------------
		// Call routes
		// ---------------------------
		calls := api.Group("/calls")
		calls.Use(authMiddleware)
		{
			calls.POST("/initiate", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userType, _ := c.Get("user_type")

				var req struct {
					ReceiverID uint `json:"receiver_id" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				// Determine receiver type (opposite of caller)
				receiverType := "advocate"
				if userType == "advocate" {
					receiverType = "client"
				}

				call, err := callService.InitiateCall(userID.(uint), userType.(string), req.ReceiverID, receiverType)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, call)
			})

			calls.POST("/accept", func(c *gin.Context) {
				var req struct {
					CallID uint `json:"call_id" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				call, err := callService.AcceptCall(req.CallID)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, call)
			})

			calls.POST("/end", func(c *gin.Context) {
				var req struct {
					CallID uint `json:"call_id" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				call, err := callService.EndCall(req.CallID)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, call)
			})

			calls.GET("/token", func(c *gin.Context) {
				callIDStr := c.Query("call_id")
				if callIDStr == "" {
					c.JSON(400, gin.H{"error": "call_id is required"})
					return
				}

				callID, err := strconv.ParseUint(callIDStr, 10, 32)
				if err != nil {
					c.JSON(400, gin.H{"error": "invalid call_id"})
					return
				}

				token, err := webrtcService.GenerateToken(callIDStr)
				if err != nil {
					c.JSON(500, gin.H{"error": "failed to generate token"})
					return
				}

				c.JSON(200, gin.H{
					"token":      token,
					"call_id":    callID,
					"expires_at": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				})
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
