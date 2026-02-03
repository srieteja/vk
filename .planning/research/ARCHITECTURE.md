# Testing Architecture Research

**Domain:** Security-Critical Go Backend Testing
**Researched:** 2026-02-03
**Confidence:** HIGH

## Current State Analysis

**Codebase Size:** 38 non-test Go files, 9 test files (19% coverage)
**Testing Framework:** Standard `testing` package with minimal adoption
**Critical Components:**
- Payment processing (race condition risks)
- Auth/OAuth flows (security-critical)
- WebRTC signaling (real-time security)
- Session management (token security)
- Database transactions (GORM with PostgreSQL/SQLite)
- Redis caching
- Idempotency middleware

**Identified Gaps:**
- No integration tests (services tested in isolation)
- No security tests (injection, bypass, timing attacks)
- No race condition tests (despite `go test -race` availability)
- No load/stress tests for payment paths
- Minimal negative/edge case coverage
- No HTTP handler tests (middleware only partially tested)

## Comprehensive Testing Strategy

### Testing Pyramid Structure

```
                    ┌──────────────────────────┐
                    │   E2E Tests (Manual)     │  5%
                    │ Full system validation   │
                    └──────────────────────────┘
                  ┌────────────────────────────────┐
                  │   Integration Tests             │  20%
                  │ Service + DB + Redis + HTTP     │
                  └────────────────────────────────┘
              ┌──────────────────────────────────────────┐
              │   Security Tests                          │  15%
              │ Injection, bypass, race, timing attacks   │
              └──────────────────────────────────────────┘
          ┌──────────────────────────────────────────────────┐
          │   Unit Tests                                      │  60%
          │ Individual functions, happy + edge + negative    │
          └──────────────────────────────────────────────────┘
```

## Testing Framework Recommendations

### 1. Core Testing Stack

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| Unit Testing | `testing` (stdlib) | Go 1.25+ | Foundation for all tests |
| Assertions | `testify/assert` | v1.10.0+ | Readable assertions, better failure messages |
| Mocking | `testify/mock` | v1.10.0+ | Interface mocking, call verification |
| HTTP Testing | `net/http/httptest` (stdlib) | Go 1.25+ | HTTP handler testing |
| DB Testing | `gorm` + SQLite in-memory | Current | Fast, isolated DB tests |
| Race Detection | `go test -race` (stdlib) | Go 1.25+ | Concurrency bug detection |
| Coverage | `go test -cover` (stdlib) | Go 1.25+ | Coverage measurement |

**Installation:**
```bash
go get github.com/stretchr/testify@v1.10.0
```

### 2. Security Testing Tools

| Tool | Purpose | Integration |
|------|---------|-------------|
| `go test -race` | Race condition detection | CI pipeline |
| SQL injection scanner | Parameterized query validation | Custom test helper |
| HMAC timing attack validator | Constant-time comparison checks | Security test suite |
| JWT expiration tester | Token lifetime validation | Auth test suite |
| OAuth state parameter validator | CSRF protection testing | OAuth test suite |

### 3. Load/Performance Testing

| Tool | Purpose | When |
|------|---------|------|
| `testing.B` (stdlib) | Benchmark critical paths | Regular benchmarks |
| `pprof` (stdlib) | Profiling payment flows | Performance debugging |
| Custom concurrent test | Simulate race conditions | Payment verification |

## Recommended Test Organization

