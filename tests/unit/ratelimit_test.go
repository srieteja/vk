package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vk_backend/internal/logger"
	"vk_backend/internal/middleware"

	"github.com/gin-gonic/gin"
	limiter "github.com/ulule/limiter/v3"
	limitermemory "github.com/ulule/limiter/v3/drivers/store/memory"
)

func TestRateLimitBlocksAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := limitermemory.NewStore()
	log := logger.NewLogger("test", logger.INFO)
	l := limiter.New(store, limiter.Rate{Period: time.Minute, Limit: 2})
	handler := middleware.NewRateLimiter(l, middleware.IPKeyGetter("test"), log)

	run := func() int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.RemoteAddr = "1.2.3.4:1234"
		handler(c)
		return w.Code
	}

	if code := run(); code == http.StatusTooManyRequests {
		t.Fatalf("request 1: expected not limited, got %d", code)
	}
	if code := run(); code == http.StatusTooManyRequests {
		t.Fatalf("request 2: expected not limited, got %d", code)
	}
	if code := run(); code != http.StatusTooManyRequests {
		t.Errorf("request 3: expected 429, got %d", code)
	}
}

func TestRateLimitScopedPerKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := limitermemory.NewStore()
	log := logger.NewLogger("test", logger.INFO)
	l := limiter.New(store, limiter.Rate{Period: time.Minute, Limit: 1})
	handler := middleware.NewRateLimiter(l, middleware.IPKeyGetter("test"), log)

	run := func(ip string) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.RemoteAddr = ip + ":1234"
		handler(c)
		return w.Code
	}

	if code := run("1.1.1.1"); code == http.StatusTooManyRequests {
		t.Fatalf("ip1 request 1: expected not limited, got %d", code)
	}
	if code := run("2.2.2.2"); code == http.StatusTooManyRequests {
		t.Fatalf("ip2 request 1: expected not limited (different IP), got %d", code)
	}
	if code := run("1.1.1.1"); code != http.StatusTooManyRequests {
		t.Errorf("ip1 request 2: expected 429, got %d", code)
	}
}
