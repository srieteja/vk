package middleware

import (
	"strings"
	"time"

	"enterprise-api/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate session
		var session models.Session
		if err := db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&session).Error; err != nil {
			c.JSON(401, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", session.UserID)
		c.Set("user_type", session.UserType)

		c.Next()
	}
}
