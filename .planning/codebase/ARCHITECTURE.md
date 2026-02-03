# Architecture

**Analysis Date:** 2026-02-03

## Pattern Overview

**Overall:** Layered architecture with service-oriented business logic, organized as a REST API built on Gin framework with support for WebSocket signaling. The system implements domain-driven design principles separating concerns into middleware, services, data access, and infrastructure layers.

**Key Characteristics:**
- REST API with WebSocket support for real-time signaling
- Service layer abstractions for business logic
- Middleware-based cross-cutting concerns (auth, metrics, tracing, idempotency)
- Event-driven outbox pattern for asynchronous event processing
- Pluggable LLM provider system with circuit breaker and caching
- Hybrid session storage (Redis + database with fallback)

## Layers

**HTTP Handler Layer:**
- Purpose: Accept and validate HTTP requests, extract parameters, invoke services, return JSON responses
- Location: `cmd/main.go` (837 lines, all route handlers inline)
- Contains: Route definitions, request binding, response formatting, error handling
- Depends on: Services, middleware, models
- Used by: HTTP clients, WebSocket clients

**Service Layer:**
- Purpose: Implement core business logic for domain entities (Auth, Advocate, Client, Call, Payment, OAuth, WebRTC, LLM)
- Location: `internal/services/`
- Contains: `auth_service.go`, `advocate_service.go`, `call_service.go`, `payment_service.go`, `oauth_service.go`, `webrtc_service.go`, `signaling_service.go`, `llm_service.go`
- Depends on: Database, logger, outbox, models
- Used by: HTTP handlers

**Middleware Layer:**
- Purpose: Intercept requests for authentication, authorization, metrics, tracing, idempotency, request ID tracking
- Location: `internal/middleware/`
- Contains: `auth.go`, `metrics.go`, `idempotency.go`, `request_id.go`
- Depends on: Sessions, idempotency store, logger
- Used by: Gin router

**Data Access Layer:**
- Purpose: Manage database connections, schema, migrations, ORM interactions
- Location: `internal/database/` uses GORM with PostgreSQL
- Contains: `db.go` (connection pooling), `migrations.go` (SQL migrations)
- Depends on: Models, config
- Used by: Services

**Infrastructure Layer:**
- Purpose: Provide external service clients and utilities (Redis, observability, logging)
- Location: `internal/infra/`, `internal/logger/`, `internal/observability/`, `internal/sessions/`
- Contains: Redis client initialization, tracing setup, session store implementations
- Depends on: Config, models
- Used by: Services, main.go

**LLM Integration Layer:**
- Purpose: Abstract LLM provider interactions with resilience patterns
- Location: `internal/llm/`
- Contains: Provider factory, circuit breaker, rate limiting, caching, router
- Depends on: Models, logger, external APIs (Anthropic, OpenAI, Custom)
- Used by: Services

## Data Flow

**User Registration & Authentication Flow:**

1. Client submits registration request to `POST /api/auth/advocate/register` or `POST /api/auth/client/register`
2. Handler extracts `RegisterRequest` from JSON body
3. Handler invokes `AuthService.RegisterAdvocate()` or `AuthService.RegisterClient()`
4. Service validates input, hashes password with bcrypt, generates UUID
5. Service creates `Advocate` or `Client` record in database via GORM
6. Service returns user object to handler
7. Handler returns 200 with user data

**Session & Authorization Flow:**

1. Authenticated user includes `Authorization: Bearer {token}` header in request
2. `AuthMiddleware` intercepts request, extracts token
3. Middleware queries session store (Redis or database) for session matching token
4. If session found and not expired, middleware sets `user_id` and `user_type` in context
5. Handler retrieves user context with `c.Get("user_id")` and `c.Get("user_type")`
6. Handler enforces role-based access control (advocate vs client endpoints)

**Payment Processing Flow:**

1. Client initiates payment via `POST /api/client/payment/initiate`
2. `IdempotencyMiddleware` checks for prior request with same idempotency key
3. Handler invokes `PaymentService.InitiatePayment()`
4. Service creates `Payment` record with status="pending"
5. Service enqueues `OutboxEvent` with event_type="payment.initiated"
6. Service returns payment object with transaction ID
7. Background `OutboxWorker` polls `outbox_events` table for pending events
8. Worker processes payment.initiated event, calls external payment processor
9. Client calls `POST /api/client/payment/verify` with transaction_id
10. Service queries database for payment, validates status
11. On success, service enqueues `payment.completed` event
12. OutboxWorker processes payment.completed, updates advocate earnings

**Call Initiation Flow:**

1. User calls `POST /api/calls/initiate` with receiver_id
2. Handler invokes `CallService.InitiateCall(caller_id, receiver_id, call_type)`
3. Service creates `Call` record with status="initiated"
4. Service enqueues `OutboxEvent` with event_type="call.initiated"
5. Handler returns call object with call ID
6. Client establishes WebSocket connection to `GET /api/ws/signal`
7. `SignalingService` authenticates via JWT token in query
8. SignalingService establishes WebSocket connection, sends call notification
9. Receiver accepts call, WebRTC peer connection negotiation begins via signaling

**LLM Completion Flow:**

