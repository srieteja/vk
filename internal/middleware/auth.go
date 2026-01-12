package middleware

import (
	"strings"
	"time"

	"vk_backend/internal/logger"
	"vk_backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	log := logger.NewLogger("AuthMiddleware", logger.INFO)

	return func(c *gin.Context) {
		log.Finest("AuthMiddleware: validating request for path: %s", c.Request.URL.Path)

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Finer("AuthMiddleware: missing authorization header")
			c.JSON(401, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Finer("AuthMiddleware: invalid authorization header format")
			c.JSON(401, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate session
		var session models.Session
		if err := db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&session).Error; err != nil {
			log.Info("AuthMiddleware: invalid or expired token")
			c.JSON(401, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", session.UserID)
		c.Set("user_type", session.UserType)

		log.Finest("AuthMiddleware: authentication successful for userID=%d, userType=%s", session.UserID, session.UserType)
		c.Next()
	}
}
