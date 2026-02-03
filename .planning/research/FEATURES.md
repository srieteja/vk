# Feature Research: Production-Ready Go API

**Domain:** Backend API Hardening (Financial Transaction Platform)
**Researched:** 2026-02-03
**Confidence:** HIGH

## Feature Landscape

This research focuses on production-readiness features for Go backends handling financial transactions and PII. The context is a functional API that needs hardening before production deployment with real user traffic and money flows.

### Table Stakes (Must Have for Production)

Features that production APIs MUST have. Missing these = security incidents, downtime, or operational failure.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Dependency Injection via Interfaces** | Enables testing, reduces coupling, standard Go practice | MEDIUM | Replace global `llmClient` with constructor injection. Extract interfaces for all services. |
| **Request Size Limits** | Prevents DoS via large payloads | LOW | Add middleware: `router.Use(gin.MaxRequestBodySize(1 << 20))` for 1MB limit. Critical for public APIs. |
| **CORS Configuration** | Required for browser-based clients | LOW | Add `cors.New()` middleware with explicit origins. Current: no CORS = browser apps broken. |
| **Rate Limiting (Per-Endpoint)** | Prevents brute force, DoS, resource exhaustion | MEDIUM | Already has LLM rate limiting. Need: auth endpoints (login/register), payment endpoints, WebRTC signaling. |
| **Structured Error Responses** | Enables proper client error handling | LOW | Standardize error format: `{"error": {"code": "...", "message": "...", "details": {...}}}`. Current: inconsistent JSON errors. |
| **Health Check Endpoints** | Required for load balancers, orchestrators | LOW | Add `/health` (liveness) and `/ready` (readiness with DB/Redis checks). K8s/ECS require these. |
| **Graceful Shutdown** | Prevents in-flight request failures during deployment | MEDIUM | Drain connections on SIGTERM. Already partially implemented with signal handling. Need: connection draining timeout. |
| **Request Timeouts** | Prevents resource leaks from slow clients | LOW | Add `ReadTimeout`, `WriteTimeout`, `IdleTimeout` to `http.Server`. Currently: no timeouts = hanging connections. |
| **Panic Recovery Middleware** | Prevents single panic from crashing entire server | LOW | Gin includes `gin.Recovery()` by default. Verify it's registered. Add custom recovery with alerting. |
| **Input Validation Middleware** | Prevents injection attacks, bad data | MEDIUM | Use `validator` package for struct validation. Current: manual validation scattered across services. |
| **Secret Management** | Separates secrets from code | MEDIUM | Replace `.env` defaults with required env vars in production. Use AWS Secrets Manager / HashiCorp Vault. Current: hardcoded `"secret"` defaults. |
| **TLS/HTTPS Configuration** | Required for production (PII, auth tokens in transit) | LOW | Add TLS config to server. Use Let's Encrypt / ACM certificates. Usually handled by load balancer. |
| **Audit Logging** | Compliance requirement (financial transactions) | MEDIUM | Log all auth events, payment operations, PII access with: who, what, when, result. Store separately from app logs. |
| **Database Connection Pooling** | Prevents connection exhaustion | LOW | Already configured (`DBMaxOpenConns`, `DBMaxIdleConns`). Verify pool metrics tracked. |

### Best Practices (Should Have for Maintainability)

