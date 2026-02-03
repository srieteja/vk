# Security Stack Research

**Domain:** Go API Security (Payments + Authentication)
**Researched:** 2026-02-03
**Confidence:** HIGH

## Executive Summary

Production-ready security patterns for hardening existing Go backend with payment processing and dual-authentication. Stack recommendations address 7 identified vulnerabilities: hardcoded credentials, SQL injection, missing rate limiting, weak session validation, missing input sanitization, OAuth timing attacks, and payment race conditions.

All recommended libraries are compatible with existing Gin/GORM/Redis stack (Go 1.24, Gin 1.9.1, GORM 1.30.0, go-redis/v9).

---

## Core Security Libraries

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| golang.org/x/crypto | v0.47.0+ | Password hashing, timing-safe comparison | Industry standard, includes bcrypt (already in use), argon2, and crypto/subtle for timing-safe comparisons. Prevents timing attacks in OAuth. |
| github.com/go-playground/validator/v10 | v10.14.0+ | Input validation | Already in dependencies. Provides struct-based validation for preventing XSS/SSRF. Validates email, URL, JSON formats before processing. |
| github.com/ulule/limiter/v3 | v3.11.2 | Rate limiting with Redis backend | Dead-simple API, Redis-backed (matches existing infrastructure), prevents brute force on auth endpoints. Supports per-IP and per-user limits. |
| github.com/microcosm-cc/bluemonday | v1.0.27 | HTML sanitization | Industry-standard XSS prevention. Sanitizes user-generated content before storage/display. Used by GitHub, GitLab. |
| github.com/gin-contrib/cors | v1.7.2 | CORS security | Official Gin middleware. Prevents cross-origin attacks with credential-aware origin validation. Critical for payment endpoints. |

---

## Security Patterns by Vulnerability

### 1. Hardcoded Credentials in Repository

**Current Issue:** `.env` file with credentials committed to git (DATABASE_URL, JWT_SECRET, payment gateway keys).

**Solution Stack:**

| Component | Purpose | Implementation |
|-----------|---------|----------------|
| .gitignore | Prevent credential commits | Add `.env` and `.env.local` to gitignore |
| .env.example | Document required variables | Template with placeholder values (`your_secret_here`) |
| godotenv (existing) | Development environment only | Use ONLY in development. Load order: OS env → .env.local → .env |
| Infrastructure secrets | Production credential management | Use Kubernetes secrets, AWS Systems Manager, Docker secrets, or environment-specific deployment configs |

**Pattern:**
```go
// config/config.go - Fail fast on missing secrets in production
func LoadConfig() *Config {
    env := getEnv("ENVIRONMENT", "development")

    // Development: Load from .env
    if env == "development" {
        godotenv.Load(".env.local")
        godotenv.Load()
    }

    // Production: Require environment variables
    jwtSecret := os.Getenv("JWT_SECRET")
    if env == "production" && jwtSecret == "" {
        log.Fatal("JWT_SECRET must be set in production")
    }

    // NEVER use default secrets in production
    if env == "production" && jwtSecret == "secret" {
        log.Fatal("Default JWT_SECRET is not allowed in production")
    }

    return &Config{JWTSecret: jwtSecret}
}
```

**Rationale:** Godotenv is designed for development only. Production systems should receive credentials through infrastructure (env vars injected by orchestrator), not file-based config. This prevents accidental credential exposure in version control.

---

### 2. SQL Injection Vulnerability

**Current Issue:** LIKE clause with string concatenation in `advocate_service.go:142`:
```go
query.Where("LOWER(location) LIKE ?", "%"+filter.Location+"%")
```

**Risk:** While GORM parameterizes the placeholder `?`, the pattern creates logical injection risk if `filter.Location` contains `%` or `_` wildcards.

**Solution Stack:**

| Component | Purpose | Why |
|-----------|---------|-----|
| GORM parameterization | SQL injection prevention | Always use `Where(query, args)` pattern, never string concatenation. GORM's database driver handles escaping. |
| validator/v10 | Input validation | Validate location format before query (alphanumeric + spaces + common chars). |
| LIKE wildcard escaping | Logical injection prevention | Escape user-provided `%` and `_` characters in LIKE patterns. |

