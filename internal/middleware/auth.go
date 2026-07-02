package middleware

import (
	"errors"
	"strings"

	"vk_backend/internal/logger"
	"vk_backend/internal/sessions"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(store sessions.Store) gin.HandlerFunc {
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

		if store == nil {
			log.Info("AuthMiddleware: session store not configured")
			c.JSON(500, gin.H{"error": "session store not configured"})
			c.Abort()
			return
		}

		// Validate session
		session, err := store.GetByToken(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, sessions.ErrSessionNotFound) {
				log.Info("AuthMiddleware: invalid or expired token")
				c.JSON(401, gin.H{"error": "invalid or expired token"})
			} else {
				log.Info("AuthMiddleware: session lookup failed: %v", err)
				c.JSON(500, gin.H{"error": "session lookup failed"})
			}
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user_id", session.UserID)
		c.Set("user_type", session.UserType)
		c.Set("session_token", token)

		log.Finest("AuthMiddleware: authentication successful for userID=%d, userType=%s", session.UserID, session.UserType)
		c.Next()
	}
}