Features that separate professional production systems from "works on my machine" code. Defer only if timeline critical.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Layered Architecture (Handler/Service/Repository)** | Clear separation of concerns, testability | HIGH | Current: 800-line main.go with inline handlers. Extract to: `handlers/` (HTTP), `services/` (business logic), `repositories/` (data access). |
| **Request Context Propagation** | Tracing, cancellation, timeouts through call stack | LOW | Use `context.Context` everywhere. Current: services take `ctx` but don't always propagate. Enforce with linter. |
| **Centralized Error Handling** | Consistent error responses, easier debugging | MEDIUM | Create error middleware that maps error types to HTTP codes. Current: each handler does `c.JSON(500, ...)` manually. |
| **API Versioning** | Allows backward-compatible changes | LOW | Add `/api/v1/` prefix. Current: no versioning = breaking changes break all clients. |
| **Response Pagination** | Prevents memory issues with large result sets | MEDIUM | Standardize pagination: `?page=1&limit=50`. Needed for: advocate list, call history, payment history. |
| **Idempotency Keys (Beyond Payments)** | Prevents duplicate operations on retries | MEDIUM | Already has payment idempotency. Extend to: call creation, advocate registration. Use same middleware pattern. |
| **Database Migrations Versioning** | Enables rollback, tracks schema history | LOW | Already has migrations. Add: version tracking table, rollback scripts, schema documentation. |
| **Metrics Instrumentation** | Enables performance monitoring, alerting | MEDIUM | Already has Prometheus. Add: endpoint latency histograms, error rate counters, DB connection pool metrics. |
| **Distributed Tracing** | Debug issues across service boundaries | LOW | Already has OpenTelemetry. Verify: all service methods create spans, context propagation works. |
| **Feature Flags** | Deploy without releasing, gradual rollouts | MEDIUM | Add simple flag system (DB or config file). Useful for: new payment gateways, experimental features. |
| **API Documentation (OpenAPI/Swagger)** | Enables client development, testing | MEDIUM | Use `swaggo/swag` for annotation-based docs. Generate from Go comments. Current: no machine-readable API docs. |
| **Background Job Queue** | Async processing for non-critical operations | MEDIUM | Already has outbox worker. Consider: separate queue for user-facing async ops (email, notifications). |
| **Database Read Replicas** | Scales read-heavy operations | HIGH | Separate read/write connections. Useful for: advocate search, call history. Requires connection manager. |

### Nice-to-Have (Can Defer)

Features that improve quality of life but aren't critical for launch. Prioritize after core hardening complete.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **GraphQL API** | Flexible data fetching for complex UIs | HIGH | Current REST API sufficient. Consider if client needs become complex. Adds significant complexity. |
| **WebSocket Connection Pooling** | Scales real-time connections | MEDIUM | Current WebRTC signaling uses direct WebSocket. Pool if connection count > 10k. |
| **Request/Response Caching** | Reduces DB load for repeated queries | MEDIUM | Redis already in stack. Add for: advocate profiles, call history. Use cache-aside pattern. |
| **Multi-Tenancy Support** | Isolates data between organizations | HIGH | Current: single tenant (advocate/client users). Not needed unless B2B pivot. |
| **A/B Testing Framework** | Enables experimentation | MEDIUM | Use feature flags + metrics. Useful for: pricing experiments, UI flows. |
| **Webhook Delivery System** | Enables integrations | MEDIUM | Outbox pattern already exists. Wrap with retry logic + signature verification for external webhooks. |
| **API Rate Limit Headers** | Communicates limits to clients | LOW | Add `X-RateLimit-Limit`, `X-RateLimit-Remaining` headers. Improves client DX. |
| **Request Replay Protection** | Prevents replay attacks on sensitive operations | MEDIUM | Add nonce validation for payment operations. Useful if payment gateway doesn't handle this. |
| **Multi-Region Deployment** | Reduces latency for global users | HIGH | Current: single region. Defer until proven user base in multiple geos. Requires: DB replication, session management. |
| **Blue-Green Deployment** | Zero-downtime deployments | MEDIUM | Handled by infrastructure (K8s, ECS). Ensure graceful shutdown + health checks work. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem valuable but create more problems than they solve in this context.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Service Mesh (Istio/Linkerd)** | "Industry best practice" for microservices | Adds massive complexity, operational overhead. You have a monolith. | Use standard middleware (rate limiting, observability). Service mesh when splitting into 5+ services. |
| **Event Sourcing** | "Never lose data" for audit trail | Requires rewriting entire data layer. Massive complexity. Current domain doesn't need full event history. | Use audit logging for critical events. Outbox pattern already provides event stream. |
| **CQRS (Command Query Separation)** | "Scales reads and writes separately" | Premature optimization. Adds consistency challenges. Current load doesn't require it. | Use read replicas if read load becomes issue. Keep simple read/write service layer. |
| **Microservices Architecture** | "Scalability, independent deployment" | Team is small, monolith is manageable. Microservices add: network calls, distributed tracing complexity, deployment coordination. | Keep monolith. Use modular code structure (packages/interfaces). Split when team > 10 or clear service boundaries emerge. |
| **NoSQL Database** | "Better performance" | PostgreSQL handles current load. Switching adds: consistency challenges, migration cost, team learning curve. | Optimize PostgreSQL: indexes, connection pooling, read replicas. NoSQL only if specific use case (e.g., time-series data). |
| **API Gateway (Kong/Apigee)** | "Centralized API management" | Adds another component to operate. Current load doesn't need dedicated gateway. Gin middleware handles auth/rate limiting. | Use Gin middleware. Add gateway only when: multiple backend services, need vendor API management features. |
| **ORM Replacement with SQL** | "ORMs are slow" | GORM performance fine for current load. Raw SQL adds: manual error handling, query building complexity, migration complexity. | Keep GORM. Optimize with: proper indexes, query profiling, raw SQL only for proven bottlenecks. |
| **Real-Time Dashboard** | "Monitor everything live" | Building custom dashboard is time sink. Operational burden to maintain. | Use Grafana with Prometheus metrics. Existing tools better than custom builds. |

