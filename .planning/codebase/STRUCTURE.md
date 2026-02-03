# Codebase Structure

**Analysis Date:** 2026-02-03

## Directory Layout

```
vk/
├── bin/                          # Compiled binaries (main executable)
├── cmd/                          # Application entrypoints
│   ├── main.go                   # HTTP server with all route handlers (837 lines)
│   └── worker/                   # Background worker for outbox event processing
├── docs/                         # Documentation files
├── internal/                     # Private packages (Go convention)
│   ├── config/                   # Configuration management
│   │   └── config.go             # Config struct, LoadConfig function
│   ├── database/                 # Database initialization and migrations
│   │   ├── db.go                 # GORM connection setup, pooling
│   │   └── migrations.go         # SQL migration execution
│   ├── idempotency/              # Idempotency key store
│   │   └── (idempotency implementation)
│   ├── infra/                    # Infrastructure clients
│   │   └── redis.go              # Redis client factory
│   ├── llm/                      # LLM integration with resilience
│   │   ├── client.go             # Main LLM client with circuit breaker, rate limiter, cache
│   │   ├── cache/                # Response caching layer
│   │   ├── circuit/              # Circuit breaker for provider fallback
│   │   ├── providers/            # LLM provider implementations
│   │   │   ├── provider.go       # Provider interface
│   │   │   ├── factory.go        # Factory to create providers
│   │   │   ├── anthropic.go      # Anthropic SDK integration
│   │   │   ├── openai.go         # OpenAI SDK integration
│   │   │   └── custom.go         # Custom LLM endpoint support
│   │   ├── ratelimit/            # Per-user rate limiting
│   │   ├── router/               # LLM provider selection logic
│   │   └── config.yaml           # LLM configuration
│   ├── logger/                   # Custom logging framework
│   │   └── logger.go             # Logger struct with priority levels
│   ├── metrics/                  # Prometheus metrics
│   │   └── (metrics initialization)
│   ├── middleware/               # Gin middleware handlers
│   │   ├── auth.go               # Bearer token validation, session lookup
│   │   ├── metrics.go            # Request counter and latency recording
│   │   ├── idempotency.go        # Idempotency key checking/storage
│   │   └── request_id.go         # Unique request ID injection
│   ├── models/                   # Data models (GORM structs)
│   │   ├── models.go             # Advocate, Client, Call, Payment, Session, OutboxEvent, IdempotencyKey
│   │   └── requests.go           # Request DTOs
│   ├── observability/            # OpenTelemetry tracing
│   │   └── tracing.go            # OTLP exporter setup
│   ├── outbox/                   # Event outbox pattern
│   │   ├── service.go            # Event enqueueing
│   │   └── worker.go             # Background event processor
│   ├── services/                 # Business logic services
│   │   ├── auth_service.go       # User registration, login, session creation
│   │   ├── advocate_service.go   # Advocate profile, availability, earnings
│   │   ├── call_service.go       # Call initiation and state management
│   │   ├── payment_service.go    # Payment processing with outbox
│   │   ├── oauth_service.go      # Google OAuth flow
│   │   ├── webrtc_service.go     # WebRTC peer connection setup
│   │   ├── signaling_service.go  # WebSocket signaling for WebRTC
│   │   └── llm_service.go        # LLM API interactions
│   └── sessions/                 # Session storage (Redis + Database hybrid)
│       └── store.go              # Session interface and implementations
├── migrations/                   # SQL migration files (executed by database/migrations.go)
├── plans/                        # Planning documents (GSD phase plans)
├── tests/                        # Test files
│   └── unit/                     # Unit tests
├── .env                          # Environment variables (secrets)
├── .env.example                  # Example environment file template
├── go.mod                        # Go module definition
├── go.sum                        # Go dependencies lock file
├── Makefile                      # Build targets (make build, make test, etc.)
├── docker-compose.yml            # Local development infrastructure (PostgreSQL, Redis)
├── Dockerfile                    # Container image definition
├── README.md                     # Project overview
├── SETUP_GUIDE.md                # Development setup instructions
├── GEMINI.md                     # (Unknown purpose, likely legacy)
└── qodana.yaml                   # Code quality scanning configuration

```

## Directory Purposes