```
vk_backend/
├── tests/
│   ├── unit/              # Current location (keep)
│   │   ├── setup_test.go  # Shared test DB setup
│   │   ├── *_service_test.go
│   │   └── *_middleware_test.go
│   ├── integration/       # NEW
│   │   ├── setup_integration_test.go
│   │   ├── payment_flow_test.go
│   │   ├── auth_flow_test.go
│   │   ├── oauth_flow_test.go
│   │   ├── call_lifecycle_test.go
│   │   └── session_management_test.go
│   ├── security/          # NEW
│   │   ├── setup_security_test.go
│   │   ├── sql_injection_test.go
│   │   ├── auth_bypass_test.go
│   │   ├── race_conditions_test.go
│   │   ├── timing_attacks_test.go
│   │   ├── session_hijacking_test.go
│   │   └── oauth_csrf_test.go
│   ├── load/              # NEW
│   │   ├── payment_load_test.go
│   │   ├── auth_load_test.go
│   │   └── websocket_load_test.go
│   └── testutil/          # NEW
│       ├── fixtures.go    # Test data builders
│       ├── mocks.go       # Mock implementations
│       ├── security.go    # Security test helpers
│       └── assertions.go  # Custom assertions
├── internal/
│   ├── services/
│   │   ├── payment_service.go
│   │   └── payment_service_test.go  # MOVE HERE (package-local)
│   ├── middleware/
│   │   ├── auth.go
│   │   └── auth_test.go             # MOVE HERE (package-local)
│   └── ...
└── go.mod
```

### Organization Rationale

**Unit tests alongside source:**
- Faster to find relevant tests
- Encourages test-driven development
- Go convention (`package_test` or same package)
- Access to internal functions when needed

**Integration tests separate:**
- Cross-service interactions
- Require full DB/Redis setup
- Slower execution (excluded from quick runs)

**Security tests isolated:**
- Specialized tooling
- Potentially dangerous test cases
- Separate CI pipeline stage

**Load tests separate:**
- Resource-intensive
- Run on-demand or nightly
- Different success criteria (throughput, latency)

## Testing Patterns by Component

### Pattern 1: Service Unit Testing with testify

**What:** Test individual service methods with mocked dependencies
**When to use:** Every service method, especially business logic
**Trade-offs:** Fast, isolated, but may miss integration issues

**Example:**
```go
package services_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "vk_backend/internal/services"
)

// Mock GORM DB interface
type MockDB struct {
    mock.Mock
}

func (m *MockDB) Create(value interface{}) *gorm.DB {
    args := m.Called(value)
    return args.Get(0).(*gorm.DB)
}

func TestPaymentService_InitiatePayment_Success(t *testing.T) {
    // Arrange
    db := setupTestDB()
    cfg := &config.Config{PlatformCommission: 20.0}
    service := services.NewPaymentService(db, cfg, nil)

    client := models.Client{Name: "Test", Email: "test@test.com", Balance: 100}
    advocate := models.Advocate{Name: "Advocate", Email: "adv@test.com"}
    db.Create(&client)
    db.Create(&advocate)

    // Act
    payment, err := service.InitiatePayment(client.ID, advocate.ID, 50.0)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 50.0, payment.Amount)
    assert.Equal(t, 10.0, payment.PlatformFee) // 20% of 50
    assert.Equal(t, 40.0, payment.AdvocateCommission)
    assert.Equal(t, "pending", payment.Status)
}

func TestPaymentService_InitiatePayment_NegativeAmount(t *testing.T) {
    db := setupTestDB()
    cfg := &config.Config{PlatformCommission: 20.0}
    service := services.NewPaymentService(db, cfg, nil)

    payment, err := service.InitiatePayment(1, 2, -50.0)

    assert.Error(t, err)
    assert.Nil(t, payment)
    assert.Contains(t, err.Error(), "invalid amount")
}
```

### Pattern 2: Table-Driven Tests for Edge Cases

**What:** Test multiple scenarios with same test logic
**When to use:** Testing input validation, edge cases, negative paths
**Trade-offs:** Compact, comprehensive, but can be harder to debug