**Safe Pattern:**
```go
// 1. Validate input first
type AdvocateFilter struct {
    Location string `validate:"omitempty,alphanum|contains= |contains=-"`
}

validate := validator.New()
if err := validate.Struct(filter); err != nil {
    return nil, errors.New("invalid filter")
}

// 2. Escape LIKE wildcards
func escapeLikeWildcards(s string) string {
    s = strings.ReplaceAll(s, `\`, `\\`)
    s = strings.ReplaceAll(s, `%`, `\%`)
    s = strings.ReplaceAll(s, `_`, `\_`)
    return s
}

// 3. Use parameterized query
location := escapeLikeWildcards(filter.Location)
query = query.Where("LOWER(location) LIKE ?", "%"+strings.ToLower(location)+"%")
```

**NEVER Do This:**
```go
// UNSAFE - String concatenation (even with GORM)
db.Where("name = " + userInput)

// UNSAFE - fmt.Sprintf injection
db.Where(fmt.Sprintf("location LIKE '%%%s%%'", userInput))

// UNSAFE - Raw SQL without parameters
db.Raw("SELECT * FROM advocates WHERE location = " + userInput)
```

**Rationale:** GORM prevents SQL injection through parameterized queries, but LIKE patterns need additional wildcard escaping to prevent logical injection (e.g., `filter.Location = "%"` matches all records).

---

### 3. No Rate Limiting on Auth Endpoints

**Current Issue:** Login/register endpoints have no rate limiting, enabling brute force attacks.

**Solution Stack:**

| Component | Version | Purpose | Why |
|-----------|---------|---------|-----|
| ulule/limiter/v3 | v3.11.2 | Rate limiting | Redis-backed, Gin middleware, handles distributed rate limiting across instances |
| golang.org/x/time/rate | v0.14.0 (existing) | In-memory fallback | Already in dependencies for LLM rate limiting. Use as fallback if Redis unavailable |

**Installation:**
```bash
go get github.com/ulule/limiter/v3@v3.11.2
go get github.com/ulule/limiter/v3/drivers/store/redis
go get github.com/ulule/limiter/v3/drivers/middleware/gin
```

**Pattern (Redis-Backed):**
```go
import (
    "github.com/ulule/limiter/v3"
    limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
    limiterredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// Auth endpoints: 5 requests per minute per IP
authRate, _ := limiter.NewRateFromFormatted("5-M")
authStore, _ := limiterredis.NewStoreWithOptions(redisClient, limiter.StoreOptions{
    Prefix: "ratelimit:auth:",
})
authLimiter := limiter.New(authStore, authRate, limiter.WithTrustForwardHeader(true))
authMiddleware := limitergin.NewMiddleware(authLimiter)

// Apply to auth routes
router.POST("/api/auth/login", authMiddleware, authHandler.Login)
router.POST("/api/auth/register", authMiddleware, authHandler.Register)

// Payment endpoints: 10 requests per minute per user
paymentRate, _ := limiter.NewRateFromFormatted("10-M")
paymentStore, _ := limiterredis.NewStoreWithOptions(redisClient, limiter.StoreOptions{
    Prefix: "ratelimit:payment:",
})
paymentLimiter := limiter.New(paymentStore, paymentRate)

// Custom key function (user-based, not IP-based)
paymentMiddleware := func(c *gin.Context) {
    userID := c.GetString("user_id")
    ctx := limiter.NewContext(c.Request.Context())
    ctx, err := paymentLimiter.Get(c.Request.Context(), userID)
    if err != nil || ctx.Reached {
        c.JSON(429, gin.H{"error": "rate limit exceeded"})
        c.Abort()
        return
    }
    c.Next()
}

router.POST("/api/payments/initiate", authMiddleware, paymentMiddleware, paymentHandler.Initiate)
```

