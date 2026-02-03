# Coding Conventions

**Analysis Date:** 2026-02-03

## Naming Patterns

**Files:**
- Go source files: lowercase with underscores (e.g., `auth_service.go`, `payment_service.go`)
- Test files: `{name}_test.go` (e.g., `auth_service_test.go`)
- Package names: lowercase, match directory name (e.g., `package services`, `package models`)

**Functions:**
- Exported functions: PascalCase (e.g., `RegisterAdvocate`, `LoginClient`, `GetProfile`)
- Unexported functions: camelCase (e.g., `generateToken`, `setupTestDB`)
- Service constructors: `New{ServiceName}` pattern (e.g., `NewAuthService`, `NewAdvocateService`)
- Handler functions: match HTTP method + resource (e.g., `HandlePaymentInitiate`)

**Variables:**
- Local variables: camelCase (e.g., `advocateID`, `sessionStore`, `existingAdvocate`)
- Constants: SCREAMING_SNAKE_CASE or used in maps (e.g., `"available": true`, `"offline": true`)
- Private struct fields: camelCase (e.g., `db`, `logger`, `sessionStore`)
- Public struct fields: PascalCase with JSON tags (e.g., `ID`, `Email`, `Name`)

**Types:**
- Struct names: PascalCase (e.g., `AuthService`, `AdvocateService`, `PaymentService`)
- Interface names: PascalCase ending with "er" when appropriate (e.g., `Store` interface in sessions)
- Request/response structs: PascalCase with suffix (e.g., `RegisterRequest`, `LoginRequest`)

## Code Style

**Formatting:**
- Standard Go fmt tool via `make fmt` (runs `go fmt ./...`)
- No custom formatting rules detected
- Line length: standard Go convention (~120 chars implied)

**Linting:**
- No `.eslintrc`, `golangci.yml`, or similar config found
- Code follows standard Go conventions by observation
- Recommend adding golangci-lint for consistency

**Import Organization:**
- Standard Go convention: imports grouped and sorted
- Order: standard library, then external packages
- Example from `auth_service.go`:
  ```go
  import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "fmt"
    "time"

    "vk_backend/internal/logger"
    "vk_backend/internal/models"
    "vk_backend/internal/sessions"

    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
  )
  ```

**Path Aliases:**
- No import aliases used
- Full paths always used (e.g., `vk_backend/internal/models`)

## Error Handling

**Patterns:**
- Explicit error checking with `if err != nil` blocks
- Sentinel errors via `errors.New()` for domain errors
- Type assertion for specific errors: `if err == gorm.ErrRecordNotFound`
- Error wrapping with `fmt.Errorf("message: %w", err)` for context
- Generic "database error" returned to API layer (no details leaked)
- Example from `auth_service.go`:
  ```go
  var existingAdvocate models.Advocate
  if err := s.db.Where("email = ?", req.Email).First(&existingAdvocate).Error; err == nil {
    s.logger.Info("RegisterAdvocate failed: email already exists: %s", req.Email)
    return nil, errors.New("email already registered")
  }
  ```

**Error Propagation:**
- Services return `(*Type, error)` tuple
- Handlers convert service errors to HTTP status codes
- No panic() usage in production code

## Logging

**Framework:** Custom logger in `internal/logger/logger.go`

**Levels (from highest to lowest priority):**
- `SEVERE`: Critical errors (production failures)
- `INFO`: General informational messages (state changes, business events)
- `FINER`: Fine-grained informational messages (method entry points, parameters)
- `FINEST`: Finest-grained messages (detailed trace)
- `DEBUG`: Debug-level messages

**Patterns:**
- Create logger per service: `logger := logger.NewLogger("ServiceName", logger.INFO)`
- Log at entry point with parameters: `s.logger.Finer("RegisterAdvocate called for email: %s", req.Email)`
- Log validation failures at INFO: `s.logger.Info("RegisterAdvocate failed: missing required fields")`
- Log errors at SEVERE: `s.logger.Severe("RegisterAdvocate failed: password hashing error: %v", err)`
- Example from `auth_service.go`:
  ```go
  s.logger.Finer("RegisterAdvocate called for email: %s", req.Email)
  // ... validation ...
  s.logger.Info("RegisterAdvocate failed: missing required fields")
  // ... processing ...
  s.logger.Severe("RegisterAdvocate failed: password hashing error: %v", err)
  s.logger.Info("Advocate registered successfully: ID=%d, Email=%s", advocate.ID, advocate.Email)
  ```

## Comments

**When to Comment:**
- Business logic that isn't obvious: comments explain "why", not "what"
- Example from `auth_service.go`:
  ```go
  // Generate a secure token (in production, use JWT or similar)
  token, err := generateToken()
  ```
- Marker comments for important checks:
  ```go
  // Hash password
  hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
  ```

**JSDoc/TSDoc:**
- Not used (Go convention is to write comments above exported functions)
- Example pattern (inferred from code organization):
  ```go
  // RegisterAdvocate creates a new advocate account with the given details.
  func (s *AuthService) RegisterAdvocate(req *models.RegisterRequest) (*models.Advocate, error) {
  ```

## Function Design

**Size:** Generally 20-50 lines per function
- Validation at start
- Business logic in middle
- Return at end
- Early returns for error cases

**Parameters:**
- Single receiver for methods: `func (s *AuthService) Method(...)`
- Minimal parameters: validated at function entry
- Request objects used: `*models.RegisterRequest` instead of multiple primitives
- Optional config passed via struct: `*config.Config`

**Return Values:**
- Two-return pattern: `(*Type, error)` or `(Type, error)`
- Zero values returned on success: no nil checks needed for happy path
- Pointers for structs: `*models.Advocate` (nil on error indicates failure)

## Module Design

**Exports:**
- Constructor functions: `NewServiceName(deps...) *ServiceName`
- Public methods: start with capital letter
- Private methods: start with lowercase letter
- Struct fields exposed when needed for JSON marshaling

**Barrel Files:**
- Models package has `models.go` with all struct definitions
- No re-exports or aliases used

**Package Structure:**
- `internal/services/`: Business logic (auth, advocates, calls, payments)
- `internal/models/`: Data structures and GORM models
- `internal/middleware/`: HTTP middleware
- `internal/logger/`: Logging utility
- `internal/config/`: Configuration management
- `internal/database/`: DB initialization and migrations
- `tests/unit/`: Unit tests

## Handler Patterns (from main.go)

**HTTP Routes:**
- RESTful pattern: `/api/{resource}/{action}`
- POST for state-changing operations: `/api/advocate/register`, `/api/calls/initiate`
- PUT for updates: `/api/advocate/availability`, `/api/client/profile-image`
- GET for retrieval: `/api/advocate/profile`, `/api/advocate/earnings`
- No versioning in URL

**Handler Structure:**
- Inline handlers in `cmd/main.go` using gin closures
- Consistent pattern:
  1. Extract user info from context (if authenticated)
  2. Parse request body/params
  3. Call service method
  4. Handle error -> HTTP error response
  5. Return success with JSON response
- Example pattern:
  ```go
  auth.POST("/advocate/register", func(c *gin.Context) {
    var req models.RegisterRequest
    if err := c.BindJSON(&req); err != nil {
      c.JSON(400, gin.H{"error": "invalid request"})
      return
    }
    user, err := authService.RegisterAdvocate(&req)
    if err != nil {
      c.JSON(400, gin.H{"error": err.Error()})
      return
    }
    c.JSON(200, user)
  })
  ```

---

*Convention analysis: 2026-02-03*