**Example:**
```go
func TestAuthService_RegisterAdvocate(t *testing.T) {
    tests := []struct {
        name        string
        request     *models.RegisterRequest
        expectError bool
        errorMsg    string
    }{
        {
            name: "Valid registration",
            request: &models.RegisterRequest{
                Email: "test@example.com",
                Password: "SecurePass123!",
                Name: "Test User",
            },
            expectError: false,
        },
        {
            name: "Empty email",
            request: &models.RegisterRequest{
                Email: "",
                Password: "SecurePass123!",
                Name: "Test User",
            },
            expectError: true,
            errorMsg: "email, password, and name are required",
        },
        {
            name: "Weak password (too short)",
            request: &models.RegisterRequest{
                Email: "test@example.com",
                Password: "123",
                Name: "Test User",
            },
            expectError: true,
            errorMsg: "password must be at least 8 characters",
        },
        {
            name: "SQL injection attempt in email",
            request: &models.RegisterRequest{
                Email: "admin'--",
                Password: "SecurePass123!",
                Name: "Test User",
            },
            expectError: true, // Should fail validation
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db := setupTestDB()
            service := services.NewAuthService(db, setupSessionStore(db))

            user, err := service.RegisterAdvocate(tt.request)

            if tt.expectError {
                assert.Error(t, err)
                if tt.errorMsg != "" {
                    assert.Contains(t, err.Error(), tt.errorMsg)
                }
                assert.Nil(t, user)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, user)
                assert.Equal(t, tt.request.Email, user.Email)
            }
        })
    }
}
```

### Pattern 3: Integration Testing with Real Dependencies

**What:** Test service interactions with actual DB, Redis, HTTP
**When to use:** Multi-service flows (auth → session → middleware)
**Trade-offs:** More realistic, catches integration bugs, but slower

**Example:**
```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "vk_backend/internal/middleware"
    "vk_backend/internal/services"
)

func TestAuthFlow_RegisterLoginAccessProtectedRoute(t *testing.T) {
    // Setup full stack
    db, redis, cleanup := setupIntegrationEnv(t)
    defer cleanup()

    authService := services.NewAuthService(db, sessions.NewStore(db, redis))

    // Setup Gin router with middleware
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.POST("/register", func(c *gin.Context) {
        var req models.RegisterRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        user, err := authService.RegisterAdvocate(&req)
        if err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, user)
    })

    protected := router.Group("/")
    protected.Use(middleware.AuthMiddleware(authService.sessionStore))
    protected.GET("/profile", func(c *gin.Context) {
        userID, _ := c.Get("user_id")
        c.JSON(200, gin.H{"user_id": userID})
    })

    // Test: Register
    regBody, _ := json.Marshal(models.RegisterRequest{
        Email: "test@example.com",
        Password: "SecurePass123!",
        Name: "Test User",
    })
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(regBody))
    req.Header.Set("Content-Type", "application/json")
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)

    // Test: Login and get token
    // ... (similar pattern)

    // Test: Access protected route with token
    w = httptest.NewRecorder()
    req, _ = http.NewRequest("GET", "/profile", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
}
```

### Pattern 4: Race Condition Testing

**What:** Detect data races in concurrent payment processing
**When to use:** Payment verification, session management, concurrent writes
**Trade-offs:** Slower tests, requires `-race` flag, may have false positives

**Example:**
```go
func TestPaymentService_ConcurrentVerification_NoDoubleCredit(t *testing.T) {
    db := setupTestDB()
    cfg := &config.Config{PlatformCommission: 20.0}
    service := services.NewPaymentService(db, cfg, nil)

    // Setup
    client := models.Client{Name: "Client", Balance: 100}
    advocate := models.Advocate{Name: "Advocate", Earnings: 0}
    db.Create(&client)
    db.Create(&advocate)

    payment, _ := service.InitiatePayment(client.ID, advocate.ID, 100.0)

    // Act: Simulate concurrent verification attempts
    const goroutines = 10
    errChan := make(chan error, goroutines)

    for i := 0; i < goroutines; i++ {
        go func() {
            _, err := service.VerifyPayment(payment.TransactionID)
            errChan <- err
        }()
    }

    // Collect results
    var successCount int
    for i := 0; i < goroutines; i++ {
        if err := <-errChan; err == nil {
            successCount++
        }
    }

    // Assert: Only one verification should succeed
    assert.Equal(t, 1, successCount, "Only one verification should succeed")

    // Verify advocate was credited exactly once
    var updatedAdvocate models.Advocate
    db.First(&updatedAdvocate, advocate.ID)
    assert.Equal(t, 80.0, updatedAdvocate.Earnings) // 80% of 100
}
```

**Run with:**
```bash
go test -race ./tests/security/race_conditions_test.go
```

### Pattern 5: Security Testing - SQL Injection

