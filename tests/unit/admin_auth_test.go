package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"vk_backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestAdminAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		configuredKey  string
		providedHeader string
		expectedStatus int
	}{
		{"valid key", "correct-secret", "correct-secret", 200},
		{"wrong key", "correct-secret", "wrong-secret", 401},
		{"missing header", "correct-secret", "", 401},
		{"admin disabled (no configured key)", "", "anything", 503},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("POST", "/", nil)
			if tc.providedHeader != "" {
				c.Request.Header.Set("X-Admin-Key", tc.providedHeader)
			}

			handler := middleware.AdminAuth(tc.configuredKey)
			handler(c)
			if !c.IsAborted() {
				c.Status(200)
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
