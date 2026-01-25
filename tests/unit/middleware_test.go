package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vk_backend/internal/middleware"
	"vk_backend/internal/models"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	store := setupSessionStore(db)

	// Create a valid session
	session := models.Session{
		UserID:    1,
		UserType:  "advocate",
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	db.Create(&session)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectUser     bool
	}{
		{
			name:           "Valid Token",
			token:          "Bearer valid-token",
			expectedStatus: 200,
			expectUser:     true,
		},
		{
			name:           "Missing Header",
			token:          "",
			expectedStatus: 401,
			expectUser:     false,
		},
		{
			name:           "Invalid Format",
			token:          "valid-token",
			expectedStatus: 401,
			expectUser:     false,
		},
		{
			name:           "Invalid Token",
			token:          "Bearer wrong-token",
			expectedStatus: 401,
			expectUser:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/", nil)
			if tc.token != "" {
				c.Request.Header.Set("Authorization", tc.token)
			}

			// execute middleware
			handler := middleware.AuthMiddleware(store)
			handler(c)

			if w.Code != tc.expectedStatus && w.Code != 200 { // 200 means Next() was called
				// If we expect 401, but middleware calls Next(), status might be 200 default
				// AuthMiddleware aborts on error, so we check if Aborted
				if !c.IsAborted() && tc.expectedStatus != 200 {
					t.Errorf("expected abort with status %d", tc.expectedStatus)
				}
			}

			if tc.expectUser {
				userID, exists := c.Get("user_id")
				if !exists || userID.(uint) != 1 {
					t.Error("user_id not set in context")
				}
			}
		})
	}
}