**What:** Verify parameterized queries prevent injection
**When to use:** All database queries accepting user input
**Trade-offs:** Requires malicious input knowledge, may not catch all vectors

**Example:**
```go
func TestAuthService_LoginAdvocate_SQLInjectionResistance(t *testing.T) {
    db := setupTestDB()
    service := services.NewAuthService(db, setupSessionStore(db))

    // Register legitimate user
    service.RegisterAdvocate(&models.RegisterRequest{
        Email: "legitimate@example.com",
        Password: "SecurePass123!",
        Name: "Legitimate User",
    })

    injectionAttempts := []string{
        "' OR '1'='1",
        "' OR '1'='1' --",
        "admin'--",
        "' UNION SELECT * FROM advocates --",
        "'; DROP TABLE advocates; --",
    }

    for _, attempt := range injectionAttempts {
        t.Run("Injection: "+attempt, func(t *testing.T) {
            user, err := service.LoginAdvocate(attempt, "anything")

            // Should fail authentication, not cause SQL error or bypass
            assert.Error(t, err)
            assert.Nil(t, user)
            assert.Equal(t, "invalid credentials", err.Error())
        })
    }

    // Verify table still exists and legitimate user unaffected
    user, err := service.LoginAdvocate("legitimate@example.com", "SecurePass123!")
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
```

### Pattern 6: Security Testing - Timing Attacks

**What:** Verify constant-time comparisons for sensitive operations
**When to use:** Token validation, password comparison, HMAC verification
**Trade-offs:** Statistical tests, may be flaky, requires many samples

**Example:**
```go
func TestOAuthService_ValidateState_TimingAttackResistance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping timing attack test in short mode")
    }

    db := setupTestDB()
    cfg := &config.Config{OAuthStateSecret: "test-secret-key"}
    service := services.NewOAuthService(db, cfg, nil)

    validState := service.GenerateState("advocate")
    invalidState := "completely.wrong.state"

    // Measure timing for valid vs invalid states
    const iterations = 1000

    validDurations := make([]time.Duration, iterations)
    for i := 0; i < iterations; i++ {
        start := time.Now()
        service.ValidateState(validState)
        validDurations[i] = time.Since(start)
    }

    invalidDurations := make([]time.Duration, iterations)
    for i := 0; i < iterations; i++ {
        start := time.Now()
        service.ValidateState(invalidState)
        invalidDurations[i] = time.Since(start)
    }

    // Statistical analysis
    validAvg := average(validDurations)
    invalidAvg := average(invalidDurations)

    // Timing difference should be minimal (< 10% variance)
    diff := math.Abs(float64(validAvg - invalidAvg))
    maxAllowedDiff := float64(validAvg) * 0.1

    assert.Less(t, diff, maxAllowedDiff,
        "Timing difference suggests vulnerability to timing attacks")
}

func average(durations []time.Duration) time.Duration {
    var sum time.Duration
    for _, d := range durations {
        sum += d
    }
    return sum / time.Duration(len(durations))
}
```

### Pattern 7: Security Testing - Session Hijacking

**What:** Verify session tokens are properly validated
**When to use:** Session management, token-based auth
**Trade-offs:** Requires understanding of attack vectors

**Example:**
```go
func TestAuthMiddleware_SessionHijackingPrevention(t *testing.T) {
    gin.SetMode(gin.TestMode)
    db := setupTestDB()
    store := setupSessionStore(db)

    // Create valid session
    validSession := &models.Session{
        UserID: 1,
        UserType: "advocate",
        Token: "valid-token-12345",
        ExpiresAt: time.Now().Add(1 * time.Hour),
    }
    store.Create(context.Background(), validSession)

    tests := []struct {
        name           string
        token          string
        expectedStatus int
        description    string
    }{
        {
            name: "Valid token",
            token: "Bearer valid-token-12345",
            expectedStatus: 200,
            description: "Legitimate access should work",
        },
        {
            name: "Modified token",
            token: "Bearer valid-token-12346", // Changed last character
            expectedStatus: 401,
            description: "Modified token should be rejected",
        },
        {
            name: "Truncated token",
            token: "Bearer valid-token-123",
            expectedStatus: 401,
            description: "Truncated token should be rejected",
        },
        {
            name: "Extended token",
            token: "Bearer valid-token-123456",
            expectedStatus: 401,
            description: "Extended token should be rejected",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := httptest.NewRecorder()
            c, _ := gin.CreateTestContext(w)
            c.Request, _ = http.NewRequest("GET", "/", nil)
            c.Request.Header.Set("Authorization", tt.token)

            handler := middleware.AuthMiddleware(store)
            handler(c)

            if tt.expectedStatus == 401 {
                assert.True(t, c.IsAborted(), tt.description)
            } else {
                assert.False(t, c.IsAborted(), tt.description)
            }
        })
    }
}
```

