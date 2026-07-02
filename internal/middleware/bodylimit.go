package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes caps request body size, so a client can't exhaust server
// memory/disk with an oversized payload. A declared Content-Length over the
// limit is rejected immediately with 413 before any handler runs; the body
// is also wrapped in http.MaxBytesReader as defense-in-depth against a
// chunked/absent Content-Length that undercounts the real body size (that
// path fails during JSON binding rather than with a clean 413).
func MaxBodyBytes(limitBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > limitBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limitBytes)
		c.Next()
	}
}
