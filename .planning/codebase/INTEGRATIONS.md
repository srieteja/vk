# External Integrations

**Analysis Date:** 2026-02-03

## APIs & External Services

**LLM (Language Model) Providers:**
- Anthropic Claude - AI chat completions
  - SDK: github.com/anthropics/anthropic-sdk-go v1.19.0
  - Auth: `ANTHROPIC_API_KEY` env var
  - Implementation: `internal/llm/providers/anthropic.go`
  - Features: Text completion with token tracking, health checks

- OpenAI GPT - Alternative AI provider
  - SDK: github.com/sashabaranov/go-openai v1.41.2
  - Auth: `OPENAI_API_KEY` env var
  - Implementation: `internal/llm/providers/openai.go`
  - Features: Chat completions, fallback support

- Custom LLM-Compatible Provider - Local/self-hosted models
  - SDK: go-openai with custom base URL
  - Auth: `CUSTOM_LLM_BASE_URL`, `CUSTOM_LLM_MODEL` env vars
  - Implementation: `internal/llm/providers/custom.go`
  - Supports: Ollama, vLLM, LocalAI (any OpenAI-compatible endpoint)
  - Default: http://localhost:11434/v1 (Ollama)

**OAuth & Identity:**
- Google OAuth2
  - SDK: golang.org/x/oauth2/google
  - Credentials: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`
  - Implementation: `internal/services/oauth_service.go`
  - Scopes: userinfo.email, userinfo.profile
  - Supports: Both advocate (provider) and client user registration

**Payment Gateways (Configured but not actively used):**
- Stripe
  - Credentials: `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` env vars
  - Model field: `Payment.PaymentGateway` set to "stripe"
  - Status: Foundation in place, webhook secret configured

- Razorpay
  - Credentials: `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET` env vars
  - Status: Configuration present, integration not implemented

## Data Storage

**Databases:**
- PostgreSQL 15
  - Primary production database
  - Connection: `DATABASE_URL` env var (psql://user:pass@host:port/db_name)
  - Client: GORM v1.30.0
  - Driver: gorm.io/driver/postgres v1.5.4
  - Connection pooling: Configurable via DB_MAX_OPEN_CONNS, DB_MAX_IDLE_CONNS
  - Migrations: SQL files in `migrations/` directory + GORM AutoMigrate

- SQLite
  - Alternative database for testing
  - Driver: gorm.io/driver/sqlite v1.6.0
  - Used for: Development and unit testing

**File Storage:**
- Not detected - No cloud storage integration (S3, GCS, etc.)
- Profile images stored as URLs in database (`ProfileImage` field)

**Caching & Session Store:**
- Redis 7
  - Connection: `REDIS_ADDR` env var (host:port format)
  - Client: github.com/redis/go-redis/v9 v9.17.2
  - Authentication: `REDIS_PASSWORD` env var
  - Database: `REDIS_DB` (default: 0)
  - Modes: Can operate in "redis", "database", or "hybrid" via `SESSION_STORE_MODE`
  - Uses:
    - Session storage (when SESSION_STORE_MODE includes redis)
    - LLM response caching (internal/llm/cache/redis.go)
    - Rate limiting state
    - WebSocket pubsub for signaling across instances

## Authentication & Identity

**Auth Provider:**
- Custom implementation (session-based)
  - Token generation: JWT (golang.org/x/crypto)
  - Session storage: PostgreSQL or Redis (configurable)
  - Implementation: `internal/services/auth_service.go`
  - OAuth integration: Google OAuth2 with state validation
  - State security: HMAC-SHA256 signed state tokens with TTL

**Session Management:**
- Dual-mode: Database or Redis backing store
- Token-based authentication via bearer tokens
- Session table: `sessions` with user_id, user_type, token, expires_at
- Fallback: If Redis unavailable, falls back to database-only storage

## Real-time Communication

**WebSocket Signaling:**
- Framework: github.com/gorilla/websocket v1.5.3
- Endpoint: GET /api/ws/signal (public, token-authenticated)
- CORS: Configurable via `WEBSOCKET_ALLOWED_ORIGINS` env var
- Implementation: `internal/services/signaling_service.go`
- Features:
  - SDP offer/answer exchange for WebRTC
  - ICE candidate routing between peers
  - Multi-instance Redis pubsub support for horizontal scaling
  - Call-based client tracking

**WebRTC Media:**
- Custom token generation: `internal/services/webrtc_service.go`
- Token format: JSON with CallID, Type, ExpiresAt, HMAC signature
- Token expiry: 1 hour
- ICE servers: Configurable TURN via `TURN_SERVER`, `TURN_USERNAME`, `TURN_PASSWORD`
- Implementation: Token-based authorization (no third-party SDN)

## Monitoring & Observability

**Error Tracking:**
- Not detected - No external error tracking service (Sentry, Rollbar, etc.)
- Errors logged via internal logger

**Logs:**
- Custom logger implementation: `internal/logger/logger.go`
- Levels: Severe, Info, Finer, Finest
- Configured via `LOG_LEVEL` env var

**Metrics:**
- Prometheus
  - Framework: github.com/prometheus/client_golang v1.20.5
  - Endpoint: GET /metrics
  - Enabled: `METRICS_ENABLED` env var (default: true)
  - Middleware: `internal/middleware/metrics.go` tracks HTTP requests

**Tracing:**
- OpenTelemetry (OTEL)
  - SDK: go.opentelemetry.io/otel v1.39.0
  - Exporter: OTLP HTTP (go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp)
  - Endpoint: `OTEL_EXPORTER_OTLP_ENDPOINT` env var (optional)
  - Gin middleware: go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.50.0
  - Service name: `OTEL_SERVICE_NAME` (default: "vk_backend")
  - TLS: Configurable via `OTEL_EXPORTER_OTLP_INSECURE` (default: true)
  - Status: Only active if OTEL_EXPORTER_OTLP_ENDPOINT is configured

## Event Processing

**Outbox Pattern:**
- Purpose: Transactional event publishing for eventual consistency
- Implementation: `internal/outbox/service.go`, `internal/outbox/worker.go`
- Storage: `outbox_events` table with JSONB payload
- Worker: Separate process (cmd/worker/main.go) that polls and processes events
- Polling: Interval configurable via `OUTBOX_POLL_INTERVAL_SECONDS`
- Batch processing: `OUTBOX_BATCH_SIZE` events per cycle
- Retry logic: `OUTBOX_MAX_ATTEMPTS` with error tracking

**Current Events:**
- payment.initiated - Fired when payment is created
- Payload includes: payment_id, client_id, advocate_id, amount

**Idempotency:**
- Table: `idempotency_keys` with request hash and response cache
- TTL: `IDEMPOTENCY_TTL_SECONDS` (default: 24 hours)
- Prevents duplicate processing of retried requests

## CI/CD & Deployment

**Hosting:**
- Not detected in codebase (deployment infrastructure not included)

**CI Pipeline:**
- Not detected in codebase (no GitHub Actions, GitLab CI, etc. config)

**Docker:**
- Dockerfile: Multi-stage Alpine Linux build
- Base image: golang:1.21-alpine (builder) → alpine:latest (runtime)
- Exposed port: 8080
- docker-compose.yml: PostgreSQL 15 + Redis 7 for local development

## Webhooks & Callbacks

**Incoming:**
- Google OAuth callback: GET /api/auth/google/callback
- Stripe webhooks: Configured (STRIPE_WEBHOOK_SECRET) but no handler implemented

**Outgoing:**
- Not detected - No webhooks sent to external services
- Event processing via outbox pattern (internal only, currently)

## Request Patterns & Middleware

**Idempotency:**
- Middleware: `internal/middleware/idempotency.go`
- Applied to: Payment payment endpoints (/api/client/payment/*)
- Prevents accidental duplicate charges on retry

**Authentication:**
- Middleware: `internal/middleware/auth.go`
- Session token validation
- User context injection (user_id, user_type)

**Request Tracking:**
- Middleware: `internal/middleware/request_id.go`
- Unique ID generation per request for tracing

**Rate Limiting:**
- Framework: golang.org/x/time v0.14.0
- Per-user rate limiting: `internal/llm/ratelimit/limiter.go`
- Config: `USERB_RATE_PER_MINUTE` env var (default: 5.0 requests/min)
- Applied to: LLM endpoints

## Circuit Breaker Pattern

**LLM Provider Fallback:**
- Implementation: `internal/llm/circuit/breaker.go`
- Primary provider: Attempts with circuit breaker
- Fallback provider: Automatic failover if primary fails
- Prevents cascade failures to LLM services

## Caching Strategy

**LLM Response Cache:**
- Storage: Redis (via `internal/llm/cache/redis.go`)
- Key generation: Hash of messages + provider
- TTL: Configurable in LLM client config
- Use case: Avoid duplicate API calls for identical prompts

---

*Integration audit: 2026-02-03*