### Pattern 8: Load Testing with Benchmarks

**What:** Measure throughput and latency of critical paths
**When to use:** Payment verification, auth flows, WebRTC signaling
**Trade-offs:** Provides performance baseline, but not realistic load

**Example:**
```go
func BenchmarkPaymentService_InitiatePayment(b *testing.B) {
    db := setupTestDB()
    cfg := &config.Config{PlatformCommission: 20.0}
    service := services.NewPaymentService(db, cfg, nil)

    // Setup test data
    client := models.Client{Name: "Client", Balance: 100000}
    advocate := models.Advocate{Name: "Advocate"}
    db.Create(&client)
    db.Create(&advocate)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.InitiatePayment(client.ID, advocate.ID, 50.0)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkPaymentService_VerifyPayment_Concurrent(b *testing.B) {
    db := setupTestDB()
    cfg := &config.Config{PlatformCommission: 20.0}
    service := services.NewPaymentService(db, cfg, nil)

    // Setup: Create multiple payments
    client := models.Client{Name: "Client", Balance: 100000}
    advocate := models.Advocate{Name: "Advocate"}
    db.Create(&client)
    db.Create(&advocate)

    payments := make([]*models.Payment, b.N)
    for i := 0; i < b.N; i++ {
        payments[i], _ = service.InitiatePayment(client.ID, advocate.ID, 50.0)
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            if i < len(payments) {
                service.VerifyPayment(payments[i].TransactionID)
                i++
            }
        }
    })
}
```

**Run benchmarks:**
```bash
go test -bench=. -benchmem ./tests/load/
```

## Test Data Management

### Test Fixtures Pattern

```go
// tests/testutil/fixtures.go
package testutil

import "vk_backend/internal/models"

type AdvocateBuilder struct {
    advocate *models.Advocate
}

func NewAdvocateBuilder() *AdvocateBuilder {
    return &AdvocateBuilder{
        advocate: &models.Advocate{
            Name: "Default Advocate",
            Email: "advocate@test.com",
            Availability: "available",
            Earnings: 0.0,
            HourlyRate: 50.0,
            UUID: "test-uuid-advocate",
        },
    }
}

func (b *AdvocateBuilder) WithEmail(email string) *AdvocateBuilder {
    b.advocate.Email = email
    return b
}

func (b *AdvocateBuilder) WithEarnings(earnings float64) *AdvocateBuilder {
    b.advocate.Earnings = earnings
    return b
}

func (b *AdvocateBuilder) Build() *models.Advocate {
    return b.advocate
}

// Usage:
// advocate := testutil.NewAdvocateBuilder().
//     WithEmail("custom@test.com").
//     WithEarnings(100.0).
//     Build()
```

## Coverage Targets and Measurement

### Recommended Coverage Levels

| Category | Target Coverage | Rationale |
|----------|----------------|-----------|
| Security-critical services | 95%+ | Payment, Auth, OAuth, WebRTC token |
| Business logic services | 85%+ | Advocate, Call, Signaling |
| Middleware | 90%+ | Auth, Idempotency, Rate limiting |
| Models/DTOs | 70%+ | Mostly data structures |
| Infrastructure | 60%+ | Database, Redis, Config |
| Overall project | 80%+ | Production-grade baseline |

### Coverage Measurement Commands

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go tool cover -func=coverage.out | grep total

# Coverage by package
go test -cover ./...

# Detailed coverage with race detection
go test -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Coverage Enforcement