## Feature Dependencies

```
Core Architecture Changes (Phase 1)
    ├──> Dependency Injection
    │       └──> Extract Service Interfaces
    │               └──> Repository Pattern
    │
    └──> Layered Architecture
            ├──> Handler Extraction
            ├──> Service Layer (depends on Interfaces)
            └──> Middleware Organization

Security Hardening (Phase 2, depends on Phase 1 for testability)
    ├──> Request Size Limits (LOW complexity, no dependencies)
    ├──> CORS Configuration (LOW complexity, no dependencies)
    ├──> Rate Limiting
    │       └──> Middleware Framework (already exists)
    ├──> Input Validation
    │       └──> Centralized Error Handling
    └──> Secret Management (LOW complexity, no dependencies)

Production Operations (Phase 3, depends on Phase 1 & 2)
    ├──> Health Checks
    │       └──> Database/Redis Health Probes
    ├──> Graceful Shutdown (already partial)
    │       └──> Connection Draining
    ├──> Audit Logging
    │       └──> Structured Logging (already exists)
    └──> Metrics Enhancement
            └──> OpenTelemetry (already exists)

API Quality (Phase 4, parallel with Phase 3)
    ├──> Structured Error Responses
    │       └──> Error Type System
    ├──> API Versioning (LOW complexity)
    ├──> Pagination
    │       └──> Repository Pattern (from Phase 1)
    └──> API Documentation
            └──> Code Annotations
```

### Dependency Notes

- **Dependency Injection requires Service Interfaces**: Can't inject without contracts. Must extract interfaces first, then refactor main.go to pass dependencies.
- **Layered Architecture enables testability**: Handlers should be thin (HTTP concern). Services contain business logic (testable without HTTP). Repositories handle data access (mockable).
- **Rate Limiting depends on Middleware Framework**: Gin already provides this. Just need to implement rate limiter middleware pattern (similar to existing auth middleware).
- **Audit Logging depends on Structured Logging**: Already using custom logger. Extend with audit-specific methods (AuditAuth, AuditPayment) that write to separate output.
- **Health Checks depend on DB/Redis Health Probes**: Need to expose DB.Ping() and Redis.Ping() through health check endpoint. Orchestrators (K8s) use this for readiness.

## MVP Definition

**Context**: This is a hardening milestone for an EXISTING functional API. MVP = production-ready, not feature-complete.

### Launch With (v1.0 - Production Ready)

Minimum required to safely run in production with real money and PII.

- [x] **Dependency Injection** — Eliminates global state, enables testing. Currently: global `llmClient` breaks tests.
- [x] **Request Size Limits** — Prevents DoS. Currently: no limits = server can be overwhelmed.
- [x] **CORS Configuration** — Required for web clients. Currently: missing = web app can't call API.
- [x] **Rate Limiting (Auth Endpoints)** — Prevents brute force. Currently: unlimited login attempts.
- [x] **Structured Error Responses** — Enables proper client error handling. Currently: inconsistent errors confuse clients.
- [x] **Health Check Endpoints** — Required for orchestrators. Currently: load balancer can't detect unhealthy instances.
- [x] **Graceful Shutdown** — Prevents request failures during deployment. Currently: partial implementation.
- [x] **Request Timeouts** — Prevents hanging connections. Currently: no timeouts = resource leaks.
- [x] **Secret Management** — Prevents credential leaks. Currently: hardcoded defaults in code.
- [x] **Audit Logging** — Compliance requirement for financial platform. Currently: basic logging insufficient.
- [x] **Layered Architecture** — Separates concerns for maintainability. Currently: 800-line main.go unmaintainable.
- [x] **Input Validation** — Prevents injection attacks. Currently: manual validation inconsistent.

