# Research Summary: Security & Quality Hardening

**Project:** VK Backend - Security & Quality Hardening Milestone
**Domain:** Production-Ready Go Backend for Financial Transaction Platform
**Researched:** 2026-02-03
**Confidence:** HIGH

## Executive Summary

This research addresses hardening a functional but not production-ready Go backend handling payments and authentication. The system has 27 identified issues spanning security vulnerabilities (9 critical/high), logic bugs (6), modularity problems (6), and testing gaps (5). Research focused on security patterns, production practices, testing strategies, and common pitfalls specific to Go backends handling financial transactions and PII.

The recommended approach is layered: first establish architecture foundations (dependency injection, layered structure) to enable testability, then systematically address security vulnerabilities (rate limiting, input validation, secret management), followed by production operations (health checks, graceful shutdown, audit logging), and finally improve testing coverage to 80%+ with emphasis on security and race condition testing. The existing stack (Gin, GORM, Redis, PostgreSQL) is solid; no technology replacement needed. Focus is on patterns, not tools.

Critical risks identified include SQL injection in LIKE queries despite GORM parameterization, payment race conditions requiring row-level locking beyond transactions, missing session revocation mechanisms, OAuth timing attacks in validation logic, and global state preventing test isolation. All risks have well-established mitigation patterns documented in research files. The brownfield nature allows breaking changes but demands baseline integration tests before refactoring to avoid breaking existing API contracts.

## Key Findings

### Recommended Stack (from STACK.md)

**Core security additions to existing infrastructure:**

All security enhancements work with the existing stack (Go 1.24, Gin 1.9.1, GORM 1.30.0, Redis v9, PostgreSQL). No breaking technology changes required.

**Key libraries to add:**
- **ulule/limiter/v3 (v3.11.2):** Redis-backed rate limiting for auth/payment endpoints — prevents brute force, integrates with existing Redis infrastructure
- **bluemonday (v1.0.27):** HTML sanitization for user-generated content — prevents XSS attacks, used by GitHub/GitLab
- **testify (v1.10.0):** Testing assertions and mocking — improves test readability, enables proper mocking
- **validator/v10 (existing):** Input validation — already in dependencies, extend usage to all endpoints
- **golang.org/x/crypto (existing):** Timing-safe comparisons and cryptographic primitives — already used for bcrypt, extend for subtle.ConstantTimeCompare

**Critical patterns:**
1. **Secret management:** Remove hardcoded defaults, fail fast on missing production secrets, never use .env files in production
2. **SQL injection prevention:** Escape LIKE wildcards even with parameterized queries, validate input formats before database operations
3. **Session security:** Add HTTP-only cookies, SameSite attributes, session rotation on login, revocation mechanisms
4. **Payment integrity:** Optimistic locking with version columns, idempotency middleware, row-level SELECT FOR UPDATE
5. **Rate limiting:** Per-endpoint strategy (5/min for auth, 10/min for payments, higher for authenticated general endpoints)

### Expected Features (from FEATURES.md)

Research organized features by production-readiness criticality for a functional-but-not-hardened system.

**Must have (blocking production):**
- Dependency injection via interfaces — eliminates global `llmClient`, enables testing (currently blocking test improvements)
- Request size limits — prevents DoS via large payloads (1MB limit)
- CORS configuration — required for browser clients (currently missing = web app broken)
- Rate limiting (auth/payment) — prevents brute force and resource exhaustion (currently only LLM has it)
- Structured error responses — enables proper client error handling (currently inconsistent)
- Health check endpoints (/health, /ready) — required for orchestrators/load balancers (currently missing)
- Graceful shutdown with connection draining — prevents request failures during deployment (partial implementation exists)
- Request timeouts — prevents resource leaks (currently no timeouts = hanging connections)
- Secret management — separates secrets from code (hardcoded "secret" defaults found)
- Audit logging — compliance requirement for financial transactions (basic logging insufficient)
- Layered architecture — separates concerns (800-line main.go unmaintainable)
- Input validation middleware — prevents injection attacks (manual validation inconsistent)

**Should have (v1.1 - operational maturity):**
- API versioning (/api/v1/) — enables non-breaking changes
- Response pagination — prevents memory issues with large datasets
- Enhanced metrics — endpoint-specific latency/error rates (Prometheus exists, needs extension)
- API documentation — OpenAPI/Swagger generation from code
- Feature flags — gradual rollouts for new capabilities
- Database read replicas — scales read operations (defer until DB CPU >70%)