```bash
# Fail if coverage drops below threshold
go test -cover ./... | \
  awk '/coverage:/ {total += $NF; count++} END {avg = total/count; if (avg < 80) exit 1}'
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Test Suite

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run unit tests
        run: go test -v -race -cover ./...

      - name: Generate coverage
        run: go test -coverprofile=coverage.out ./...

      - name: Check coverage threshold
        run: |
          total=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$total < 80" | bc -l) )); then
            echo "Coverage $total% is below 80% threshold"
            exit 1
          fi

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run integration tests
        run: go test -v ./tests/integration/...
        env:
          DATABASE_URL: postgres://postgres:test@localhost:5432/test
          REDIS_URL: redis://localhost:6379

  security-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run security tests
        run: go test -v ./tests/security/...

      - name: Run race detector
        run: go test -race ./internal/services/...
```

## Test Execution Strategy

### Quick Feedback Loop (Development)

```bash
# Run tests in changed package only
go test -v ./internal/services/

# Run specific test
go test -v -run TestPaymentService_InitiatePayment ./internal/services/

# Run with race detector (slower)
go test -race ./internal/services/payment_service_test.go
```

### Pre-Commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Run tests on changed packages
changed_packages=$(git diff --cached --name-only | grep '\.go$' | xargs -I {} dirname {} | sort -u)

for pkg in $changed_packages; do
  echo "Testing $pkg..."
  go test -race -cover ./$pkg/
  if [ $? -ne 0 ]; then
    echo "Tests failed in $pkg"
    exit 1
  fi
done
```

### Full CI Suite (Pre-Merge)

```bash
# Run all tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Run integration tests
go test -v ./tests/integration/...

# Run security tests
go test -v ./tests/security/...

# Run benchmarks
go test -bench=. -benchmem ./tests/load/
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Testing Implementation Details

**What people do:**
```go
func TestPaymentService_calculateCommission(t *testing.T) {
    // Testing private method indirectly
    service := services.NewPaymentService(...)
    // This is fragile - breaks if implementation changes
}
```

**Why it's wrong:** Tests become coupled to implementation, not behavior

**Do this instead:**
```go
func TestPaymentService_InitiatePayment_CorrectCommissions(t *testing.T) {
    // Test public API and verify observable behavior
    payment, _ := service.InitiatePayment(client.ID, advocate.ID, 100.0)
    assert.Equal(t, 20.0, payment.PlatformFee)
    assert.Equal(t, 80.0, payment.AdvocateCommission)
}
```

### Anti-Pattern 2: Shared Mutable Test State

**What people do:**
```go
var db *gorm.DB

func init() {
    db = setupTestDB() // Shared across all tests
}

func TestA(t *testing.T) {
    db.Create(&models.Client{...}) // Mutates shared state
}

func TestB(t *testing.T) {
    // Relies on state from TestA - flaky!
}
```

**Why it's wrong:** Tests become order-dependent and flaky

**Do this instead:**
```go
func TestA(t *testing.T) {
    db := setupTestDB() // Fresh DB per test
    db.Create(&models.Client{...})
    // ...
}
```

### Anti-Pattern 3: Testing Mocks, Not Behavior

**What people do:**
```go
func TestPaymentService_InitiatePayment(t *testing.T) {
    mockDB := new(MockDB)
    mockDB.On("Create", mock.Anything).Return(nil)

    service := services.NewPaymentService(mockDB, cfg, nil)
    service.InitiatePayment(1, 2, 50.0)

    mockDB.AssertExpectations(t) // Only verifies mock was called
}
```

**Why it's wrong:** Doesn't verify actual behavior or return values

**Do this instead:**
```go
func TestPaymentService_InitiatePayment(t *testing.T) {
    db := setupTestDB() // Real DB (or minimal mock)
    service := services.NewPaymentService(db, cfg, nil)

    payment, err := service.InitiatePayment(1, 2, 50.0)

    // Verify behavior
    assert.NoError(t, err)
    assert.Equal(t, 50.0, payment.Amount)
    assert.Equal(t, "pending", payment.Status)
}
```

