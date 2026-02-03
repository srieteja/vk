# Technology Stack

**Analysis Date:** 2026-02-03

## Languages

**Primary:**
- Go 1.24.0 - Backend API and all core services

## Runtime

**Environment:**
- Go 1.24.0 (Linux: Alpine)

**Package Manager:**
- Go modules (go.mod/go.sum)
- Lockfile: Present (`go.sum` 18KB)

## Frameworks

**Core:**
- Gin v1.9.1 - HTTP web framework and routing

**Database:**
- GORM v1.30.0 - ORM for database operations
- PostgreSQL driver v1.5.4 - Production database
- SQLite driver v1.6.0 - Alternative/testing database

**Authentication & Tokens:**
- golang.org/x/oauth2 v0.34.0 - OAuth2 implementation
- golang.org/x/oauth2/google - Google OAuth provider
- golang.org/x/crypto v0.45.0 - Cryptographic operations

**Real-time Communication:**
- github.com/gorilla/websocket v1.5.3 - WebSocket connections for signaling
- Custom WebRTC implementation with token-based authentication

**Caching & Sessions:**
- github.com/redis/go-redis/v9 v9.17.2 - Redis client for caching and session store

**Testing:**
- Go native testing package (no explicit framework in go.mod)

**Build/Dev:**
- Makefile for build targets
- Docker & Docker Compose for local development

## Key Dependencies

**Critical:**
- github.com/anthropics/anthropic-sdk-go v1.19.0 - Anthropic Claude API integration
- github.com/sashabaranov/go-openai v1.41.2 - OpenAI API integration
- github.com/google/uuid v1.6.0 - UUID generation for users/calls

**Infrastructure:**
- github.com/prometheus/client_golang v1.20.5 - Metrics collection and export
- go.opentelemetry.io/* - Observability/tracing stack (OTEL)
  - go.opentelemetry.io/otel v1.39.0
  - go.opentelemetry.io/otel/sdk v1.25.0
  - go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.25.0
  - go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.50.0

**Utilities:**
- github.com/joho/godotenv v1.5.1 - Environment variable loading
- golang.org/x/time v0.14.0 - Time utilities for rate limiting

## Configuration

**Environment:**
- `.env` file for local development (`.env.example` provided)
- Environment variables for all critical config (database, auth, API keys)
- Structured config in `internal/config/config.go`

**Build:**
- Dockerfile (Alpine Linux multi-stage build)
- docker-compose.yml for PostgreSQL 15 and Redis 7 services
- Makefile with targets: build, run, worker, test, test-coverage, fmt, docker-up, docker-down

## Platform Requirements

**Development:**
- Go 1.24.0
- PostgreSQL 15 (or compatible)
- Redis 7
- Docker & Docker Compose (optional)

**Production:**
- Linux container environment (Alpine-based Docker image)
- PostgreSQL database (configurable via DATABASE_URL)
- Optional: Redis for session store and caching
- Optional: OTEL collector for tracing

## Key Environment Variables

**Database:**
- `DATABASE_URL` - PostgreSQL connection string
- `DB_MAX_CONNECTIONS`, `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS` - Connection pooling
- `DB_CONN_MAX_LIFETIME_SECONDS`, `DB_TIMEOUT` - Connection lifecycle

**Auth & Security:**
- `JWT_SECRET` - Session token signing key
- `OAUTH_STATE_SECRET` - OAuth state encryption
- `OAUTH_STATE_TTL_SECONDS` - OAuth state expiration
- `TOKEN_EXPIRY` - Session token expiration (seconds)

**LLM Providers:**
- `ANTHROPIC_API_KEY` - Anthropic Claude API key
- `OPENAI_API_KEY` - OpenAI API key
- `CUSTOM_LLM_BASE_URL` - Custom LLM endpoint (default: http://localhost:11434/v1 for Ollama)
- `CUSTOM_LLM_MODEL` - Model name for custom provider (default: llama3)

**Redis:**
- `REDIS_ADDR` - Redis host:port
- `REDIS_PASSWORD` - Redis authentication
- `REDIS_DB` - Redis database number
- `SESSION_STORE_MODE` - "redis", "database", or "hybrid"

**Payment & Monetization:**
- `STRIPE_SECRET_KEY` - Stripe API key (configured but not actively used in code)
- `STRIPE_WEBHOOK_SECRET` - Stripe webhook signing
- `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET` - Razorpay credentials (configured but not used)
- `PLATFORM_COMMISSION_PERCENTAGE` - Commission percentage (default: 20%)

**WebRTC & Signaling:**
- `TURN_SERVER`, `TURN_USERNAME`, `TURN_PASSWORD` - TURN server credentials for NAT traversal
- `WEBSOCKET_ALLOWED_ORIGINS` - CORS origins for WebSocket
- `FRONTEND_URL` - Frontend URL for redirects

**Observability:**
- `METRICS_ENABLED` - Enable Prometheus metrics (default: true)
- `OTEL_SERVICE_NAME` - Service name for traces
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint
- `OTEL_EXPORTER_OTLP_INSECURE` - Disable TLS for OTEL (default: true)

**Outbox Pattern:**
- `OUTBOX_POLL_INTERVAL_SECONDS` - Polling frequency (default: 5)
- `OUTBOX_BATCH_SIZE` - Event batch size (default: 10)
- `OUTBOX_MAX_ATTEMPTS` - Retry attempts (default: 5)

**Idempotency:**
- `IDEMPOTENCY_TTL_SECONDS` - Request idempotency cache TTL (default: 86400)

---

*Stack analysis: 2026-02-03*