**Defer (v2.0+ - scale & sophistication):**
- Request/response caching — Redis exists but not needed until proven bottleneck
- Webhook delivery system — outbox pattern exists, wrap with retry/signature for external webhooks
- Background job queue (enhanced) — separate from outbox worker
- Multi-region deployment — defer until global user base
- Blue-green deployment — infrastructure concern, ensure health checks work first

**Anti-features (avoid):**
- Service mesh (Istio/Linkerd) — premature for monolith, massive complexity
- Event sourcing — rewrite entire data layer, current domain doesn't need full history
- CQRS — premature optimization, adds consistency challenges
- Microservices — team too small, monolith manageable with modular structure
- NoSQL replacement — PostgreSQL handles current load fine

### Architecture Approach (from ARCHITECTURE.md)

**Current state:** Functional but not production-structured. 800-line main.go with inline handlers, global state variables, manual validation scattered across services.

**Recommended structure (three-layer architecture):**

1. **Handler Layer (HTTP concerns only)**
   - Request parsing, validation
   - Response formatting
   - HTTP-specific middleware (CORS, rate limiting)
   - Thin layer delegating to services

2. **Service Layer (business logic)**
   - Business rule validation
   - Transaction orchestration
   - Authorization checks (verify requesting user owns resource)
   - Error handling and logging
   - Dependencies injected via interfaces

3. **Repository Layer (data access)**
   - Database queries (GORM)
   - Redis operations
   - External API calls
   - Interface-based for test mocking

**Critical patterns from research:**

- **Dependency injection:** Constructor injection with interfaces (`NewPaymentService(db DB, cache Cache, llm LLMProvider)`)
- **Error handling:** Typed errors with middleware catching and mapping to HTTP codes (centralized `ErrorMiddleware`)
- **Context propagation:** `context.Context` as first parameter everywhere, enables tracing and cancellation
- **Middleware composition:** Explicit chains showing security stack per route group
- **Test fixtures:** Builder pattern for test data (`NewAdvocateBuilder().WithEmail(...).Build()`)

**Transaction patterns:**
- Use GORM transactions for multi-table updates (already implemented)
- Add row-level locking for check-then-update patterns: `Clauses(clause.Locking{Strength: "UPDATE"})`
- Implement optimistic locking with version columns for concurrent webhook handling

### Critical Pitfalls (from PITFALLS.md)

**Top 10 pitfalls with phase mapping:**

1. **SQL injection in LIKE queries (Phase 1 - Security)**
   - Issue: `Where("LOWER(location) LIKE ?", "%"+filter.Location+"%")` concatenates wildcards before parameterization
   - Risk: User input with `%` or `_` bypasses filters or causes logical injection
   - Fix: Escape wildcards (`\%`, `\_`) before concatenation, validate input format (alphanumeric + spaces only)

2. **Payment race conditions despite transactions (Phase 2 - Logic)**
   - Issue: Transaction wraps check-then-update without row locking; concurrent webhooks can duplicate credits
   - Risk: Advocate credited twice for same payment if webhooks arrive simultaneously
   - Fix: Add `SELECT ... FOR UPDATE` locking or optimistic locking with version column, extend idempotency middleware

3. **Session revocation missing (Phase 1 - Security)**
   - Issue: Store interface lacks `Revoke()` method; compromised sessions can't be invalidated before expiration
   - Risk: Stolen sessions remain valid until natural expiry
   - Fix: Add `Revoke(token)` and `RevokeAllForUser(userID, userType)` to Store interface, implement in all stores

4. **OAuth timing attacks in validation (Phase 1 - Security)**
   - Issue: State signature uses timing-safe `hmac.Equal()` but subsequent checks leak timing info
   - Risk: Attacker learns payload structure through timing differences (lower severity but still addressable)
   - Fix: Validate all checks before returning errors, return generic "invalid state" for all failure modes

5. **Breaking changes during refactoring (Phase 3 - Testing, BEFORE Phase 4)**
   - Issue: No integration tests capturing current API behavior; refactoring changes status codes or error messages
   - Risk: Existing clients (mobile apps) break in production
   - Fix: Write integration tests capturing current behavior BEFORE extracting handlers from main.go