### Anti-Pattern 4: Ignoring Race Conditions

**What people do:**
```go
func TestConcurrentAccess(t *testing.T) {
    // No -race flag, no synchronization testing
    go service.VerifyPayment(txnID)
    go service.VerifyPayment(txnID)
}
```

**Why it's wrong:** Misses race conditions that cause production bugs

**Do this instead:**
```go
func TestConcurrentAccess(t *testing.T) {
    var wg sync.WaitGroup
    results := make(chan error, 2)

    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := service.VerifyPayment(txnID)
            results <- err
        }()
    }

    wg.Wait()
    close(results)

    // Verify only one succeeded
    var successCount int
    for err := range results {
        if err == nil {
            successCount++
        }
    }
    assert.Equal(t, 1, successCount)
}
```

**Run with:** `go test -race`

### Anti-Pattern 5: No Negative Testing

**What people do:**
```go
func TestPaymentService_InitiatePayment(t *testing.T) {
    // Only tests happy path
    payment, err := service.InitiatePayment(1, 2, 50.0)
    assert.NoError(t, err)
}
```

**Why it's wrong:** Misses validation bugs, security issues

**Do this instead:**
```go
func TestPaymentService_InitiatePayment_EdgeCases(t *testing.T) {
    tests := []struct{
        name string
        amount float64
        expectError bool
    }{
        {"Valid amount", 50.0, false},
        {"Zero amount", 0.0, true},
        {"Negative amount", -50.0, true},
        {"Very large amount", 999999999.99, false},
        {"NaN", math.NaN(), true},
        {"Infinity", math.Inf(1), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            payment, err := service.InitiatePayment(1, 2, tt.amount)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Security Testing Checklist

### Authentication/Authorization Tests

- [ ] SQL injection in login/register forms
- [ ] Password complexity validation
- [ ] Bcrypt hash strength (cost factor)
- [ ] Session token randomness (sufficient entropy)
- [ ] Session expiration enforcement
- [ ] Token reuse prevention
- [ ] Authorization bypass attempts
- [ ] Role-based access control (advocate vs client)

### OAuth Security Tests

- [ ] State parameter validation (CSRF protection)
- [ ] State parameter expiration
- [ ] State parameter replay attack prevention
- [ ] HMAC signature verification (timing-safe)
- [ ] Redirect URL validation
- [ ] Token exchange error handling

### Payment Security Tests

- [ ] Race condition in payment verification
- [ ] Double-credit prevention
- [ ] Negative amount rejection
- [ ] Integer overflow in commission calculation
- [ ] Transaction isolation
- [ ] Idempotency key enforcement

### WebRTC Security Tests

- [ ] Token signature validation
- [ ] Token expiration enforcement
- [ ] User authorization for call access
- [ ] Call status validation before token generation
- [ ] HMAC timing-safe comparison

### Session Security Tests

- [ ] Session hijacking prevention
- [ ] Token truncation/extension attacks
- [ ] Concurrent session limits
- [ ] Session invalidation on logout
- [ ] Expired session cleanup

## Performance Testing Guidelines

### Benchmark Baselines

Establish performance baselines for critical operations:

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Payment initiation | < 10ms | p99 latency |
| Payment verification | < 50ms | p99 latency (includes DB transaction) |
| User login | < 100ms | p99 latency (bcrypt is slow) |
| Session validation | < 5ms | p99 latency |
| WebRTC token generation | < 10ms | p99 latency |

### Load Testing Scenarios

```go
func TestPaymentService_LoadTest_100ConcurrentPayments(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }

    db := setupTestDB()
    cfg := &config.Config{PlatformCommission: 20.0}
    service := services.NewPaymentService(db, cfg, nil)

    // Setup
    const numPayments = 100
    var wg sync.WaitGroup
    errors := make(chan error, numPayments)
    latencies := make(chan time.Duration, numPayments)

    // Execute
    for i := 0; i < numPayments; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            start := time.Now()
            _, err := service.InitiatePayment(uint(id), uint(id+1), 50.0)
            latencies <- time.Since(start)
            errors <- err
        }(i)
    }

    wg.Wait()
    close(errors)
    close(latencies)

    // Analyze
    var errorCount int
    for err := range errors {
        if err != nil {
            errorCount++
        }
    }

    var maxLatency time.Duration
    for latency := range latencies {
        if latency > maxLatency {
            maxLatency = latency
        }
    }

    // Assert
    assert.Equal(t, 0, errorCount, "No errors should occur under load")
    assert.Less(t, maxLatency, 100*time.Millisecond, "p100 latency under 100ms")
}
```

## Implementation Roadmap

### Phase 1: Foundation (Week 1)
- Add `testify` dependency
- Refactor existing tests to use testify assertions
- Add table-driven tests for existing service methods
- Setup coverage measurement in local dev

### Phase 2: Security Tests (Week 2)
- Create `tests/security/` directory
- Implement SQL injection tests
- Implement race condition tests (`go test -race`)
- Implement timing attack tests for auth/oauth
- Add session hijacking tests

### Phase 3: Integration Tests (Week 3)
- Create `tests/integration/` directory
- Write full auth flow tests (register → login → access)
- Write payment flow tests (initiate → verify → credit)
- Write OAuth flow tests (state → callback → session)
- Setup PostgreSQL + Redis in CI

### Phase 4: Load Tests (Week 4)
- Create `tests/load/` directory
- Write benchmarks for payment operations
- Write concurrent access tests
- Establish performance baselines
- Add load tests to CI (nightly)

### Phase 5: Coverage Improvement (Week 5-6)
- Achieve 80%+ overall coverage
- 95%+ coverage for security-critical services
- Add pre-commit hooks
- Enforce coverage in CI

### Phase 6: Continuous Maintenance
- Run tests on every commit
- Review and update security tests quarterly
- Update load test baselines as features grow
- Refactor tests as code evolves

## Tools and Resources

### Testing Libraries

```bash
# Core testing
go get github.com/stretchr/testify@v1.10.0

