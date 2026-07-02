package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vk_backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestMaxBodyBytesRejectsOversizedContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := middleware.MaxBodyBytes(10)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(strings.Repeat("x", 100))
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.ContentLength = 100

	handler(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
	if !c.IsAborted() {
		t.Error("expected request to be aborted")
	}
}

func TestMaxBodyBytesAllowsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := middleware.MaxBodyBytes(100)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader("small body")
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.ContentLength = int64(body.Len())

	handler(c)

	if c.IsAborted() {
		t.Error("expected request not to be aborted")
	}
}