6. **Global LLM client prevents test isolation (Phase 4 - Architecture)**
   - Issue: `var llmClient *llm.Client` at package level; tests can't mock without affecting other tests
   - Risk: Flaky tests, parallel execution fails, initialization order matters
   - Fix: Constructor injection with interfaces, pass dependencies explicitly

7. **Missing authorization in service layer (Phase 1 - Security)**
   - Issue: Services don't verify requesting user owns resource; authorization only in scattered handler checks
   - Risk: Attacker verifies another user's payment by guessing transaction ID
   - Fix: Add `requestingUserID` parameter to service methods, check ownership before operations

8. **Idempotency key race condition (Phase 2 - Logic)**
   - Issue: Check-then-insert without unique constraint; concurrent requests with same key can both proceed
   - Risk: Duplicate operations despite idempotency middleware
   - Fix: Add unique constraint on `key` column, use `INSERT ... ON CONFLICT DO NOTHING`, handle constraint violations

9. **Call state machine not enforced (Phase 2 - Logic)**
   - Issue: EndCall can mark completed even if never started; duration=0 for non-started calls
   - Risk: Incorrect billing, calls stuck in invalid states
   - Fix: Design explicit state machine with valid transitions, validate current state before status changes, add timeouts

10. **Outbox errors silently ignored (Phase 2 - Logic)**
    - Issue: `_ = s.outboxService.Enqueue(...)` throughout services; event publishing failures invisible
    - Risk: Payment completes but downstream systems never notified, data inconsistency
    - Fix: Log errors at ERROR level, add metrics, alert on outbox processing lag, consider failing critical operations if outbox fails

**Common recovery costs:**
- SQL injection in production: HIGH (audit logs, check data exfiltration, reset credentials, possible disclosure)
- Payment race condition: HIGH (query duplicates, issue refunds, contact affected users)
- Session revocation missing: MEDIUM (deploy fix, manually delete sessions, force password reset)
- Authorization bypass: HIGH (audit logs, contact affected users, security review)

## Implications for Roadmap

Based on combined research, recommended phase structure addresses security first (compliance/safety), then reliability (race conditions, state machines), then architecture (enables maintainability), then testing (verification).

### Phase 1: Security Hardening (CRITICAL - Week 1-2)

**Rationale:** Address critical/high security vulnerabilities before any other work. Financial transaction platforms must have security baseline before production deployment. These are compliance requirements, not nice-to-haves.

**Delivers:**
- Rate limiting on auth and payment endpoints (prevents brute force, DoS)
- Input validation middleware with validator/v10 (prevents injection, XSS, SSRF)
- SQL injection protection via wildcard escaping (secures LIKE queries)
- Session revocation mechanism (enables emergency response to compromised accounts)
- Secret management hardening (fail fast on missing/default secrets in production)
- Authorization checks in service layer (prevents broken object level access)
- CORS configuration (enables browser clients securely)
- Request size limits (DoS prevention)
- OAuth timing attack mitigation (constant-time comparisons throughout)

**Addresses features:**
- Rate limiting (must have - currently only LLM service has it)
- Input validation (must have - manual validation inconsistent)
- Secret management (must have - hardcoded defaults found)
- CORS (must have - currently missing)

**Avoids pitfalls:**
- Pitfall 1: SQL injection in LIKE queries
- Pitfall 3: Session revocation missing
- Pitfall 4: OAuth timing attacks
- Pitfall 7: Missing authorization checks

**Stack additions:**
- ulule/limiter/v3 for Redis-backed rate limiting
- bluemonday for HTML sanitization
- Extend validator/v10 usage to all endpoints

**Research flag:** NO — well-documented security patterns, established libraries

### Phase 2: Logic & Reliability (CRITICAL - Week 3-4)

**Rationale:** Fix race conditions and state machine issues that cause financial inconsistencies. Must come after Phase 1 (security baseline) but before refactoring (Phase 4) to ensure known behavior before restructuring.

**Delivers:**
- Payment verification race condition fix with row-level locking
- Idempotency middleware extended to payment verification
- Optimistic locking with version columns for concurrent webhooks
- Call state machine enforcement with valid transition rules
- Outbox error handling with logging and metrics
- Graceful shutdown with connection draining (complete partial implementation)
- Health check endpoints (/health with liveness, /ready with DB/Redis probes)
- Request timeouts on http.Server (prevents hanging connections)