**Rationale**: These features directly address identified security vulnerabilities and operational requirements for production deployment. All are HIGH or MEDIUM complexity but necessary for safe operation.

### Add After Launch (v1.1 - Operational Maturity)

Features that improve operations and developer experience but aren't blocking production launch.

- [ ] **API Versioning** — Enables non-breaking changes. Add once API stabilizes with real users.
- [ ] **Response Pagination** — Prevents memory issues. Add when advocate count > 1000 or performance issues surface.
- [ ] **Enhanced Metrics** — Improves observability. Prometheus already exists; add endpoint-specific metrics based on production patterns.
- [ ] **API Documentation** — Improves client development. Generate OpenAPI spec once API structure stabilizes.
- [ ] **Feature Flags** — Enables gradual rollouts. Useful for second payment gateway or experimental features.
- [ ] **Database Read Replicas** — Scales read operations. Add when DB CPU > 70% consistently.

**Trigger for adding**: User base > 500 active users OR performance issues detected in production monitoring.

### Future Consideration (v2.0+ - Scale & Sophistication)

Features to defer until product-market fit established and scale demands them.

- [ ] **Request/Response Caching** — Redis already in stack. Add when DB becomes bottleneck (query time > 100ms consistently).
- [ ] **Webhook Delivery System** — Enables partner integrations. Add when partners request integration capabilities.
- [ ] **Background Job Queue (Enhanced)** — Separate from outbox worker. Add when async operations > 1000/hour.
- [ ] **Multi-Region Deployment** — Reduces latency globally. Add when user base spreads across multiple continents.
- [ ] **Blue-Green Deployment** — Infrastructure concern. Add when deployment frequency > 5/week.

**Trigger**: Product-market fit achieved, consistent revenue, scaling challenges emerge.

## Feature Prioritization Matrix

### Phase 1: Core Architecture (v1.0 Foundation)

| Feature | User Value | Implementation Cost | Priority | Rationale |
|---------|------------|---------------------|----------|-----------|
| Dependency Injection | LOW (internal) | MEDIUM | P1 | Enables all other refactoring. Blocks testing improvements. |
| Layered Architecture | LOW (internal) | HIGH | P1 | Blocks maintainability. 800-line main.go unmaintainable. |
| Service Interfaces | LOW (internal) | MEDIUM | P1 | Dependency for DI. Enables mocking in tests. |

### Phase 2: Security Hardening (v1.0 Critical)

| Feature | User Value | Implementation Cost | Priority | Rationale |
|---------|------------|---------------------|----------|-----------|
| Rate Limiting (Auth) | HIGH | MEDIUM | P1 | Prevents brute force. High security risk without it. |
| Request Size Limits | HIGH | LOW | P1 | Prevents DoS. Easy win. |
| CORS Configuration | HIGH | LOW | P1 | Required for web clients. Easy win. |
| Secret Management | HIGH | MEDIUM | P1 | Prevents credential leaks. Compliance requirement. |
| Input Validation | HIGH | MEDIUM | P1 | Prevents injection. Already identified vulnerability. |
| Audit Logging | HIGH | MEDIUM | P1 | Compliance requirement for financial transactions. |

### Phase 3: Production Operations (v1.0 Critical)

| Feature | User Value | Implementation Cost | Priority | Rationale |
|---------|------------|---------------------|----------|-----------|
| Health Check Endpoints | MEDIUM | LOW | P1 | Required for orchestrators. Easy win. |
| Request Timeouts | MEDIUM | LOW | P1 | Prevents resource leaks. Easy win. |
| Graceful Shutdown | HIGH | MEDIUM | P1 | Prevents failed requests during deployment. Partially exists. |
| Structured Error Responses | MEDIUM | LOW | P1 | Improves client error handling. Easy standardization. |
| Panic Recovery | HIGH | LOW | P1 | Prevents crashes. Verify existing Gin recovery works. |

### Phase 4: API Quality (v1.1 Enhancement)

| Feature | User Value | Implementation Cost | Priority | Rationale |
|---------|------------|---------------------|----------|-----------|
| API Versioning | LOW | LOW | P2 | Good practice. Not blocking. Add before breaking changes. |
| Response Pagination | MEDIUM | MEDIUM | P2 | Needed when data grows. Current small dataset OK. |
| Enhanced Metrics | LOW | MEDIUM | P2 | Improves observability. Basic metrics already exist. |
| API Documentation | MEDIUM | MEDIUM | P2 | Improves DX. Not blocking for launch. |