**Fallback Pattern (In-Memory):**
```go
// Use existing golang.org/x/time/rate for single-instance deployments
import "golang.org/x/time/rate"

type InMemoryRateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func (r *InMemoryRateLimiter) getLimiter(key string) *rate.Limiter {
    r.mu.RLock()
    limiter, exists := r.limiters[key]
    r.mu.RUnlock()

    if exists {
        return limiter
    }

    r.mu.Lock()
    defer r.mu.Unlock()
    limiter = rate.NewLimiter(r.rate, r.burst)
    r.limiters[key] = limiter
    return limiter
}

func RateLimitMiddleware(limiter *InMemoryRateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.ClientIP()
        if !limiter.getLimiter(key).Allow() {
            c.JSON(429, gin.H{"error": "rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**Rationale:** ulule/limiter provides Redis-backed distributed rate limiting (critical for multi-instance deployments), Gin integration, and IP extraction with proxy awareness. golang.org/x/time/rate works for single-instance or development environments.

---

### 4. Weak Session Validation

**Current Issue:** Session validation only checks token existence and expiry. Missing: session rotation, secure token generation, CSRF protection.

**Solution Stack:**

| Component | Purpose | Implementation |
|-----------|---------|----------------|
| crypto/rand | Secure token generation | Already used in `generateToken()`. Ensures cryptographically secure randomness (32 bytes = 256 bits). |
| Session rotation | Prevent session fixation | Regenerate session token after login, privilege change. |
| HTTP-only cookies | XSS prevention | Set session token in HTTP-only cookie (not localStorage). |
| SameSite attribute | CSRF prevention | Set `SameSite=Strict` or `SameSite=Lax` on session cookies. |

**Enhanced Pattern:**
```go
// 1. Secure token generation (already implemented correctly)
func generateToken() (string, error) {
    b := make([]byte, 32) // 256 bits
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("failed to generate secure token: %w", err)
    }
    return hex.EncodeToString(b), nil
}

// 2. Session rotation after login
func (s *AuthService) LoginAdvocate(email, password string) (*models.Session, error) {
    advocate, err := s.authenticateAdvocate(email, password)
    if err != nil {
        return nil, err
    }

    // Invalidate old sessions (prevent session fixation)
    s.sessionStore.DeleteByUserID(context.Background(), advocate.ID, "advocate")

    // Create new session
    return s.CreateSession(advocate.ID, "advocate")
}

// 3. HTTP-only cookie middleware
func SetSessionCookie(c *gin.Context, token string, expiry time.Time) {
    c.SetCookie(
        "session_token",           // name
        token,                     // value
        int(time.Until(expiry).Seconds()), // maxAge
        "/",                       // path
        "",                        // domain (empty = current)
        true,                      // secure (HTTPS only)
        true,                      // httpOnly (no JavaScript access)
    )
    c.SetSameSite(http.SameSiteStrictMode) // CSRF protection
}