**Addresses features:**
- Graceful shutdown (must have - partial implementation exists)
- Health checks (must have - required for orchestrators)
- Request timeouts (must have - prevents resource leaks)

**Avoids pitfalls:**
- Pitfall 2: Payment race conditions
- Pitfall 8: Idempotency key race condition
- Pitfall 9: Call state machine not enforced
- Pitfall 10: Outbox errors silently ignored

**Stack patterns:**
- GORM row-level locking (`Clauses(clause.Locking{Strength: "UPDATE"})`)
- Optimistic locking with version columns
- Database unique constraints for idempotency

**Research flag:** NO — standard concurrency patterns, well-documented locking strategies

### Phase 3: Testing Foundation (CRITICAL - Week 5-6)

**Rationale:** MUST come before Phase 4 (architecture refactoring). Integration tests capture current API behavior to prevent breaking changes during refactoring. Security tests verify Phase 1/2 fixes actually work. This is the safety net for Phase 4.

**Delivers:**
- Integration test suite covering auth, payment, OAuth flows (captures baseline behavior)
- Security test suite (SQL injection, race conditions, timing attacks, session hijacking, auth bypass)
- Test organization (tests/ directory with unit/, integration/, security/, load/ subdirectories)
- testify/assert and testify/mock integration (improves test readability)
- Race detector integration (`go test -race` in CI)
- Coverage measurement and enforcement (80%+ overall, 95%+ for security-critical services)
- Concurrent test patterns for payment verification

**Addresses features:**
- Testing infrastructure (must have - currently 19% coverage)

**Avoids pitfalls:**
- Pitfall 5: Breaking changes during refactoring (integration tests capture baseline BEFORE Phase 4)

**Stack additions:**
- testify/assert and testify/mock (v1.10.0)
- Go stdlib testing, httptest, race detector
- SQLite in-memory for fast test database

**Testing targets:**
- Unit tests: 60% of test pyramid, table-driven for edge cases
- Integration tests: 20%, full stack with real DB/Redis
- Security tests: 15%, specialized for vulnerabilities
- Load tests: 5%, benchmarks and concurrent scenarios

**Research flag:** NO — standard Go testing patterns, established libraries

### Phase 4: Architecture & Modularity (HIGH PRIORITY - Week 7-9)

**Rationale:** Refactor 800-line main.go into layered architecture AFTER establishing test baseline (Phase 3). Tests from Phase 3 verify no breaking changes. Enables long-term maintainability and team scaling.

**Delivers:**
- Three-layer architecture (handlers/, services/, repositories/)
- Dependency injection via constructor injection with interfaces
- Remove global `llmClient` variable
- Centralized error handling middleware (typed errors mapping to HTTP codes)
- Structured error responses (consistent format across all endpoints)
- Middleware composition patterns (explicit security stacks per route group)
- Context propagation enforcement (context.Context first parameter everywhere)
- Service interfaces for testing (mockable dependencies)

**Addresses features:**
- Dependency injection (must have - blocks testing improvements)
- Layered architecture (must have - 800-line main.go unmaintainable)
- Structured error responses (must have - improves client error handling)

**Avoids pitfalls:**
- Pitfall 6: Global LLM client prevents test isolation
- Pitfall 5: Breaking changes during refactoring (prevented by Phase 3 tests)

**Verification:**
- All integration tests from Phase 3 still pass after refactoring
- No changes to API contract (status codes, error messages, response formats)
- Tests run in parallel without flaky failures

**Research flag:** NO — standard Go architecture patterns, dependency injection well-documented

### Phase 5: Production Operations (MEDIUM PRIORITY - Week 10-11)

**Rationale:** Operational improvements that enhance observability and compliance. Can happen in parallel with or after Phase 4. Not blocking production but significantly improves operational maturity.

**Delivers:**
- Audit logging for auth events, payment operations, PII access (compliance requirement)
- Enhanced Prometheus metrics (endpoint-specific latency histograms, error rate counters)
- Distributed tracing verification (OpenTelemetry span propagation working correctly)
- Panic recovery middleware with alerting (verify Gin recovery, add custom handling)
- API versioning (/api/v1/ prefix)
- Centralized configuration management (environment-specific configs)
- Database migration versioning and rollback scripts

**Addresses features:**
- Audit logging (must have - compliance requirement for financial transactions)
- Enhanced metrics (should have - operational maturity)
- API versioning (should have - enables non-breaking changes)