### Phase 5: Scale & Sophistication (v2.0+)

| Feature | User Value | Implementation Cost | Priority | Rationale |
|---------|------------|---------------------|----------|-----------|
| Request Caching | MEDIUM | MEDIUM | P3 | Performance optimization. Not needed yet. |
| Feature Flags | LOW | MEDIUM | P3 | Nice to have. Not needed for current feature set. |
| Database Read Replicas | LOW | HIGH | P3 | Scale optimization. Current load doesn't require. |
| Multi-Region | LOW | HIGH | P3 | Geographic scale. Not needed for current user base. |

**Priority Key:**
- **P1 (Must Have)**: Blocking production launch. Security, compliance, or operational requirement.
- **P2 (Should Have)**: Improves quality but not blocking. Add when time permits or triggered by usage.
- **P3 (Nice to Have)**: Future optimization. Defer until scale or complexity demands it.

## Production Readiness Checklist

Based on Google SRE practices and industry standards for financial transaction platforms.

### Security

- [x] **Authentication & Authorization**: Session-based auth with Redis. OAuth implemented.
- [ ] **Rate Limiting**: Currently only LLM service. MUST add: auth endpoints, payment endpoints.
- [ ] **Input Validation**: Manual validation exists. MUST standardize with validator middleware.
- [ ] **Secret Management**: .env with hardcoded defaults. MUST use secret manager in production.
- [ ] **CORS**: Missing. MUST configure before browser clients.
- [ ] **Request Size Limits**: Missing. MUST add to prevent DoS.
- [x] **TLS/HTTPS**: Usually handled by load balancer. Verify certificate config.
- [ ] **Audit Logging**: Basic logging exists. MUST add audit-specific events.

### Reliability

- [ ] **Health Checks**: Missing `/health` and `/ready`. MUST add for orchestrators.
- [x] **Graceful Shutdown**: Signal handling exists. MUST add connection draining.
- [ ] **Request Timeouts**: Missing. MUST add to http.Server config.
- [x] **Panic Recovery**: Gin.Recovery() used. MUST verify + add alerting.
- [x] **Database Connection Pooling**: Configured. VERIFY pool metrics tracked.
- [x] **Idempotency**: Payment operations covered. EXTEND to other critical operations.
- [x] **Circuit Breaker**: LLM service has it. CONSIDER for external payment gateways.

### Observability

- [x] **Structured Logging**: Custom logger implemented. EXTEND with audit methods.
- [x] **Metrics**: Prometheus configured. ENHANCE with endpoint latency, error rates.
- [x] **Distributed Tracing**: OpenTelemetry configured. VERIFY span propagation works.
- [ ] **Error Tracking**: Basic logging. CONSIDER integrating Sentry/Rollbar.
- [x] **Request ID Propagation**: Middleware exists. VERIFY propagation to all logs.

### Architecture

- [ ] **Layered Architecture**: 800-line main.go. MUST extract handlers/services/repos.
- [ ] **Dependency Injection**: Global `llmClient`. MUST refactor to constructor injection.
- [ ] **Service Interfaces**: Services tightly coupled. MUST extract interfaces for DI.
- [ ] **Error Handling**: Manual in each handler. SHOULD centralize in middleware.
- [ ] **API Versioning**: Not versioned. SHOULD add `/api/v1/` before public launch.

### Testing

- [ ] **Unit Test Coverage**: 19% file coverage. MUST achieve >70% for critical paths.
- [ ] **Integration Tests**: Missing. MUST add for auth + payment flows.
- [ ] **Security Tests**: Missing. MUST add injection tests, bypass attempts.
- [ ] **Load Tests**: Missing. SHOULD add for payment + auth endpoints.

### Operations

- [ ] **Database Migrations**: Implemented. ENHANCE with version tracking, rollback scripts.
- [ ] **Configuration Management**: .env file. MIGRATE to environment-specific configs.
- [ ] **Deployment Automation**: Not visible in code. ENSURE CI/CD pipeline exists.
- [ ] **Monitoring Alerts**: Not visible in code. SETUP alerts for error rates, latency, DB health.
- [ ] **Incident Response**: Not visible. DOCUMENT runbooks for common issues.