// 4. Enhanced auth middleware
func AuthMiddleware(store sessions.Store) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Try cookie first (preferred)
        token, err := c.Cookie("session_token")
        if err != nil {
            // Fallback to Authorization header (for API clients)
            authHeader := c.GetHeader("Authorization")
            if authHeader == "" {
                c.JSON(401, gin.H{"error": "missing credentials"})
                c.Abort()
                return
            }

            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                c.JSON(401, gin.H{"error": "invalid authorization format"})
                c.Abort()
                return
            }
            token = parts[1]
        }

        session, err := store.GetByToken(c.Request.Context(), token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid or expired session"})
            c.Abort()
            return
        }

        c.Set("user_id", session.UserID)
        c.Set("user_type", session.UserType)
        c.Next()
    }
}
```

**Session Store Enhancement:**
```go
type Store interface {
    Create(ctx context.Context, session *models.Session) error
    GetByToken(ctx context.Context, token string) (*models.Session, error)
    DeleteByToken(ctx context.Context, token string) error
    DeleteByUserID(ctx context.Context, userID uint, userType string) error // Invalidate all user sessions
}
```

**Rationale:** Secure session management requires multiple layers: cryptographically secure tokens (crypto/rand), session rotation (prevent fixation), HTTP-only cookies (prevent XSS theft), and SameSite attribute (prevent CSRF). The existing hybrid Redis/DB session store is solid; enhancements focus on rotation and cookie security.

---

### 5. Missing Input Sanitization

**Current Issue:** No validation on user inputs (email, name, profile image URLs). Risk: XSS, SSRF, malformed data.

**Solution Stack:**

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| validator/v10 | v10.14.0 (existing) | Format validation | Validate ALL inputs before processing. Prevents malformed data. |
| bluemonday | v1.0.27 | HTML sanitization | Sanitize user-generated content (bio, comments). Prevents stored XSS. |
| net/url (stdlib) | - | URL validation | Validate profile image URLs, prevent SSRF attacks. |

**Installation:**
```bash
go get github.com/microcosm-cc/bluemonday@v1.0.27
```

**Validation Pattern:**
```go
import (
    "github.com/go-playground/validator/v10"
    "github.com/microcosm-cc/bluemonday"
    "net/url"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// 1. Input validation (registration)
type RegisterRequest struct {
    Email    string `json:"email" validate:"required,email,max=255"`
    Password string `json:"password" validate:"required,min=8,max=72"` // bcrypt max 72 bytes
    Name     string `json:"name" validate:"required,min=2,max=100,excludesall=<>"`
}

func (s *AuthService) RegisterAdvocate(req *RegisterRequest) (*models.Advocate, error) {
    // Validate input structure
    if err := validate.Struct(req); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Sanitize name (prevent XSS in display)
    policy := bluemonday.StrictPolicy() // Strips ALL HTML
    req.Name = policy.Sanitize(req.Name)

    // ... rest of registration logic
}

// 2. URL validation (profile image upload)
func (s *AuthService) UpdateProfileImage(clientID uint, imageURL string) error {
    // Validate URL format
    if err := validate.Var(imageURL, "required,url"); err != nil {
        return errors.New("invalid image URL")
    }

    // Parse and validate URL scheme (prevent SSRF)
    parsedURL, err := url.Parse(imageURL)
    if err != nil {
        return errors.New("malformed URL")
    }

    // Only allow HTTPS from trusted domains
    if parsedURL.Scheme != "https" {
        return errors.New("only HTTPS URLs allowed")
    }

    // Prevent localhost/internal network SSRF
    if isPrivateIP(parsedURL.Hostname()) {
        return errors.New("private IP addresses not allowed")
    }

    // Update database
    return s.db.Model(&models.Client{}).Where("id = ?", clientID).
        Update("profile_image", imageURL).Error
}

func isPrivateIP(hostname string) bool {
    ip := net.ParseIP(hostname)
    if ip == nil {
        return false // Not an IP, assume domain is safe (validate domain separately)
    }
    return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// 3. HTML sanitization (rich text content)
func SanitizeHTML(input string) string {
    policy := bluemonday.UGCPolicy() // Allows safe HTML subset (links, emphasis, etc.)
    policy.AllowAttrs("class").Matching(regexp.MustCompile("^[a-zA-Z0-9-_]+$")).OnElements("span", "div")
    return policy.Sanitize(input)
}
```

**Custom Validators:**
```go
// Register custom validators for domain-specific rules
func init() {
    validate.RegisterValidation("no_sql_keywords", func(fl validator.FieldLevel) bool {
        value := strings.ToUpper(fl.Field().String())
        keywords := []string{"SELECT", "DROP", "DELETE", "INSERT", "UPDATE", "UNION"}
        for _, kw := range keywords {
            if strings.Contains(value, kw) {
                return false
            }
        }
        return true
    })
}

type SearchFilter struct {
    Location string `validate:"omitempty,max=100,no_sql_keywords"`
}
```

**Rationale:** validator/v10 provides format validation (first line of defense). bluemonday sanitizes HTML content (prevents stored XSS). URL validation prevents SSRF attacks where attacker-controlled URLs could access internal services. Layered validation (format → sanitize → context-specific checks) is critical.

---

### 6. Timing Attacks in OAuth

**Current Issue:** `ValidateState()` in `oauth_service.go:262` uses `hmac.Equal()` correctly, but other string comparisons (state format, expiry checks) are not timing-safe.

**Solution Stack:**

| Component | Purpose | Implementation |
|-----------|---------|----------------|
| crypto/subtle | Timing-safe comparison | Use `subtle.ConstantTimeCompare()` for all security-critical string comparisons. |
| hmac.Equal (existing) | Timing-safe HMAC comparison | Already used correctly for signature validation. |

**Pattern:**
```go
import "crypto/subtle"

// CURRENT (line 262) - Correct for HMAC
if !hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
    return "", errors.New("invalid state signature")
}

// ADDITIONAL - Use subtle.ConstantTimeCompare for other sensitive comparisons
func (s *OAuthService) ValidateState(state string) (string, error) {
    parts := strings.Split(state, ".")
    if len(parts) != 2 {
        return "", errors.New("invalid state format")
    }

    data, err := base64.RawURLEncoding.DecodeString(parts[0])
    if err != nil {
        return "", errors.New("invalid state encoding")
    }

    // Timing-safe signature comparison (already correct)
    expectedSig := s.signState(data)
    if !hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
        return "", errors.New("invalid state signature")
    }

    var payload oauthState
    if err := json.Unmarshal(data, &payload); err != nil {
        return "", errors.New("invalid state payload")
    }

    // Validate payload fields exist
    if payload.UserType == "" || payload.ExpiresAt == 0 || payload.IssuedAt == 0 {
        return "", errors.New("invalid state payload")
    }

    // Timing-safe user type validation (prevent user type enumeration)
    validTypes := []string{"advocate", "client"}
    validType := false
    for _, vt := range validTypes {
        if subtle.ConstantTimeCompare([]byte(payload.UserType), []byte(vt)) == 1 {
            validType = true
            break
        }
    }
    if !validType {
        return "", errors.New("invalid user type")
    }

    // Expiry check (timing variation acceptable here)
    if time.Now().Unix() > payload.ExpiresAt {
        return "", errors.New("state expired")
    }

    return payload.UserType, nil
}
```

**When to Use Timing-Safe Comparison:**

| Comparison Type | Use subtle.ConstantTimeCompare? | Reason |
|----------------|--------------------------------|--------|
| Password hashes | NO (use bcrypt.CompareHashAndPassword) | bcrypt already timing-safe |
| HMAC signatures | YES (use hmac.Equal) | Prevents signature forgery timing attacks |
| API keys | YES | Prevents key enumeration |
| Session tokens | YES | Prevents token enumeration |
| User types, roles | YES | Prevents role enumeration |
| Expiry timestamps | NO | Timing variation reveals no secrets |
| Public data | NO | No security impact |

**Rationale:** Timing attacks exploit variations in comparison time to guess secrets byte-by-byte. `subtle.ConstantTimeCompare()` ensures constant-time execution regardless of input, preventing these attacks. OAuth state validation already uses `hmac.Equal()` correctly for signatures; extend to user type validation to prevent role enumeration.

---

### 7. Payment Race Conditions

**Current Issue:** `VerifyPayment()` in `payment_service.go` uses database transaction correctly but missing idempotency key validation for webhook retries.

**Solution Stack:**

| Component | Purpose | Implementation |
|-----------|---------|----------------|
| GORM transactions (existing) | Atomic updates | Already implemented correctly. Ensures advocate earnings + payment status update atomically. |
| Idempotency middleware (existing) | Duplicate prevention | Extend existing `idempotency` package to payment verification. |
| Optimistic locking | Concurrent update prevention | Add `version` column to payments table. |

**Enhanced Pattern:**
```go
// 1. Add version column for optimistic locking
type Payment struct {
    ID                 uint
    Status             string
    Version            int  `gorm:"default:1"` // NEW: Optimistic lock
    // ... other fields
}