**Stack extensions:**
- Extend existing custom logger with audit-specific methods
- Prometheus histogram metrics for latency
- OpenTelemetry span instrumentation

**Research flag:** NO — standard observability patterns, existing infrastructure in place

### Phase 6: Coverage Improvement (ONGOING - Week 12+)

**Rationale:** Continuous improvement to achieve and maintain 80%+ overall coverage, 95%+ for security-critical services. Establishes quality gates for ongoing development.

**Delivers:**
- 80%+ overall test coverage
- 95%+ coverage for payment, auth, OAuth, WebRTC token services
- Pre-commit hooks enforcing test success and coverage threshold
- CI pipeline enforcing coverage thresholds
- Load tests establishing performance baselines
- Nightly security and load test runs
- Coverage regression detection

**Stack:**
- Go stdlib coverage tools (`go test -cover`, `go tool cover`)
- GitHub Actions workflows for CI
- Benchmark baselines (payment initiation <10ms p99, verification <50ms p99, login <100ms p99)

**Research flag:** NO — standard testing and CI practices

### Phase Ordering Rationale

**Why security first (Phase 1):**
- Financial transaction platform with PII requires security baseline before any other work
- Rate limiting, input validation, secret management are compliance requirements
- Enables safe testing in subsequent phases (test with real attack vectors)

**Why reliability second (Phase 2):**
- Race conditions in payment processing cause financial inconsistencies (refunds, customer support burden)
- Must fix before refactoring (Phase 4) to ensure known correct behavior
- Health checks required for production deployment to orchestrated environments

**Why testing before refactoring (Phase 3 before Phase 4):**
- Integration tests capture current API contract to prevent breaking changes
- Security tests verify Phase 1/2 fixes actually work under adversarial conditions
- Race detector catches concurrency bugs introduced during refactoring
- Golden test files snapshot current responses for regression detection

**Why architecture refactoring after testing (Phase 4 after Phase 3):**
- Tests provide safety net for major restructuring
- Dependency injection enables better testing in future development
- 800-line main.go is technical debt but not blocking production (unlike security issues)

**Why operations in parallel/after architecture (Phase 5):**
- Audit logging and enhanced metrics improve compliance but not blocking
- Can happen alongside Phase 4 refactoring
- Observability improvements valuable but not critical path

### Research Flags

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Security):** Well-documented security libraries (ulule/limiter, bluemonday, validator), OWASP patterns
- **Phase 2 (Reliability):** Standard GORM locking patterns, concurrency control well-documented
- **Phase 3 (Testing):** Go stdlib testing, testify widely adopted, security testing patterns established
- **Phase 4 (Architecture):** Three-layer architecture is Go best practice, dependency injection patterns documented
- **Phase 5 (Operations):** Prometheus, OpenTelemetry integration standard, audit logging patterns clear

**No phases require `/gsd:research-phase`:** All patterns are well-established for Go backends. Research provided comprehensive coverage of security, testing, and architecture patterns specific to this domain.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All libraries verified via pkg.go.dev, version compatibility checked, existing codebase analysis confirms integration feasibility |
| Features | HIGH | Direct codebase analysis identified missing must-haves, production-readiness checklist based on Google SRE practices |
| Architecture | HIGH | Standard Go patterns (three-layer, dependency injection, error handling), verified against Go project layout conventions |
| Pitfalls | HIGH | All pitfalls derived from direct codebase analysis (specific line numbers cited), common Go security/concurrency issues well-documented |

**Overall confidence:** HIGH

All research based on:
1. Direct codebase analysis (specific files and line numbers examined)
2. Official library documentation (pkg.go.dev for all recommended libraries)
3. Industry standards (OWASP API Security Top 10, PCI DSS for financial transactions, GDPR for PII)
4. Go best practices (official Go documentation, Go project layout conventions)
5. Established patterns for production Go backends

No speculative recommendations. All findings grounded in either codebase analysis or official documentation.

### Gaps to Address

**During Phase 1 implementation:**
- Determine specific allowed origins for CORS (development vs production environments)
- Select secrets manager solution based on deployment environment (AWS Secrets Manager, HashiCorp Vault, Kubernetes Secrets)
- Define rate limit thresholds based on expected traffic patterns (research suggests 5/min for auth, 10/min for payments, tune based on monitoring)