## Go-Specific Production Patterns

### 1. Dependency Injection Pattern

**Problem**: Current code uses global variable `var llmClient *llm.Client` making testing difficult and hiding dependencies.

**Solution**: Constructor injection with interfaces.

```go
// Define interface
type LLMProvider interface {
    Generate(ctx context.Context, prompt string) (string, error)
}

// Service depends on interface, not concrete type
type SomeService struct {
    llm LLMProvider
}

func NewSomeService(llm LLMProvider) *SomeService {
    return &SomeService{llm: llm}
}

// In main.go, wire dependencies
llmClient := llm.NewClient(config)
someService := services.NewSomeService(llmClient)
```

**Complexity**: MEDIUM. Requires refactoring all services to accept dependencies in constructor.

### 2. Handler/Service/Repository Pattern

**Problem**: Handlers in main.go contain business logic and data access, making them untestable.

**Solution**: Three-layer architecture.

```go
// Handler (HTTP concerns only)
func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
    var req PaymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse("invalid_request", err))
        return
    }

    payment, err := h.service.InitiatePayment(c.Request.Context(), req)
    if err != nil {
        c.JSON(500, ErrorResponse("payment_failed", err))
        return
    }

    c.JSON(200, payment)
}

// Service (business logic)
func (s *PaymentService) InitiatePayment(ctx context.Context, req PaymentRequest) (*Payment, error) {
    // Validate business rules
    // Calculate fees
    // Call repository
    return s.repo.Create(ctx, payment)
}

// Repository (data access)
func (r *PaymentRepository) Create(ctx context.Context, payment *Payment) error {
    return r.db.Create(payment).Error
}
```

**Complexity**: HIGH. Requires restructuring entire codebase. Benefits: testability, maintainability, clear separation of concerns.

### 3. Error Handling Pattern

**Problem**: Inconsistent error responses: `c.JSON(500, gin.H{"error": "..."})`

**Solution**: Centralized error middleware with typed errors.

```go
// Define error types
type AppError struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
    Status  int         `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

// Middleware catches errors
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err

            var appErr *AppError
            if errors.As(err, &appErr) {
                c.JSON(appErr.Status, appErr)
                return
            }

            c.JSON(500, &AppError{
                Code:    "internal_error",
                Message: "Internal server error",
            })
        }
    }
}

// Services return typed errors
func (s *Service) DoSomething() error {
    return &AppError{
        Code:    "invalid_input",
        Message: "Email already registered",
        Status:  400,
    }
}
```

**Complexity**: MEDIUM. Requires defining error types and updating all error returns.

### 4. Middleware Composition Pattern

**Problem**: Current middleware applied ad-hoc. Hard to see what protections apply to which routes.

**Solution**: Explicit middleware chains.

```go
// Define middleware groups
publicRoutes := router.Group("/api/v1")
publicRoutes.Use(
    middleware.RateLimit(100),  // 100 req/min
    middleware.RequestSize(1 << 20),  // 1MB
    middleware.CORS(),
)

authenticatedRoutes := router.Group("/api/v1")
authenticatedRoutes.Use(
    middleware.RateLimit(1000),  // Higher limit for authenticated
    middleware.RequestSize(1 << 20),
    middleware.Auth(sessionStore),
    middleware.AuditLog(),  // Audit all authenticated actions
)

paymentRoutes := router.Group("/api/v1/payments")
paymentRoutes.Use(
    middleware.Idempotency(store),
    middleware.RateLimit(10),  // Very restrictive for payments
    middleware.AuditLog(),
)
```

**Complexity**: LOW. Just reorganize existing route registration.

### 5. Context Propagation Pattern

**Problem**: Context not always propagated through call chain, losing trace info and cancellation.

**Solution**: Always accept `context.Context` as first parameter.

```go
// Service method
func (s *PaymentService) InitiatePayment(ctx context.Context, req PaymentRequest) error {
    // Start span for tracing
    ctx, span := otel.Tracer("payment").Start(ctx, "InitiatePayment")
    defer span.End()

    // Pass context to repository
    return s.repo.Create(ctx, payment)
}

