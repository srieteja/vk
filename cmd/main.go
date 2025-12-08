package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"enterprise-api/internal/config"
	"enterprise-api/internal/database"
	"enterprise-api/internal/logger"
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

	// Initialize logger
	logPriority := logger.ParsePriority(cfg.LogLevel)
	logger.Init("enterprise-api", logPriority)
	log := logger.GetLogger()

	log.Info("Initializing enterprise-api service")
	log.Finer("Environment: %s, Log Level: %s", cfg.Environment, cfg.LogLevel)

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
		log.Finer("Gin mode set to ReleaseMode")
	}

	// Initialize DB
	log.Finer("Connecting to database...")
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Severe("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	log.Info("Database connection established")

	// Run migrations
	log.Finer("Running database migrations...")
	if err := database.Migrate(db); err != nil {
		log.Severe("Failed to run migrations: %v", err)
		os.Exit(1)
	}
	log.Info("Database migrations completed")

	// Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		log.Finest("Health check requested")
		c.JSON(200, gin.H{"status": "ok", "service": "enterprise-api"})
	})

	// Initialize services
	authService := services.NewAuthService(db)
	advocateService := services.NewAdvocateService(db)
	callService := services.NewCallService(db)
	paymentService := services.NewPaymentService(db)
	webrtcService := services.NewWebRTCService(db, cfg.JWTSecret)

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
				log.Finer("Advocate registration request received")
				var req models.RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid registration request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := authService.RegisterAdvocate(&req)
				if err != nil {
					log.Info("Advocate registration failed: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				log.Info("Advocate registered successfully: ID=%d, Email=%s", advocate.ID, advocate.Email)
				c.JSON(200, advocate)
			})

			auth.POST("/advocate/login", func(c *gin.Context) {
				log.Finer("Advocate login request received")
				var req models.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid login request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := authService.LoginAdvocate(req.Email, req.Password)
				if err != nil {
					log.Info("Advocate login failed for email: %s", req.Email)
					c.JSON(401, gin.H{"error": err.Error()})
					return
				}

				// Create session
				session, err := authService.CreateSession(advocate.ID, "advocate")
				if err != nil {
					log.Severe("Failed to create session for advocate ID=%d: %v", advocate.ID, err)
					c.JSON(500, gin.H{"error": "failed to create session"})
					return
				}

				log.Info("Advocate logged in successfully: ID=%d, Email=%s", advocate.ID, advocate.Email)

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
				log.Finer("Client registration request received")
				var req models.RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid registration request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				client, err := authService.RegisterClient(&req)
				if err != nil {
					log.Info("Client registration failed: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				log.Info("Client registered successfully: ID=%d, Email=%s", client.ID, client.Email)
				c.JSON(200, client)
			})

			auth.POST("/client/login", func(c *gin.Context) {
				log.Finer("Client login request received")
				var req models.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid login request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				client, err := authService.LoginClient(req.Email, req.Password)
				if err != nil {
					log.Info("Client login failed for email: %s", req.Email)
					c.JSON(401, gin.H{"error": err.Error()})
					return
				}

				// Create session
				session, err := authService.CreateSession(client.ID, "client")
				if err != nil {
					log.Severe("Failed to create session for client ID=%d: %v", client.ID, err)
					c.JSON(500, gin.H{"error": "failed to create session"})
					return
				}

				log.Info("Client logged in successfully: ID=%d, Email=%s", client.ID, client.Email)

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
				userID, _ := c.Get("user_id")
				log.Finer("Call accept request received: userID=%d", userID)

				var req struct {
					CallID uint `json:"call_id" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid accept call request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				call, err := callService.AcceptCall(req.CallID, userID.(uint))
				if err != nil {
					log.Info("Call accept failed: callID=%d, error=%v", req.CallID, err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				log.Info("Call accepted successfully: ID=%d", call.ID)
				c.JSON(200, call)
			})

			calls.POST("/end", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				log.Finer("Call end request received: userID=%d", userID)

				var req struct {
					CallID uint `json:"call_id" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid end call request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				call, err := callService.EndCall(req.CallID, userID.(uint))
				if err != nil {
					log.Info("Call end failed: callID=%d, error=%v", req.CallID, err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				log.Info("Call ended successfully: ID=%d, Duration=%d seconds", call.ID, call.Duration)
				c.JSON(200, call)
			})

			calls.GET("/token", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				log.Finer("WebRTC token request received: userID=%d", userID)

				callIDStr := c.Query("call_id")
				if callIDStr == "" {
					log.Finer("WebRTC token request failed: missing call_id")
					c.JSON(400, gin.H{"error": "call_id is required"})
					return
				}

				callID, err := strconv.ParseUint(callIDStr, 10, 32)
				if err != nil {
					log.Finer("WebRTC token request failed: invalid call_id format: %s", callIDStr)
					c.JSON(400, gin.H{"error": "invalid call_id"})
					return
				}

				token, err := webrtcService.GenerateToken(callIDStr, userID.(uint))
				if err != nil {
					log.Info("WebRTC token generation failed: callID=%s, error=%v", callIDStr, err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				iceServers := webrtcService.GetICEServers()

				log.Info("WebRTC token generated successfully: callID=%s, userID=%d", callIDStr, userID)
				c.JSON(200, gin.H{
					"token":       token,
					"call_id":     callID,
					"expires_at":  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					"ice_servers": iceServers,
				})
			})
		}
	}

	// HTTP server
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		log.Info("Starting HTTP server on port :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Severe("Server error: %v", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Severe("Server shutdown error: %v", err)
	} else {
		log.Info("Server shutdown completed successfully")
	}
}