1. Service calls `LLMClient.Complete(context, request)`
2. Client checks rate limiter for user rate limit
3. Client checks circuit breaker (if primary provider is degraded, fail fast)
4. Client checks response cache (if result exists, return cached)
5. Client calls primary provider (Anthropic or OpenAI)
6. If primary provider fails and circuit open, fallback provider called
7. Response cached if caching enabled
8. Response returned to service
9. Service logs completion, may enqueue outbox event

**State Management:**

- **User State:** Sessions table tracks active sessions with token, user_id, user_type, expiry
- **Call State:** Calls table tracks call lifecycle (initiated -> accepted -> connected -> ended)
- **Payment State:** Payments table tracks payment lifecycle (pending -> completed/failed)
- **Outbox State:** OutboxEvents table queues events for eventual consistency processing
- **Session Storage Hybrid Mode:** Sessions stored in both Redis (fast reads/writes, volatile) and database (persistent fallback)

## Key Abstractions

**Service Pattern:**
- Purpose: Encapsulate business logic with dependency injection
- Examples: `AuthService`, `PaymentService`, `CallService` in `internal/services/`
- Pattern: Constructor takes dependencies (db, sessionStore, outbox), methods operate on domain models, returns errors for failures

**Middleware Pattern:**
- Purpose: Cross-cutting concerns applied to all/subset of routes
- Examples: `AuthMiddleware`, `MetricsMiddleware`, `IdempotencyMiddleware` in `internal/middleware/`
- Pattern: Gin `HandlerFunc` that validates request, sets context, calls `c.Next()` or `c.Abort()`

**Session Store Interface:**
- Purpose: Abstract session storage backend (Redis vs Database)
- Location: `internal/sessions/store.go`
- Pattern: Interface with `Create`, `GetByToken`, `Delete`, `Update` methods supporting both memory/Redis and database implementations

**LLM Provider Interface:**
- Purpose: Support multiple LLM backends with consistent interface
- Location: `internal/llm/providers/provider.go`
- Pattern: Interface with `Complete` method, factory pattern instantiates concrete provider (Anthropic, OpenAI, Custom)

**OutboxService Pattern:**
- Purpose: Queue asynchronous events for eventual consistency
- Location: `internal/outbox/service.go`, `internal/outbox/worker.go`
- Pattern: `Enqueue` writes event to database, background worker polls and processes events with retry logic

**Idempotency Store:**
- Purpose: Prevent duplicate request processing
- Location: `internal/idempotency/`
- Pattern: Store hashed request + response, subsequent identical requests return cached response

## Entry Points

**HTTP Server:**
- Location: `cmd/main.go` main function
- Triggers: Binary execution with optional --worker flag
- Responsibilities: Load config, initialize database, setup middleware, register routes, start HTTP server listening on PORT

**Worker Process:**
- Location: `cmd/worker/main.go` (referenced in main.go)
- Triggers: Binary execution with --worker flag
- Responsibilities: Poll outbox_events table, process events (call notifications, payment processing), update event status

**WebSocket Handler:**
- Location: `cmd/main.go` route `GET /api/ws/signal`
- Triggers: Client initiates WebSocket connection
- Responsibilities: Authenticate via JWT token, upgrade HTTP to WebSocket, forward signaling messages between peers

## Error Handling

**Strategy:** Explicit error returns with structured error messages, middleware-level aggregation, no global panic recovery beyond Gin default.

**Patterns:**

- **Service Errors:** Services return `error` interface, callers check error presence. Common errors: "invalid email", "email already registered", "session not found"
- **Database Errors:** GORM errors wrapped with context: `fmt.Errorf("failed to create call: %w", err)` providing caller with wrapped original error
- **Validation Errors:** Handler-level JSON binding validation with detailed "must be required" messages. Custom validation in services with business-specific messages
- **Authorization Errors:** Middleware returns 401/403 JSON with "access denied" or "invalid token". Handler returns 403 for role mismatch
- **Outbox Events:** Events marked status="failed" with `last_error` field, worker retries up to `OutboxMaxAttempts` times

## Cross-Cutting Concerns

**Logging:**
- Framework: Custom logger in `internal/logger/logger.go` with priority levels (SEVERE, INFO, FINER, FINEST, DEBUG)
- Pattern: Each service constructs logger with service name, calls `logger.Info()`, `logger.Finer()`, etc. with printf-style formatting
- Configuration: Log level set via `LOG_LEVEL` env var (default: INFO)

**Validation:**
- Framework: Gin's struct tag binding (`json:"field" binding:"required"`) for JSON parsing
- Pattern: Handlers use `c.ShouldBindJSON(&req)` which returns error if validation fails
- Services perform additional business logic validation (email uniqueness, balance checks, etc.)

**Authentication:**
- Framework: JWT tokens + session table
- Pattern: `AuthService.CreateSession()` generates random token, stores in sessions table with expiry
- `AuthMiddleware` validates token exists and not expired, injects user context
- Two user types: "advocate" and "client" with role-based route protection

**Metrics:**
- Framework: Prometheus via `github.com/prometheus/client_golang`
- Pattern: `MetricsMiddleware` records HTTP request count/latency. Endpoint `GET /metrics` exposes Prometheus format
- Configuration: `METRICS_ENABLED` env var

**Tracing:**
- Framework: OpenTelemetry with OTLP exporter
- Pattern: `observability.SetupTracing()` initializes tracer if `OTEL_EXPORTER_OTLP_ENDPOINT` configured
- `otelgin.Middleware` adds distributed tracing to all HTTP routes

---

*Architecture analysis: 2026-02-03*
