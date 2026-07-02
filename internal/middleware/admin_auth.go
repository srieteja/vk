package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"
)

// AdminAuth gates admin-only endpoints behind a shared secret passed in the
// X-Admin-Key header, compared in constant time to avoid a timing side
// channel. If adminAPIKey is empty, the admin endpoints are disabled
// entirely (every request is rejected) rather than accepting any key.
func AdminAuth(adminAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if adminAPIKey == "" {
			c.JSON(503, gin.H{"error": "admin endpoints are not configured"})
			c.Abort()
			return
		}

		provided := c.GetHeader("X-Admin-Key")
		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(adminAPIKey)) != 1 {
			c.JSON(401, gin.H{"error": "invalid admin key"})
			c.Abort()
			return
		}

		c.Next()
	}
}