// Repository method
func (r *PaymentRepository) Create(ctx context.Context, payment *Payment) error {
    // GORM respects context for cancellation
    return r.db.WithContext(ctx).Create(payment).Error
}
```

**Complexity**: LOW. Services already partially do this. Just need to enforce consistently.

## Technology-Specific Recommendations

### Gin Framework

- **Use**: Excellent performance, good middleware ecosystem, active maintenance
- **Already adopted**: Current codebase uses Gin
- **Best practices to adopt**:
  - Use `gin.SetMode(gin.ReleaseMode)` in production (already done)
  - Add `gin.Recovery()` middleware (already exists, verify)
  - Use `router.MaxMultipartMemory` for file uploads if needed
  - Use `c.ShouldBindJSON()` for validation (already doing)

### GORM

- **Use**: Already adopted, fine for current scale
- **Avoid premature optimization**: Don't replace with raw SQL unless proven bottleneck
- **Best practices to adopt**:
  - Use `db.WithContext(ctx)` for cancellation support
  - Add proper indexes on foreign keys (verify migrations have these)
  - Use `db.Session(&gorm.Session{PrepareStmt: true})` for prepared statements
  - Monitor slow query log in production

### Redis

- **Use**: Already adopted for sessions and LLM cache
- **Extend usage**: Consider for advocate profile caching after launch
- **Best practices**:
  - Always set TTL to prevent memory leaks
  - Use pipelining for multiple operations
  - Monitor memory usage
  - Configure maxmemory-policy (verify deployment config)

### PostgreSQL

- **Use**: Already adopted, excellent for transactional workloads
- **Best practices**:
  - Add indexes on filter columns (location, availability)
  - Use `EXPLAIN ANALYZE` to optimize slow queries
  - Configure connection pool based on instance size
  - Enable query logging for slow queries (> 1s)
  - Regular VACUUM ANALYZE in production

## Sources

**Confidence Level: HIGH** - Based on direct codebase analysis and established Go production practices.

### Primary Sources

1. **Direct Codebase Analysis**:
   - `/Users/srie/work/vk/cmd/main.go` - 800-line main with inline handlers
   - `/Users/srie/work/vk/internal/services/llm_service.go` - Global `llmClient` variable
   - `/Users/srie/work/vk/internal/config/config.go` - Hardcoded secret defaults
   - `/Users/srie/work/vk/internal/middleware/` - Existing middleware patterns
   - `.planning/PROJECT.md` - Identified security issues and hardening goals

2. **Go Standard Practices** (High confidence - well-established patterns):
   - Dependency injection via constructor parameters (standard Go idiom since 1.0)
   - Three-layer architecture (handler/service/repository) - documented in Go Wiki
   - Context propagation for cancellation and tracing - standard since Go 1.7
   - Interface-based dependency injection for testing - core Go testing pattern

3. **Production Go API Patterns** (High confidence - industry standard):
   - Graceful shutdown with SIGTERM handling - Kubernetes requirement
   - Health check endpoints (`/health`, `/ready`) - Cloud Native Computing Foundation standards
   - Request timeouts on http.Server - Go http package documentation
   - Rate limiting per endpoint - OWASP API Security Top 10
   - Structured error responses - REST API best practices (RFC 7807)

4. **Financial Transaction Platform Requirements** (High confidence - compliance):
   - Audit logging for financial operations - PCI DSS requirement
   - Request size limits - OWASP DoS prevention
   - Secret management (no hardcoded credentials) - PCI DSS 6.3.1
   - TLS for PII/auth tokens in transit - GDPR Article 32

5. **Gin Framework Specifics** (High confidence - official documentation):
   - `gin.Recovery()` middleware for panic recovery
   - `gin.SetMode(gin.ReleaseMode)` for production
   - `router.Use()` for middleware composition
   - Gin already includes sensible defaults, minimal additional hardening needed

### Verification Notes

- **Global state issue**: Verified by reading `llm_service.go` line 19: `var llmClient *llm.Client`
- **800-line main.go**: Confirmed by examining file structure
- **Missing CORS/rate limiting/request limits**: Verified by checking middleware directory - only auth, idempotency, metrics, request_id exist
- **Hardcoded secrets**: Verified in `config.go` line 51, 56: default values `"secret"`
- **Existing observability**: Confirmed OpenTelemetry, Prometheus, custom logger already implemented

---
*Feature research for: Production-Ready Go API Hardening*
*Researched: 2026-02-03*
*Context: Functional API → Production-Ready API for financial transaction platform*