// 2. Payment verification with optimistic locking
func (s *PaymentService) VerifyPayment(ctx context.Context, txnID string, idempotencyKey string) (*models.Payment, error) {
    // Check idempotency (prevent duplicate processing)
    if s.idempotencyStore != nil {
        processed, err := s.idempotencyStore.Exists(ctx, idempotencyKey)
        if err != nil {
            return nil, errors.New("idempotency check failed")
        }
        if processed {
            // Return existing payment (webhook retry)
            return s.getPaymentByIdempotencyKey(ctx, idempotencyKey)
        }
    }

    var payment models.Payment
    if err := s.db.Where("transaction_id = ?", txnID).First(&payment).Error; err != nil {
        return nil, err
    }

    if payment.Status == "completed" {
        return &payment, nil // Already processed
    }

    // Atomic transaction with optimistic locking
    err := s.db.Transaction(func(tx *gorm.DB) error {
        // Update payment with version check (optimistic lock)
        result := tx.Model(&models.Payment{}).
            Where("id = ? AND version = ?", payment.ID, payment.Version). // Lock check
            Updates(map[string]interface{}{
                "status":  "completed",
                "version": gorm.Expr("version + 1"), // Increment version
            })

        if result.Error != nil {
            return result.Error
        }

        if result.RowsAffected == 0 {
            // Version mismatch = concurrent update detected
            return errors.New("payment was modified concurrently, retry")
        }

        // Update advocate earnings
        if err := tx.Model(&models.Advocate{}).Where("id = ?", payment.AdvocateID).
            UpdateColumn("earnings", gorm.Expr("earnings + ?", payment.AdvocateCommission)).Error; err != nil {
            return err
        }

        // Store idempotency key
        if s.idempotencyStore != nil {
            if err := s.idempotencyStore.Set(ctx, idempotencyKey, payment.ID, 24*time.Hour); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    payment.Status = "completed"
    payment.Version++
    return &payment, nil
}

// 3. Webhook handler with idempotency
func (h *PaymentHandler) StripeWebhook(c *gin.Context) {
    // Extract idempotency key from Stripe header
    idempotencyKey := c.GetHeader("Stripe-Signature")
    if idempotencyKey == "" {
        c.JSON(400, gin.H{"error": "missing signature"})
        return
    }

    // Parse webhook payload
    var event StripeEvent
    if err := c.ShouldBindJSON(&event); err != nil {
        c.JSON(400, gin.H{"error": "invalid payload"})
        return
    }

    // Verify payment with idempotency
    payment, err := h.paymentService.VerifyPayment(
        c.Request.Context(),
        event.Data.Object.ID,
        idempotencyKey,
    )

    if err != nil {
        c.JSON(500, gin.H{"error": "verification failed"})
        return
    }

    c.JSON(200, payment)
}
```

**Idempotency Store Interface:**
```go
type IdempotencyStore interface {
    Exists(ctx context.Context, key string) (bool, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Get(ctx context.Context, key string) (interface{}, error)
}

// Redis implementation (extends existing internal/idempotency package)
type RedisIdempotencyStore struct {
    client *redis.Client
    prefix string
}

func (s *RedisIdempotencyStore) Exists(ctx context.Context, key string) (bool, error) {
    exists, err := s.client.Exists(ctx, s.prefix+key).Result()
    return exists > 0, err
}

func (s *RedisIdempotencyStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return s.client.Set(ctx, s.prefix+key, data, ttl).Err()
}
```

**Rationale:** Existing GORM transaction is correct for atomicity. Adding optimistic locking (version column) prevents concurrent updates when multiple webhook deliveries arrive simultaneously. Idempotency middleware (already exists in codebase) prevents duplicate payment processing. Redis-backed idempotency store ensures distributed systems handle retries correctly.

---

## Additional Security Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/google/uuid | v1.6.0 (existing) | Secure UUID generation | Already used correctly. Generates V4 UUIDs for user IDs, nonces. |
| github.com/gin-contrib/secure | v1.0.0 | Security headers middleware | Adds security headers (X-Frame-Options, X-Content-Type-Options, HSTS). Use in production. |
| github.com/rs/cors | v1.11.1 | Alternative CORS (if needed) | More configurable than gin-contrib/cors. Use if need dynamic origin validation. |

---

## Installation Commands

```bash
# Core security libraries
go get github.com/ulule/limiter/v3@v3.11.2
go get github.com/ulule/limiter/v3/drivers/store/redis
go get github.com/ulule/limiter/v3/drivers/middleware/gin
go get github.com/microcosm-cc/bluemonday@v1.0.27
go get github.com/gin-contrib/cors@v1.7.2

# Optional security headers
go get github.com/gin-contrib/secure@v1.0.0
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|------------------------|
| ulule/limiter/v3 | tollbooth/v7 | If need more complex limiting logic (headers, methods, custom keys). Heavier dependency. |
| ulule/limiter/v3 | golang.org/x/time/rate | Single-instance deployments or development. No Redis dependency. |
| bluemonday | html/template (stdlib) | If ONLY displaying HTML (not storing). Template auto-escapes output. |
| validator/v10 | ozzo-validation | If prefer fluent API over struct tags. More verbose. |
| gin-contrib/cors | rs/cors | If need dynamic origin callbacks with complex logic. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| JWT for sessions | Stateless JWTs can't be revoked. Session fixation risk. | Redis-backed session tokens (already implemented) |
| MD5/SHA1 for passwords | Cryptographically broken. Fast = vulnerable to brute force. | bcrypt (already in use) or argon2 |
| Custom crypto | High risk of implementation errors. | Use stdlib crypto packages or vetted libraries |
| regexp for email validation | Incomplete, bypassable. | validator/v10 `email` tag |
| String comparison for secrets | Timing attack vulnerable. | crypto/subtle.ConstantTimeCompare |
| .env files in production | Accidental credential exposure. | Infrastructure secrets (K8s, AWS SSM) |

---

## Security Configuration Checklist

### Development Environment
- [ ] `.env` and `.env.local` in `.gitignore`
- [ ] `.env.example` with placeholder values committed
- [ ] godotenv loads `.env` files
- [ ] Validation enabled but lenient

### Production Environment
- [ ] NO `.env` files (use infrastructure secrets)
- [ ] Fail fast on missing required secrets
- [ ] Reject default/weak secrets (e.g., `JWT_SECRET=secret`)
- [ ] Rate limiting enabled (Redis-backed)
- [ ] CORS configured with specific allowed origins
- [ ] HTTP-only, Secure, SameSite cookies
- [ ] TLS/HTTPS enforced
- [ ] Security headers middleware enabled
- [ ] Input validation on ALL endpoints
- [ ] HTML sanitization on user-generated content
- [ ] Optimistic locking on payment updates
- [ ] Idempotency keys on payment webhooks

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| ulule/limiter/v3@v3.11.2 | go-redis/v9 v9.17.2 | Requires v9+ of go-redis. Already compatible. |
| validator/v10@v10.14.0 | Gin v1.9.1 | Already in dependencies. No conflicts. |
| bluemonday@v1.0.27 | Go 1.24 | No special requirements. |
| gin-contrib/cors@v1.7.2 | Gin v1.9.1 | Official Gin middleware. Fully compatible. |
| golang.org/x/crypto@v0.47.0+ | Go 1.24 | Already in dependencies. No conflicts. |

---

## Sources

- **golang.org/x/crypto** — [pkg.go.dev/golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) (v0.47.0, published 2026-01-12) — HIGH confidence
  - Verified bcrypt, argon2, crypto/subtle availability and usage patterns
  - Note: Package not yet v1.0 stable, but industry-standard for Go cryptography

- **validator/v10** — [pkg.go.dev/github.com/go-playground/validator/v10](https://pkg.go.dev/github.com/go-playground/validator/v10) — HIGH confidence
  - Verified email, URL, custom validation features
  - Confirmed NOT a sanitization library (requires separate HTML sanitization)

- **golang.org/x/time/rate** — [pkg.go.dev/golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) — HIGH confidence
  - Verified token bucket algorithm, thread-safety, API methods
  - Already in project dependencies (internal/llm/ratelimit)

- **ulule/limiter/v3** — [pkg.go.dev/github.com/ulule/limiter/v3](https://pkg.go.dev/github.com/ulule/limiter/v3) — HIGH confidence
  - Verified Redis backend support, Gin middleware, rate format syntax
  - Confirmed v3.11.2 compatibility with go-redis/v9

- **gin-contrib/cors** — [pkg.go.dev/github.com/gin-contrib/cors](https://pkg.go.dev/github.com/gin-contrib/cors) — HIGH confidence
  - Verified credential protection, origin validation, SameSite support
  - Confirmed official Gin middleware

- **gorm.io/gorm** — [pkg.go.dev/gorm.io/gorm](https://pkg.go.dev/gorm.io/gorm) — HIGH confidence
  - Verified parameterized query patterns, SQL injection prevention
  - Confirmed safe vs. unsafe query patterns

- **godotenv** — [pkg.go.dev/github.com/joho/godotenv](https://pkg.go.dev/github.com/joho/godotenv) — HIGH confidence
  - Verified development-only recommendation, environment variable precedence
  - Confirmed NOT for production secret management

- **Existing codebase analysis** — `/Users/srie/work/vk/go.mod`, `internal/services/*.go`, `internal/middleware/*.go` — HIGH confidence
  - Verified current vulnerabilities: hardcoded credentials (.env), SQL injection (LIKE concatenation), missing rate limiting, timing attacks (OAuth), payment race conditions
  - Confirmed existing infrastructure: Gin 1.9.1, GORM 1.30.0, Redis v9, bcrypt, sessions store (hybrid Redis/DB)

---

*Stack research for: Go API Security (Payments + Authentication)*
*Researched: 2026-02-03*
*Confidence: HIGH — All libraries verified via official documentation (pkg.go.dev) and existing codebase analysis*
