# Testing Patterns

**Analysis Date:** 2026-02-03

## Test Framework

**Runner:**
- Go's built-in `testing` package
- Tests run via `make test` which executes `go test -v ./...`
- Coverage available via `make test-coverage` which generates `coverage.out` and `coverage.html`

**Assertion Library:**
- Manual assertions using `t.Errorf()`, `t.Fatalf()`, `t.Error()`
- No external assertion library (testify, etc.) - pure Go testing

**Run Commands:**
```bash
make test              # Run all tests with verbose output (go test -v ./...)
make test-coverage     # Run all tests and generate HTML coverage report
go test ./tests/unit   # Run only unit tests
go test -run TestName  # Run specific test by name
```

## Test File Organization

**Location:**
- Tests stored in `tests/unit/` directory (separate from source)
- All test files in single package `unit` (`package unit`)
- Not co-located with source code

**Naming:**
- Test files: `{service_name}_test.go` (e.g., `auth_service_test.go`, `advocate_service_test.go`)
- Test functions: `Test{FunctionName}{Scenario}` (e.g., `TestLoginUserA`, `TestRegisterUserA_InvalidEmail`)
- Setup functions: `setup{ComponentType}()` (e.g., `setupTestDB()`, `setupSessionStore()`)

**Structure:**
```
tests/
├── unit/
│   ├── setup_test.go              # Shared test fixtures and setup
│   ├── auth_service_test.go        # Auth service tests
│   ├── advocate_service_test.go    # Advocate service tests
│   ├── call_service_test.go        # Call service tests
│   ├── payment_service_test.go     # Payment service tests
│   ├── webrtc_service_test.go      # WebRTC service tests
│   ├── middleware_test.go          # Middleware tests
│   ├── signaling_service_test.go   # Signaling service tests
│   └── oauth_service_test.go       # OAuth service tests
```

## Test Structure

**Suite Organization:**
```go
func TestLoginUserA(t *testing.T) {
	db := setupTestDB()
	service := services.NewAuthService(db, setupSessionStore(db))

	// Arrange: Set up test data
	req := &models.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	_, err := service.RegisterAdvocate(req)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Act: Execute the function being tested
	user, err := service.LoginAdvocate("test@example.com", "password123")

	// Assert: Verify results
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected email, got %s", user.Email)
	}
}
```

**Patterns:**
- `t.Fatalf()`: Test setup failures (stop execution)
- `t.Errorf()`: Assertion failures (continue execution, collect all failures)
- `t.Error()`: General assertion failures
- Each test creates fresh database via `setupTestDB()`
- Each test instantiates required service fresh
- No shared state between tests (isolation enforced)

## Mocking

**Framework:** None - tests use real implementations with test doubles

**Patterns:**
- No mocking library used (no gomock, mockito, etc.)
- Real in-memory SQLite database for tests
- Fake/stub services passed where needed
- Example from `payment_service_test.go`:
  ```go
  func TestInitiatePayment(t *testing.T) {
    db := setupTestDB()
    cfg := setupTestConfig()  // Fake config with test values
    service := services.NewPaymentService(db, cfg, nil)  // nil for payment gateway

    // Real DB, real service logic, fake config
  }
  ```

**What to Mock:**
- External services: payment gateways, third-party APIs (pass nil for optional dependencies)
- Configuration: `setupTestConfig()` creates test-specific config
- Database: use in-memory SQLite instead of mocking GORM

**What NOT to Mock:**
- Database layer: real SQLite in-memory DB ensures ORM interactions work correctly
- Service layer: test real business logic, not mocks
- Request/response structures: test with real models

## Fixtures and Factories

**Test Data:**
Inline model creation pattern, no factory functions:
```go
func TestGetAdvocateSchedule(t *testing.T) {
	db := setupTestDB()

	// Direct struct construction
	advocate := models.Advocate{Name: "Advocate", Email: "a@test.com", UUID: "a-uuid"}
	db.Create(&advocate)

	// Slice construction
	calls := []models.Call{
		{
			CallerID:     100,
			ReceiverID:   advocate.ID,
			ReceiverType: "advocate",
			Status:       "scheduled",
			ScheduledAt:  &later,
		},
	}
	db.Create(&calls)
}
```

**Location:**
- Shared setup in `tests/unit/setup_test.go`:
  ```go
  func setupTestDB() *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    // ... configure and migrate ...
    return db
  }

  func setupSessionStore(db *gorm.DB) sessions.Store {
    store, _, err := sessions.NewStore(db, nil, "db")
    // ... return configured store ...
  }
  ```

**Test Config Fixture:**
From `payment_service_test.go`:
```go
func setupTestConfig() *config.Config {
	return &config.Config{
		PlatformCommission: 20.0, // 20% commission for testing
	}
}
```

## Coverage

**Requirements:** No specific coverage target enforced (optional)

**View Coverage:**
```bash
make test-coverage              # Generates coverage.out and coverage.html
open coverage.html              # View interactive HTML report in browser
go tool cover -html=coverage.out # Manually generate HTML
go tool cover -func=coverage.out # View function-level coverage summary
```

## Test Types

**Unit Tests:**
- Scope: Single service method or function
- Approach: Direct instantiation of service with test DB and config
- Example: `TestRegisterUserA` tests `AuthService.RegisterAdvocate()` in isolation
- Located in: `tests/unit/*.go`

**Integration Tests:**
- Scope: Multiple service methods or endpoint flows
- Approach: Sequence of operations (register → login → use)
- Example: `TestLoginUserA` first registers then logs in
- Located in: Same `tests/unit/` directory (not separated)

**E2E Tests:**
- Framework: Not present
- Would test full HTTP request → response cycle
- Current tests are pre-HTTP layer (services only)

## Common Patterns

**Async Testing:**
Not used - synchronous operations throughout
```go
// All calls are synchronous
user, err := service.LoginAdvocate(email, password)
if err != nil {
  t.Fatalf("unexpected error: %v", err)
}
```

**Error Testing:**
Two patterns for expected errors:

1. **Expect non-nil error:**
   ```go
   _, err := service.VerifyPayment(invalidID)
   if err == nil {
       t.Fatalf("expected error for invalid ID")
   }
   ```

2. **Expect specific error type:**
   ```go
   if err != nil && !strings.Contains(err.Error(), "not found") {
       t.Errorf("wrong error: %v", err)
   }
   ```

3. **Test error path with assertion:**
   ```go
   _, err := service.RegisterAdvocate(&models.RegisterRequest{
       Email: "", // Invalid
   })
   if err == nil {
       t.Fatalf("expected error for empty email")
   }
   ```

**Table-Driven Tests (Sub-tests):**
Pattern from `middleware_test.go`:
```go
func TestAuthMiddleware(t *testing.T) {
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test logic using tc fields
		})
	}
}
```

## Test Coverage Observations

**Well-Tested Areas:**
- Core business logic in services (auth, advocates, calls, payments)
- Error validation (empty fields, invalid IDs)
- Authorization checks (only receiver can accept call)
- State transitions (initiated → accepted → completed)

**Test Gaps:**
- No middleware tests beyond auth
- No handler/routing tests (no gin test context setup)
- No idempotency tests
- No outbox/event tests
- No WebSocket or signaling tests (services exist but not tested)
- No Redis/caching tests
- No OpenTelemetry/observability tests

---

*Testing analysis: 2026-02-03*