**cmd/**
- Purpose: Application entry points - server and background workers
- Contains: Go main packages with route handlers and initialization logic
- Key files: `cmd/main.go` is the primary server (837 lines, all routes defined inline)

**internal/**
- Purpose: Private Go packages (not importable by external projects)
- Contains: All business logic, infrastructure, and utility code
- Key files: Organized by concern (config, database, services, middleware, models)

**internal/services/**
- Purpose: Domain-driven business logic services
- Contains: 8 service files for Auth, Advocate, Client, Call, Payment, OAuth, WebRTC, LLM
- Key files: Each service has NewXXXService constructor and business methods

**internal/llm/**
- Purpose: Language model integration with resilience patterns
- Contains: Provider abstraction, circuit breaker, rate limiter, cache
- Key files: `client.go` orchestrates all LLM concerns

**internal/middleware/**
- Purpose: Cross-cutting HTTP request concerns
- Contains: Auth validation, metrics collection, idempotency, request tracking
- Key files: Each middleware is its own file (auth.go, metrics.go, idempotency.go)

**internal/models/**
- Purpose: Data transfer objects and GORM database models
- Contains: Advocate, Client, Call, Payment, Session, OutboxEvent, IdempotencyKey structs
- Key files: `models.go` (main structs with GORM tags), `requests.go` (request DTOs)

**migrations/**
- Purpose: SQL schema versions for PostgreSQL
- Contains: Timestamped SQL files executed in order
- Key files: Applied automatically on startup if `RUN_MIGRATIONS=true`

**tests/unit/**
- Purpose: Unit test files
- Contains: Test cases for services and utilities
- Key files: Organized by package being tested

## Key File Locations

**Entry Points:**
- `cmd/main.go`: HTTP API server with all route handlers, middleware setup, service initialization
- `cmd/worker/main.go`: Background worker for outbox event processing

**Configuration:**
- `internal/config/config.go`: Config struct with 40+ environment variables, LoadConfig() function
- `.env`: Runtime environment variables (DATABASE_URL, JWT_SECRET, API keys, etc.)
- `internal/llm/config.yaml`: LLM provider selection and fallback configuration

**Core Logic:**
- `internal/services/`: 8 services handling Auth, Advocate, Client, Call, Payment, OAuth, WebRTC, LLM
- `internal/database/`: GORM connection management and schema initialization

**Data Models:**
- `internal/models/models.go`: GORM entity structs (Advocate, Client, Call, Payment, Session, OutboxEvent, IdempotencyKey)
- `internal/models/requests.go`: Request DTO structs

**Testing:**
- `tests/unit/`: Unit test files for services and utilities

**Infrastructure:**
- `internal/infra/redis.go`: Redis client initialization
- `internal/observability/tracing.go`: OpenTelemetry setup
- `internal/logger/logger.go`: Custom logging framework
- `internal/sessions/store.go`: Session store interface and implementations (Redis, Database, Hybrid)

## Naming Conventions

**Files:**
- Service files: `{domain}_service.go` (e.g., `auth_service.go`, `payment_service.go`)
- Models: `models.go` (all GORM structs in one file)
- Handlers: Routes defined inline in `cmd/main.go`, no separate handler files
- Middleware: `{concern}.go` (e.g., `auth.go`, `metrics.go`)

**Directories:**
- Go package directories use lowercase, underscores for multi-word names: `internal/llm`, `internal/sessions`
- No hyphens in directory names

**Functions:**
- Public methods (exported): `PascalCase` (e.g., `RegisterAdvocate`, `InitiateCall`)
- Private methods: `camelCase` (standard Go convention)
- Constructor functions: `NewXXXService` or `NewXXX` (e.g., `NewAuthService`)
- Handler methods: Named after HTTP method + path intent (e.g., route handler for `POST /api/auth/advocate/register` is unnamed inline)

**Variables:**
- Local variables: `camelCase`
- Constants: `UPPER_CASE` (Go convention)
- Struct fields: `PascalCase` with GORM/JSON tags (e.g., `ID`, `Email`, `CreatedAt`)

**Types:**
- Domain models: `PascalCase` (e.g., `Advocate`, `Client`, `Call`)
- Interfaces: `PascalCase` ending in "er" or "Store" (e.g., `Provider`, `Store`)
- Service structs: `PascalCase` ending in "Service" (e.g., `AuthService`, `PaymentService`)

## Where to Add New Code

**New Feature (Domain Logic):**
- Primary code: Create service method in `internal/services/{domain}_service.go`
- Database queries: Use GORM in service methods with `s.db.Create()`, `s.db.Where()`, etc.
- Outbox events: Call `s.outboxService.Enqueue()` for async processing
- Tests: Add unit test file `tests/unit/{domain}_service_test.go`

**New HTTP Endpoint:**
- Route definition: Add route handler to `cmd/main.go` in appropriate group (auth, advocate, client, calls)
- Handler logic: Inline in route definition (current pattern is no separate handler files)
- Middleware: Apply `authMiddleware` or other middleware via `.Use()` on route group
- Response: Return JSON via `c.JSON(statusCode, responseStruct)`

**New Service:**
- Implementation: Create `internal/services/{domain}_service.go` with `New{Domain}Service` constructor
- Dependencies: Inject via constructor parameters (db, sessionStore, outboxService, logger)
- Logger: Create logger with `logger.NewLogger("{DomainService}", logger.INFO)`
- Database models: Add struct to `internal/models/models.go` with GORM tags

**New Middleware:**
- Implementation: Create `internal/middleware/{concern}.go` with `func {Concern}Middleware(...) gin.HandlerFunc`
- Pattern: Return function that accepts `*gin.Context`, performs validation, calls `c.Next()` or `c.Abort()`
- Registration: Call `router.Use()` or `routeGroup.Use()` in `cmd/main.go`

**New Data Model:**
- Definition: Add struct to `internal/models/models.go` with GORM tags
- Table name: Add `TableName()` method returning database table name
- Schema: GORM auto-migration handled by `database.InitSchema()` if `AUTO_MIGRATE=true`

**New Request DTO:**
- Definition: Add struct to `internal/models/requests.go` with JSON tags and binding tags
- Validation: Gin will validate during `c.ShouldBindJSON()`

**New External Integration:**
- Provider: Create `internal/llm/providers/{provider_name}.go` implementing `Provider` interface (for LLM)
- Client: Create wrapper in `internal/infra/` or integrate in service
- Configuration: Add env vars to `internal/config/config.go`, handle in config loading

## Special Directories

**migrations/**
- Purpose: SQL schema versions
- Generated: No (manually written)
- Committed: Yes, all migration files in git
- How applied: `database.ApplyMigrations()` executed if `RUN_MIGRATIONS=true` at startup

**cmd/worker/**
- Purpose: Background worker process for outbox event processing
- Generated: No
- Committed: Yes
- How run: Separate binary or goroutine spawned from main server

**docker-compose.yml**
- Purpose: Local development infrastructure (PostgreSQL, Redis)
- Generated: No (manually maintained)
- Committed: Yes
- How used: `docker-compose up` spins up PostgreSQL and Redis for local development

**tests/unit/**
- Purpose: Unit test files
- Generated: No (manually written by developers)
- Committed: Yes
- How run: `go test ./internal/...` or `make test`

---

*Structure analysis: 2026-02-03*