# HTTP testing (stdlib)
import "net/http/httptest"

# Coverage tools (stdlib)
go test -cover
go tool cover -html=coverage.out
```

### Security Resources

- OWASP Go Secure Coding Practices
- Go Security Checker: `gosec` (static analysis)
- Go vulnerability database: `govulncheck`

### CI/CD Templates

GitHub Actions templates provided in `CI/CD Integration` section above.

## Success Metrics

### Week 1
- [ ] Testify integrated
- [ ] All existing tests use testify assertions
- [ ] Coverage measurement automated

### Week 2
- [ ] Security test suite created
- [ ] SQL injection tests pass
- [ ] Race detector runs without errors
- [ ] Timing attack tests establish baselines

### Week 3
- [ ] Integration tests cover auth flow
- [ ] Integration tests cover payment flow
- [ ] Integration tests cover OAuth flow
- [ ] CI runs integration tests

### Week 4
- [ ] Benchmarks establish performance baselines
- [ ] Load tests identify bottlenecks
- [ ] Performance regression detection in CI

### Week 5-6
- [ ] 80%+ overall coverage achieved
- [ ] 95%+ coverage for payment/auth/oauth services
- [ ] Pre-commit hooks enforce test success
- [ ] Coverage threshold enforced in CI

### Ongoing
- [ ] All new code has tests (coverage doesn't drop)
- [ ] Security tests run on every PR
- [ ] Load tests run nightly
- [ ] Zero race conditions in production

## Sources

- Go testing package documentation: https://pkg.go.dev/testing (HIGH confidence)
- Go HTTP testing utilities: https://pkg.go.dev/net/http/httptest (HIGH confidence)
- Testify assertion library: https://github.com/stretchr/testify (MEDIUM confidence - inferred from ecosystem knowledge)
- GORM documentation for test patterns (MEDIUM confidence - based on current codebase patterns)
- OWASP security testing principles (MEDIUM confidence - general security knowledge applied to Go)
- Go race detector: https://go.dev/doc/articles/race_detector (HIGH confidence)

---
*Testing architecture research for: vk_backend*
*Researched: 2026-02-03*
*Confidence: HIGH - Based on official Go documentation, stdlib tools, and security best practices*
