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

	"vk_backend/internal/config"
	"vk_backend/internal/database"
	"vk_backend/internal/idempotency"
	"vk_backend/internal/infra"
	piilog "vk_backend/internal/log"
	"vk_backend/internal/logger"
	"vk_backend/internal/metrics"
	"vk_backend/internal/middleware"
	"vk_backend/internal/models"
	"vk_backend/internal/observability"
	"vk_backend/internal/outbox"
	"vk_backend/internal/services"
	"vk_backend/internal/sessions"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	limiter "github.com/ulule/limiter/v3"
	limitermemory "github.com/ulule/limiter/v3/drivers/store/memory"
	limiterredis "github.com/ulule/limiter/v3/drivers/store/redis"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	// Load environment variables
	envErr := godotenv.Load()

	cfg := config.LoadConfig()

	// Initialize logger
	logPriority := logger.ParsePriority(cfg.LogLevel)
	logger.Init("vk_backend", logPriority)
	log := logger.GetLogger()
	if envErr != nil {
		log.Info("Failed to load .env: %v", envErr)
	}

	log.Info("Initializing vk_backend service")
	log.Finer("Environment: %s, Log Level: %s", cfg.Environment, cfg.LogLevel)

	if err := cfg.Validate(); err != nil {
		log.Severe("Invalid configuration: %v", err)
		os.Exit(1)
	}

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
		log.Finer("Gin mode set to ReleaseMode")

		if cfg.AutoMigrate {
			log.Info("AUTO_MIGRATE disabled in production")
			cfg.AutoMigrate = false
		}
	}

	shutdownTracing, tracingEnabled, err := observability.SetupTracing(context.Background(), cfg)
	if err != nil {
		log.Info("Tracing setup failed: %v", err)
	}
	if tracingEnabled {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTracing(ctx)
		}()
	}

	// Initialize DB
	log.Finer("Connecting to database...")
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Severe("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	log.Info("Database connection established")

	if cfg.RunMigrations {
		log.Finer("Applying database migrations...")
		if err := database.ApplyMigrations(db, cfg.MigrationsDir); err != nil {
			log.Severe("Failed to apply migrations: %v", err)
			os.Exit(1)
		}
		log.Info("Database migrations applied")
	}

	// Initialize LLM service
	log.Info("Initializing LLM service...")
	llmService := services.NewLLMService(db)
	llmService.Start(cfg)
	log.Info("LLM service initialized")

	// Initialize database schema
	if cfg.AutoMigrate {
		log.Finer("Initializing database schema...")
		if err := database.InitSchema(db); err != nil {
			log.Severe("Failed to initialize schema: %v", err)
			os.Exit(1)
		}
		log.Info("Database schema initialized")
	}

	redisClient, err := infra.NewRedisClient(cfg)
	if err != nil {
		log.Info("Redis unavailable: %v", err)
		redisClient = nil
	}

	sessionStore, sessionMode, err := sessions.NewStore(db, redisClient, cfg.SessionStoreMode)
	if err != nil {
		log.Severe("Failed to initialize session store: %v", err)
		os.Exit(1)
	}
	if sessionMode != cfg.SessionStoreMode {
		log.Info("Session store fallback to %s", sessionMode)
	}

	outboxService := outbox.NewService(db)
	idempotencyStore := idempotency.NewStore(db)
	idempotencyTTL := time.Duration(cfg.IdempotencyTTLSeconds) * time.Second

	// Gin router
	router := gin.Default()
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.MaxBodyBytes(10 << 20)) // 10MB
	if len(cfg.CORSAllowedOrigins) > 0 {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CORSAllowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-Id", "X-Admin-Key"},
			ExposeHeaders:    []string{"X-Request-Id"},
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		}))
	} else {
		log.Info("CORS_ALLOWED_ORIGINS not set; no cross-origin requests will be allowed")
	}
	if tracingEnabled {
		router.Use(otelgin.Middleware(cfg.OtelServiceName))
	}
	if cfg.MetricsEnabled {
		metrics.Init()
		router.Use(middleware.MetricsMiddleware())
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		log.Finest("Health check requested")
		c.JSON(200, gin.H{"status": "ok", "service": "vk_backend"})
	})

	// Initialize services
	authService := services.NewAuthService(db, sessionStore)
	oauthService := services.NewOAuthService(db, cfg, authService)
	advocateService := services.NewAdvocateService(db)
	callService := services.NewCallService(db, outboxService)
	paymentService := services.NewPaymentService(db, cfg, outboxService)
	webrtcService := services.NewWebRTCService(db, cfg.JWTSecret)
	signalingService := services.NewSignalingService(webrtcService, cfg.WebSocketAllowedOrigins, redisClient)

	// Initialize middleware
	authMiddleware := middleware.AuthMiddleware(sessionStore)

	// Rate limiting: Redis-backed so limits are shared across instances;
	// falls back to an in-process store when Redis isn't configured (local dev).
	var rateLimitStore limiter.Store
	if redisClient != nil {
		rateLimitStore, err = limiterredis.NewStoreWithOptions(redisClient, limiter.StoreOptions{Prefix: "ratelimit"})
		if err != nil {
			log.Severe("Failed to init redis rate limit store: %v", err)
			os.Exit(1)
		}
	} else {
		log.Info("Redis unavailable, using in-process rate limit store (not shared across instances)")
		rateLimitStore = limitermemory.NewStore()
	}
	authRateLimit := middleware.NewRateLimiter(
		limiter.New(rateLimitStore, limiter.Rate{Period: time.Minute, Limit: 5}),
		middleware.IPKeyGetter("auth"), log)
	oauthRateLimit := middleware.NewRateLimiter(
		limiter.New(rateLimitStore, limiter.Rate{Period: time.Minute, Limit: 10}),
		middleware.IPKeyGetter("oauth"), log)
	paymentRateLimit := middleware.NewRateLimiter(
		limiter.New(rateLimitStore, limiter.Rate{Period: time.Minute, Limit: 10}),
		middleware.UserKeyGetter("payment"), log)

	// API routes
	api := router.Group("/api")
	{
		// WebSocket Signaling Endpoint (Public, handles its own auth via token)
		api.GET("/ws/signal", func(c *gin.Context) {
			signalingService.HandleWebSocket(c.Writer, c.Request)
		})

		auth := api.Group("/auth")
		{
			credsAuth := auth.Group("")
			credsAuth.Use(authRateLimit)
			oauthAuth := auth.Group("")
			oauthAuth.Use(oauthRateLimit)
			// ---------------------------
			// Advocate (Provider) routes
			// ---------------------------
			credsAuth.POST("/advocate/register", func(c *gin.Context) {
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

				log.Info("Advocate registered successfully: ID=%d, Email=%s", advocate.ID, piilog.MaskEmail(advocate.Email))
				c.JSON(200, advocate)
			})

			credsAuth.POST("/advocate/login", func(c *gin.Context) {
				log.Finer("Advocate login request received")
				var req models.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid login request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				advocate, err := authService.LoginAdvocate(req.Email, req.Password)
				if err != nil {
					log.Info("Advocate login failed for email: %s", piilog.MaskEmail(req.Email))
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

				log.Info("Advocate logged in successfully: ID=%d, Email=%s", advocate.ID, piilog.MaskEmail(advocate.Email))

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
			credsAuth.POST("/client/register", func(c *gin.Context) {
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

				log.Info("Client registered successfully: ID=%d, Email=%s", client.ID, piilog.MaskEmail(client.Email))
				c.JSON(200, client)
			})

			credsAuth.POST("/client/login", func(c *gin.Context) {
				log.Finer("Client login request received")
				var req models.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					log.Finer("Invalid login request: %v", err)
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				client, err := authService.LoginClient(req.Email, req.Password)
				if err != nil {
					log.Info("Client login failed for email: %s", piilog.MaskEmail(req.Email))
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

				log.Info("Client logged in successfully: ID=%d, Email=%s", client.ID, piilog.MaskEmail(client.Email))

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
			oauthAuth.GET("/google/advocate", func(c *gin.Context) {
				state := oauthService.GenerateState("advocate")
				if state == "" {
					c.JSON(500, gin.H{"error": "failed to generate oauth state"})
					return
				}
				url := oauthService.GetAuthURL(state)
				c.Redirect(http.StatusTemporaryRedirect, url)
			})

			oauthAuth.GET("/google/client", func(c *gin.Context) {
				state := oauthService.GenerateState("client")
				if state == "" {
					c.JSON(500, gin.H{"error": "failed to generate oauth state"})
					return
				}
				url := oauthService.GetAuthURL(state)
				c.Redirect(http.StatusTemporaryRedirect, url)
			})

			oauthAuth.GET("/google/callback", func(c *gin.Context) {
				code := c.Query("code")
				state := c.Query("state")

				if code == "" {
					c.JSON(400, gin.H{"error": "missing authorization code"})
					return
				}

				userType, err := oauthService.ValidateState(state)
				if err != nil {
					c.JSON(400, gin.H{"error": "invalid state"})
					return
				}

				if userType != "advocate" && userType != "client" {
					c.JSON(400, gin.H{"error": "invalid user type"})
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
					if err := db.First(&advocate, session.UserID).Error; err != nil {
						log.Severe("OAuth callback: failed to load advocate ID=%d after login: %v", session.UserID, err)
						c.JSON(500, gin.H{"error": "failed to load user after login"})
						return
					}
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
				} else {
					var client models.Client
					if err := db.First(&client, session.UserID).Error; err != nil {
						log.Severe("OAuth callback: failed to load client ID=%d after login: %v", session.UserID, err)
						c.JSON(500, gin.H{"error": "failed to load user after login"})
						return
					}
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

				log.Info("OAuth login successful: UserType=%s, UserID=%d", userType, session.UserID)
				c.JSON(200, userData)
			})

			auth.POST("/logout", authMiddleware, func(c *gin.Context) {
				token, _ := c.Get("session_token")
				if err := authService.Logout(c.Request.Context(), token.(string)); err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				c.JSON(200, gin.H{"status": "logged out"})
			})
		}

		// ---------------------------
		// Admin routes
		// ---------------------------
		admin := api.Group("/admin")
		admin.Use(middleware.AdminAuth(cfg.AdminAPIKey))
		{
			admin.POST("/sessions/revoke", func(c *gin.Context) {
				var req struct {
					UserID uint `json:"user_id" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}
				if err := authService.RevokeUserSessions(c.Request.Context(), req.UserID); err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				log.Info("Admin revoked all sessions for userID=%d", req.UserID)
				c.JSON(200, gin.H{"status": "revoked"})
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
				userType, _ := c.Get("user_type")
				if userType != "client" {
					c.JSON(403, gin.H{"error": "access denied"})
					return
				}

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

			client.GET("/advocate/:id/schedule", func(c *gin.Context) {
				idStr := c.Param("id")
				id, err := strconv.ParseUint(idStr, 10, 32)
				if err != nil {
					c.JSON(400, gin.H{"error": "invalid advocate id"})
					return
				}

				from := c.Query("from")
				to := c.Query("to")

				schedule, err := advocateService.GetAdvocateSchedule(uint(id), from, to)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}

				c.JSON(200, schedule)
			})

			payment := client.Group("/payment")
			{
				payment.Use(paymentRateLimit)
				payment.Use(middleware.IdempotencyMiddleware(idempotencyStore, idempotencyTTL))
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

					payment, err := paymentService.VerifyPayment(req.TransactionID, userID.(uint))
					if err != nil {
						c.JSON(400, gin.H{"error": err.Error()})
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
