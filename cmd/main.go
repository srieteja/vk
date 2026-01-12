package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"enterprise-api/internal/config"
	"enterprise-api/internal/database"
	"enterprise-api/internal/logger"
	"enterprise-api/internal/middleware"
	"enterprise-api/internal/models"
	"enterprise-api/internal/services"

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

	// Initialize LLM service
	log.Info("Initializing LLM service...")
	llmService := services.NewLLMService(db)
	llmService.Start(cfg)
	log.Info("LLM service initialized")

	// Initialize database schema
	log.Finer("Initializing database schema...")
	if err := database.InitSchema(db); err != nil {
		log.Severe("Failed to initialize schema: %v", err)
		os.Exit(1)
	}
	log.Info("Database schema initialized")

	// Gin router
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		log.Finest("Health check requested")
		c.JSON(200, gin.H{"status": "ok", "service": "enterprise-api"})
	})

	// Initialize services
	authService := services.NewAuthService(db)
	oauthService := services.NewOAuthService(db, cfg, authService)
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
				log.Finest("Request method: %s, Content-Type: %s, ContentLength: %d",
					c.Request.Method, c.GetHeader("Content-Type"), c.Request.ContentLength)

				// Check Content-Type header (be more lenient - allow charset variations)
				contentType := c.GetHeader("Content-Type")
				if contentType != "" && !strings.Contains(contentType, "application/json") {
					log.Info("Invalid Content-Type: %s", contentType)
					c.JSON(400, gin.H{
						"error":    "Content-Type must be application/json",
						"received": contentType,
					})
					return
				}

				var req models.RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					// Provide more specific error messages
					errorMsg := err.Error()
					if errorMsg == "EOF" {
						errorMsg = "request body is empty or malformed. Please ensure you're sending valid JSON with email, password, and name fields"
					}
					log.Info("Invalid registration request: %v, Content-Type: %s, ContentLength: %d",
						err, contentType, c.Request.ContentLength)
					c.JSON(400, gin.H{
						"error": errorMsg,
						"hint":  "Make sure you're using 'raw' body type with 'JSON' selected in Postman",
					})
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
					"uuid":          advocate.UUID,
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
					"uuid":          client.UUID,
					"profile_image": client.ProfileImage,
					"balance":       client.Balance,
					"created_at":    client.CreatedAt,
					"updated_at":    client.UpdatedAt,
					"token":         session.Token,
				}

				c.JSON(200, response)
			})

			// ---------------------------
			// Google OAuth routes
			// ---------------------------
			auth.GET("/google/advocate", func(c *gin.Context) {
				state := oauthService.GenerateState("advocate")
				url := oauthService.GetAuthURL(state)
				c.Redirect(http.StatusTemporaryRedirect, url)
			})

			auth.GET("/google/client", func(c *gin.Context) {
				state := oauthService.GenerateState("client")
				url := oauthService.GetAuthURL(state)
				c.Redirect(http.StatusTemporaryRedirect, url)
			})

			auth.GET("/google/callback", func(c *gin.Context) {
				code := c.Query("code")
				state := c.Query("state")

				if code == "" {
					c.JSON(400, gin.H{"error": "missing authorization code"})
					return
				}

				// Extract user type from state
				var userType string
				if strings.HasPrefix(state, "advocate_") {
					userType = "advocate"
				} else if strings.HasPrefix(state, "client_") {
					userType = "client"
				} else {
					c.JSON(400, gin.H{"error": "invalid state"})
					return
				}

				session, err := oauthService.HandleCallback(code, userType)
				if err != nil {
					log.Info("OAuth callback failed: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				// Get user info to return
				var userData gin.H
				if userType == "advocate" {
					var advocate models.Advocate
					if err := db.First(&advocate, session.UserID).Error; err == nil {
						userData = gin.H{
							"id":            advocate.ID,
							"email":         advocate.Email,
							"name":          advocate.Name,
							"availability":  advocate.Availability,
							"uuid":          advocate.UUID,
							"profile_image": advocate.ProfileImage,
							"bio":           advocate.Bio,
							"earnings":      advocate.Earnings,
							"hourly_rate":   advocate.HourlyRate,
							"created_at":    advocate.CreatedAt,
							"updated_at":    advocate.UpdatedAt,
							"token":         session.Token,
						}
					}
				} else {
					var client models.Client
					if err := db.First(&client, session.UserID).Error; err == nil {
						userData = gin.H{
							"id":            client.ID,
							"email":         client.Email,
							"name":          client.Name,
							"uuid":          client.UUID,
							"profile_image": client.ProfileImage,
							"balance":       client.Balance,
							"created_at":    client.CreatedAt,
							"updated_at":    client.UpdatedAt,
							"token":         session.Token,
						}
					}
				}

				log.Info("OAuth login successful: UserType=%s, UserID=%d", userType, session.UserID)
				c.JSON(200, userData)
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

			advocate.PUT("/profile-image", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userType, _ := c.Get("user_type")

				if userType != "advocate" {
					c.JSON(403, gin.H{"error": "access denied"})
					return
				}

				var req struct {
					ProfileImage string `json:"profile_image" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := advocateService.UpdateProfileImage(userID.(uint), req.ProfileImage)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, advocate)
			})
		}

		// ---------------------------
		// Client (Consumer) routes
		// ---------------------------
		client := api.Group("/client")
		client.Use(authMiddleware)
		{
			client.PUT("/profile-image", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				userType, _ := c.Get("user_type")

				if userType != "client" {
					c.JSON(403, gin.H{"error": "access denied"})
					return
				}

				var req struct {
					ProfileImage string `json:"profile_image" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				client, err := authService.UpdateClientProfileImage(userID.(uint), req.ProfileImage)
				if err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, client)
			})

			client.GET("/available-users", func(c *gin.Context) {
				// Parse query parameters
				filter := &services.AdvocateFilter{
					Availability: c.Query("availability"), // "all" or "available"/"online"
					Location:     c.Query("location"),
				}

				// Parse rate range
				if minRateStr := c.Query("min_rate"); minRateStr != "" {
					if minRate, err := strconv.ParseFloat(minRateStr, 64); err == nil {
						filter.MinRate = minRate
					}
				}
				if maxRateStr := c.Query("max_rate"); maxRateStr != "" {
					if maxRate, err := strconv.ParseFloat(maxRateStr, 64); err == nil {
						filter.MaxRate = maxRate
					}
				}

				advocates, err := advocateService.GetAvailableAdvocates(filter)
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
						"location":      a.Location,
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

				// Convert parsed callID to uint for type safety
				callIDUint := uint(callID)
				token, err := webrtcService.GenerateToken(callIDUint, userID.(uint))
				if err != nil {
					log.Info("WebRTC token generation failed: callID=%d, error=%v", callIDUint, err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				iceServers := webrtcService.GetICEServers()

				log.Info("WebRTC token generated successfully: callID=%d, userID=%d", callIDUint, userID)
				c.JSON(200, gin.H{
					"token":       token,
					"call_id":     callIDUint,
					"expires_at":  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					"ice_servers": iceServers,
				})
			})
		}
	}

	// LLM endpoints
	llmGroup := api.Group("/llm")
	{
		// Chat endpoint (protected by auth)
		llmGroup.POST("/chat", authMiddleware, func(c *gin.Context) {
			services.HandleChat(c.Writer, c.Request)
		})

		// Health check endpoint (public)
		llmGroup.GET("/health", func(c *gin.Context) {
			handleHealth := http.HandlerFunc(services.HandleHealth)
			handleHealth.ServeHTTP(c.Writer, c.Request)
		})
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
