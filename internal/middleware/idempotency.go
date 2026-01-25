package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"vk_backend/internal/idempotency"
	"vk_backend/internal/logger"
	"vk_backend/internal/models"

	"github.com/gin-gonic/gin"
)

type responseRecorder struct {
	gin.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

func IdempotencyMiddleware(store *idempotency.Store, ttl time.Duration) gin.HandlerFunc {
	log := logger.NewLogger("IdempotencyMiddleware", logger.INFO)

	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		if store == nil {
			c.JSON(500, gin.H{"error": "idempotency store not configured"})
			c.Abort()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to read request body"})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		requestHash := hashRequest(c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery, bodyBytes)
		record, err := store.Get(c.Request.Context(), key)
		if err == nil {
			if record.ExpiresAt.Before(time.Now()) {
				_ = store.DeleteExpired(c.Request.Context())
			} else if record.RequestHash != requestHash {
				c.JSON(409, gin.H{"error": "idempotency key reused with different request"})
				c.Abort()
				return
			} else {
				c.Header("Idempotency-Replayed", "true")
				c.Data(record.ResponseStatus, "application/json", []byte(record.ResponseBody))
				c.Abort()
				return
			}
		} else if err != idempotency.ErrKeyNotFound {
			log.Info("Idempotency lookup failed: %v", err)
			c.JSON(500, gin.H{"error": "idempotency lookup failed"})
			c.Abort()
			return
		}

		recorder := &responseRecorder{ResponseWriter: c.Writer}
		c.Writer = recorder

		c.Next()

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		expiresAt := time.Now().Add(ttl)

		err = store.Save(c.Request.Context(), &models.IdempotencyKey{
			Key:            key,
			UserID:         toUint(userID),
			UserType:       toString(userType),
			RequestHash:    requestHash,
			ResponseStatus: status,
			ResponseBody:   recorder.body.String(),
			CreatedAt:      time.Now(),
			ExpiresAt:      expiresAt,
		})
		if err != nil {
			log.Info("Failed to persist idempotency key: %v", err)
		}
	}
}

func hashRequest(method string, path string, rawQuery string, body []byte) string {
	key := method + "|" + path + "|" + rawQuery + "|"
	sum := sha256.Sum256(append([]byte(key), body...))
	return hex.EncodeToString(sum[:])
}

func toUint(value interface{}) uint {
	if value == nil {
		return 0
	}
	if v, ok := value.(uint); ok {
		return v
	}
	return 0
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}