**During Phase 2 implementation:**
- Test idempotency integration with payment gateway webhooks (Stripe/Razorpay signature verification patterns)
- Define specific call state transition rules and timeout durations (research suggests 5-minute timeout for accepted→started)

**During Phase 3 implementation:**
- Determine coverage thresholds per package type (research suggests 95% for security-critical, 85% for business logic, 70% for models)
- Establish performance baselines through benchmarking (research provides targets: <10ms payment initiation, <50ms verification)

**During Phase 4 implementation:**
- Decide on package structure details (handlers/ vs internal/handlers/, repository interface naming conventions)
- Define error type hierarchy (AppError subtypes for domain-specific errors)

**During Phase 5 implementation:**
- Define audit log schema and retention policy (compliance-driven, typically 7 years for financial transactions)
- Select monitoring alert thresholds based on established baselines

**No blocking gaps:** All gaps are implementation details that become clear during execution, not architectural uncertainties requiring additional research.

## Sources

### Primary (HIGH confidence)

**Codebase analysis:**
- `/Users/srie/work/vk/go.mod` — dependency versions, existing stack verification
- `/Users/srie/work/vk/internal/services/` — auth_service.go, payment_service.go, advocate_service.go, oauth_service.go, call_service.go, llm_service.go (specific vulnerabilities and patterns analyzed)
- `/Users/srie/work/vk/internal/middleware/` — auth.go, idempotency (existing patterns)
- `/Users/srie/work/vk/internal/sessions/store.go` — session management implementation
- `/Users/srie/work/vk/.planning/PROJECT.md` — 27 identified issues, hardening goals

**Official documentation:**
- pkg.go.dev/golang.org/x/crypto — bcrypt, argon2, crypto/subtle usage patterns
- pkg.go.dev/github.com/go-playground/validator/v10 — validation features, custom validators
- pkg.go.dev/github.com/ulule/limiter/v3 — Redis backend, Gin middleware, rate formats
- pkg.go.dev/github.com/gin-contrib/cors — credential protection, origin validation
- pkg.go.dev/gorm.io/gorm — parameterized queries, locking strategies
- pkg.go.dev/testing — Go testing patterns, race detector, coverage tools
- pkg.go.dev/net/http/httptest — HTTP handler testing

**Industry standards:**
- OWASP API Security Top 10 (2023) — rate limiting, authorization, injection prevention
- PCI DSS requirements — audit logging, secret management, encryption in transit
- GDPR Article 32 — TLS for PII, security incident response
- Go Security Checker (gosec) — static analysis patterns
- Google SRE practices — production readiness checklist

### Secondary (MEDIUM confidence)

**Community resources:**
- GitHub usage of bluemonday (GitHub, GitLab use for XSS prevention)
- Go project layout conventions (three-layer architecture, internal/ package structure)
- Payment processing best practices (idempotency, webhooks, locking)
- OAuth 2.0 security (RFC 6819) — state parameter CSRF protection

**Ecosystem knowledge:**
- testify adoption in Go community (most popular assertion library)
- Gin framework best practices (middleware composition, recovery patterns)
- GORM transaction patterns (common pitfalls with check-then-update)

### Tertiary (LOW confidence)

No tertiary sources used. All recommendations based on direct codebase analysis or official documentation.

---

## Roadmap Creation Guidance

**For gsd-roadmapper agent:**

This SUMMARY.md provides:
- **Phase structure suggestion:** 6 phases with clear rationale and ordering
- **Feature mapping:** Each phase addresses specific features from FEATURES.md
- **Pitfall avoidance:** Each phase explicitly lists which pitfalls it prevents
- **Stack integration:** Technology additions specified per phase
- **Research flags:** All phases marked as NO ADDITIONAL RESEARCH NEEDED (comprehensive coverage provided)

**Key recommendations for roadmap:**
1. Do NOT reorder phases — security must come before refactoring, testing must come before architecture changes
2. Phase 3 (Testing) is CRITICAL before Phase 4 (Architecture) — this prevents Pitfall 5 (breaking changes)
3. All phases have clear deliverables and verification criteria
4. No phases require `/gsd:research-phase` — patterns are well-established

**Confidence level for roadmap creation:** HIGH — proceed directly to requirements definition.

---

*Research completed: 2026-02-03*
*Ready for roadmap: YES*
*Comprehensive coverage: Security patterns, production practices, testing strategies, common pitfalls*
