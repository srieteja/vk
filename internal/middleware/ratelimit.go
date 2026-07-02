package middleware

import (
	"fmt"
	"net/http"

	"vk_backend/internal/logger"

	"github.com/gin-gonic/gin"
	limiter "github.com/ulule/limiter/v3"
	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
)

// NewRateLimiter builds a gin middleware enforcing rate on l, keyed by
// keyGetter. Store errors fail open (request proceeds) rather than the
// library's default of panicking, so a rate-limit-store outage degrades to
// "no rate limiting" instead of a full 500 on every request.
func NewRateLimiter(l *limiter.Limiter, keyGetter limitergin.KeyGetter, log *logger.Logger) gin.HandlerFunc {
	return limitergin.NewMiddleware(l,
		limitergin.WithKeyGetter(keyGetter),
		limitergin.WithErrorHandler(func(c *gin.Context, err error) {
			log.Severe("rate limiter store error, failing open: %v", err)
			c.Next()
		}),
		limitergin.WithLimitReachedHandler(func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded, please try again later"})
		}),
	)
}

// IPKeyGetter keys the limiter bucket by client IP, namespaced by name so
// multiple limiters can share one underlying store without key collisions.
func IPKeyGetter(name string) limitergin.KeyGetter {
	return func(c *gin.Context) string {
		return name + ":" + c.ClientIP()
	}
}

// UserKeyGetter keys the limiter bucket by authenticated user ID, namespaced
// by name. Must run after AuthMiddleware, which sets "user_id" in context;
// falls back to client IP if it's somehow missing.
func UserKeyGetter(name string) limitergin.KeyGetter {
	return func(c *gin.Context) string {
		if userID, ok := c.Get("user_id"); ok {
			return fmt.Sprintf("%s:user:%v", name, userID)
		}
		return name + ":" + c.ClientIP()
	}
}
