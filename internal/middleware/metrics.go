package middleware

import (
	"strconv"
	"time"

	"vk_backend/internal/metrics"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		metrics.Inflight.WithLabelValues(c.Request.Method, path).Inc()
		start := time.Now()

		c.Next()

		metrics.Inflight.WithLabelValues(c.Request.Method, path).Dec()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()
		metrics.RequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.RequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
