# Codebase Concerns

**Analysis Date:** 2026-02-03

## Tech Debt

### Incomplete WebRTC Token Enhancement

**Issue:** WebRTC tokens lack UserID for peer identification. Currently only CallID is used, making it difficult to track which peer sent messages.

**Files:** `internal/services/signaling_service.go` (line 143 TODO), `internal/services/webrtc_service.go`

**Impact:** Signaling service cannot reliably identify individual peers in multi-party calls. Message attribution becomes ambiguous in group scenarios. Makes implementation of features like speaker identification or selective message routing more complex.

**Fix Approach:**
1. Add `UserID` field to `WebRTCToken` struct in `webrtc_service.go`
2. Update token generation in `GenerateToken()` to include UserID
3. Modify `ValidateToken()` to expose UserID
4. Update signaling service to store and use UserID in `Client` struct and broadcast logic

---

### Unhandled Background Context Usage

**Issue:** Multiple outbox enqueueing operations use `context.Background()` which loses request context and timeout information.

**Files:**
- `internal/services/payment_service.go` (lines 71, 128)
- `internal/services/call_service.go` (lines 53, 107, 161)

**Impact:** Outbox operations run without deadline constraints. If outbox service is slow or Redis is unavailable, these operations may hang indefinitely, blocking request handlers. Loss of trace context and request correlation.

**Fix Approach:**
1. Pass request context or derive with timeout from handler context
2. Create derived context with reasonable timeout (e.g., `context.WithTimeout(ctx, 5*time.Second)`)
3. Handle context deadline exceeded errors gracefully
4. Add trace context propagation for observability

---

### Incomplete Error Handling in Outbox Worker

**Issue:** Outbox worker ignores errors when marking events as completed or failed. Errors are silently discarded with `_ =` pattern.

**Files:** `internal/outbox/worker.go` (lines 129, 141, 148)

**Impact:** If database updates fail (e.g., disk full, connection loss), the worker continues processing as if successful. Events may be retried indefinitely without notification. Silent failures make debugging production issues difficult.

**Fix Approach:**
1. Log all update errors with appropriate severity
2. Implement circuit breaker for repeated database failures
3. Add metrics/alerts for update failures
4. Consider dead-letter queue for events with permanent failures
5. Implement graceful degradation with fallback notification

---

### Global LLM Client State

**Issue:** LLM client is stored in module-level variable, creating implicit dependency and making testing difficult.

**Files:** `internal/services/llm_service.go` (line 19, 85, 126, 161)

**Impact:** Cannot run parallel tests safely. Service initialization errors are not propagated to callers. Handler functions panic if service wasn't initialized. Makes dependency injection and middleware patterns harder to implement.

**Fix Approach:**
1. Inject `llm.Client` through service struct or context
2. Use constructor injection in HTTP handler initialization
3. Add nil checks with appropriate HTTP error responses (already partially done)
4. Implement health check endpoint to verify initialization
5. Move initialization to main() with proper error handling

---

### Incomplete Rate Limiter Cleanup

**Issue:** Rate limiter cleanup logic is naive and may lose state or accumulate memory unbounded if user count approaches 10,000.

**Files:** `internal/llm/ratelimit/limiter.go` (lines 70-82)

**Impact:** Memory leak potential if system has more than 10,000 active users. Cleanup blanks all limiters periodically, causing reset of rate limit counters and temporary allowance spike. Unfair distribution of quota in high-traffic scenarios.

**Fix Approach:**
1. Implement LRU eviction strategy for least-recently-used user limiters
2. Add configurable threshold for cleanup trigger
3. Track last-access time per user limiter
4. Implement graceful degradation that evicts oldest entries first
5. Add metrics for cleanup events and eviction count

---

## Known Bugs

### Response Body Buffering Memory Leak

**Issue:** Idempotency middleware buffers entire response body in memory as string, storing in database for all requests with Idempotency-Key header.

**Files:** `internal/middleware/idempotency.go` (lines 21-35, 97-105)

**Symptoms:** Memory usage grows with response size. Large response handling becomes problematic. Database disk usage accumulates unbounded.

**Trigger:** Send request with Idempotency-Key header and large JSON response body

**Workaround:** Avoid Idempotency-Key header for endpoints with large response bodies. Set aggressive TTL for idempotency records via IDEMPOTENCY_TTL_SECONDS.

**Fix Approach:**
1. Implement streaming response handling for large bodies
2. Store response hash instead of full body for verification
3. Return 304 Not Modified for replayed requests without re-serializing
4. Add content-length threshold check before buffering
5. Implement garbage collection for expired keys

---

### Session Store State Inconsistency

**Issue:** Hybrid session store attempts Redis first, but on Redis failure, falls back to DB. However, on creation, data is written to both. This creates potential for stale Redis cache on retrieval.

**Files:** `internal/sessions/store.go` (lines 127-155)

**Symptoms:** Session created via hybrid store is in both Redis and DB. If Redis is cleared, subsequent requests still work from DB but new Redis lookups succeed with stale data from another instance.

**Trigger:** Create session in hybrid mode, then restart Redis, attempt to look up session

**Workaround:** Clear sessions table when resetting Redis cache. Use DB-only mode if consistency is critical.

**Fix Approach:**
1. Implement cache invalidation on DB writes
2. Add versioning to session records
3. Implement TTL alignment between Redis and DB
4. Add consistency check on mismatches
5. Document fallback strategy in README

---

## Security Considerations

### Hardcoded Database Credentials

**Issue:** Default DATABASE_URL in config contains hardcoded username and password visible in environment and logs.

**Files:** `internal/config/config.go` (line 56), `.env.example` (line 5)

**Risk:** Credentials exposed in git history, logs, and environment variable dumps. Default credentials remain in production if .env not properly secured.

**Current Mitigation:** .env file exists (should be .gitignored), README mentions it exists but security implications not documented.

**Recommendations:**
1. Use database secrets manager (AWS Secrets Manager, HashiCorp Vault)
2. Implement read-only DB credentials for application
3. Audit git history for exposed credentials
4. Add pre-commit hook to prevent .env commits
5. Document secure credential rotation procedure

---

### Weak JWT Secret Default

**Issue:** JWT_SECRET defaults to string "secret" if not provided, shown in config.

**Files:** `internal/config/config.go` (line 51), `.env.example` (line 25)

**Risk:** Anyone with source code can forge sessions using default secret. Production instances without environment override are completely compromised.

**Current Mitigation:** Example file suggests change in production but not enforced.

**Recommendations:**
1. Refuse startup if JWT_SECRET is not set (remove default)
2. Implement mandatory env var validation at startup
3. Add audit logging for all session operations
4. Implement session revocation mechanism
5. Add HSM/KMS support for key storage

---

### WebRTC STUN Server Hardcoded

**Issue:** WebRTC service hardcodes Google STUN servers with no configuration option.

**Files:** `internal/services/webrtc_service.go` (lines 162-167)

**Risk:** Dependent on external service with no fallback. ICE gathering fails if Google servers are unavailable. Potential for third-party traffic analysis (calls visible to Google).

**Current Mitigation:** Comment mentions need for custom TURN servers but not enforced.

**Recommendations:**
1. Add environment variables for STUN/TURN configuration
2. Implement fallback STUN server list
3. Add TURN credential management
4. Support self-hosted TURN servers
5. Add ICE server health checks

---

### Session Token Not Validated on Creation

**Issue:** `generateToken()` in AuthService doesn't validate randomness. Uses only standard rand which may have insufficient entropy in some scenarios.

**Files:** `internal/services/auth_service.go` (lines 224-233)

**Risk:** Token collision possible if random source is weak or reseeded. Session hijacking via guessed tokens.

**Current Mitigation:** Uses crypto/rand which is cryptographically secure.

**Recommendations:**
1. Implement token rotation on every request
2. Add session binding (IP address, User-Agent)
3. Implement short-lived access tokens + refresh tokens
4. Add session activity tracking
5. Implement automatic logout on suspicious activity

---

## Performance Bottlenecks

### Synchronous Signaling Broadcast

**Issue:** WebSocket broadcast to peers is synchronous in message loop, blocking on write errors.

**Files:** `internal/services/signaling_service.go` (lines 199-211)

**Problem:** If one peer is slow to receive messages, entire broadcast blocks. Slow client causes latency for all other peers in call.

**Cause:** Direct socket write without non-blocking send or buffered channel.

**Improvement Path:**
1. Implement non-blocking write with timeout per peer
2. Add per-peer message queue/buffer
3. Implement graceful peer disconnection on timeout
4. Monitor and alert on broadcast latency
5. Consider multiplexing with context deadline per write

---

### Missing Database Indexes

**Issue:** No explicit indexes on frequently queried columns. Query patterns likely trigger full table scans.

**Files:** `internal/models/models.go` (no indexes defined), `internal/database/db.go` (no index migration)

**Problem:** `WHERE email = ?` and `WHERE token = ?` queries perform poorly on large datasets.

**Cause:** Only primary key and unique constraints indexed.

**Improvement Path:**
1. Add composite indexes on `(email, deleted_at)` for advocates/clients
2. Add index on `(token, expires_at)` for sessions
3. Add index on `(aggregate_type, aggregate_id, status)` for outbox events
4. Add index on `(created_at, status)` for idempotency keys
5. Implement query plan analysis in CI/CD

---

### Rate Limiter Map Unbounded Growth

**Issue:** Rate limiter stores one limiter per user ID indefinitely until 10,000 limit, then clears all.

**Files:** `internal/llm/ratelimit/limiter.go` (line 77-78)

**Problem:** Memory grows linearly with unique users. Hard ceiling at 10K users causes thundering herd when map cleared.

**Cause:** No LRU or TTL-based eviction strategy.

**Improvement Path:**
1. Implement token bucket with sliding window per user
2. Add LRU eviction triggered at configurable threshold
3. Track last-access timestamp
4. Implement metrics for eviction events
5. Add distributed rate limiting with Redis for multi-instance deployments

---

### JSON Serialization Overhead in Cache Key

**Issue:** Cache key generation serializes entire message history to JSON then hashes.

**Files:** `internal/llm/client.go` (lines 56, 95, 104-113)

**Problem:** Large conversation histories cause expensive serialization and hash computation on every request.

**Cause:** Naive string-based cache key generation.

**Improvement Path:**
1. Implement incremental hash of messages
2. Use Blake3 or xxHash for faster hashing
3. Cache the cache key if messages unchanged
4. Add metrics for cache key generation time
5. Consider message digest instead of full serialization

---

## Fragile Areas

### Call Lifecycle State Management

**Files:** `internal/services/call_service.go`

**Why Fragile:** Hard-coded state strings ("initiated", "accepted", "completed") with no validation. Multiple functions check status but inconsistently. Missing intermediate states. No state machine to enforce valid transitions.

**Safe Modification:**
1. Define state machine with enum
2. Create validator for state transitions
3. Add pre-condition checks to each state change
4. Implement audit log for all state transitions
5. Add database constraints for state values

**Test Coverage Gaps:**
- Invalid state transitions not explicitly tested
- Race conditions in concurrent state updates not tested
- Missing tests for status edge cases (e.g., ending already-completed call)

---

### Outbox Event Processing

**Files:** `internal/outbox/worker.go`

**Why Fragile:** Handler registration is global map. No validation that handlers are registered for all event types. Unregistered events silently fail. No dead-letter queue for unhandled events.

**Safe Modification:**
1. Validate all event types have handlers on startup
2. Add pre-handler registration checks
3. Implement dead-letter queue for unregistered types
4. Add metrics for unhandled event types
5. Implement event schema validation

**Test Coverage Gaps:**
- Unregistered event type handling not tested
- Transaction failure during locking not tested
- Concurrent attempt updates not tested

---

### WebSocket Connection Management

**Files:** `internal/services/signaling_service.go`

**Why Fragile:** Peer removal doesn't handle concurrent access to connection. sendMessage doesn't protect against use-after-close. No heartbeat/ping to detect stale connections. Memory leak if connection not properly closed on error.

**Safe Modification:**
1. Implement connection state machine (open, closing, closed)
2. Add heartbeat/ping-pong mechanism
3. Use atomic operations for peer list mutations
4. Add connection pooling and cleanup
5. Implement graceful shutdown with timeout

**Test Coverage Gaps:**
- Concurrent peer addition/removal not tested
- Connection close during broadcast not tested
- Redis subscription failure recovery not tested

---

## Scaling Limits

### In-Memory Call Registry

**Current Capacity:** Entire call peer list stored in single process memory (`calls map[string][]*Client`)

**Limit:** Single instance limited by available RAM. With ~1000 peers per call, practical limit around 1000-10000 active calls depending on system resources.

**Scaling Path:**
1. Replace in-memory map with Redis-backed registry
2. Implement distributed peer discovery
3. Add load balancing across signaling instances
4. Implement call affinity routing
5. Add metrics for call/peer count and memory usage

---

### Session Store Capacity

**Current Capacity:** DB + Redis hybrid with no cleanup. Sessions accumulate indefinitely.

**Limit:** Database grows without bound. TTL-based cleanup via idempotency middleware insufficient for production scale.

**Scaling Path:**
1. Implement automatic session cleanup job
2. Add configurable session TTL with hard expiration
3. Use Redis as primary cache with DB as backup
4. Implement distributed session management
5. Add background job for expired session cleanup

---

### Rate Limiter Per-User State

**Current Capacity:** Maps entire unique user ID space in memory.

**Limit:** ~10,000 unique users before map reset. No distributed rate limiting across instances.

**Scaling Path:**
1. Move rate limiting to Redis
2. Implement distributed quota tracking
3. Use token bucket algorithm with shared state
4. Add circuit breaker for rate limiter failures
5. Implement graceful degradation without rate limiting

---

## Dependencies at Risk

### Gorilla WebSocket Without Keepalive

**Risk:** WebSocket connections may silently drop due to intermediary timeouts (proxies, load balancers).

**Impact:** Calls dropped without notification. Users experience sudden disconnect with no recovery mechanism.

**Migration Plan:**
1. Implement ping/pong frames every 30 seconds
2. Add automatic reconnection logic on client
3. Implement graceful degradation with fallback
4. Add metrics for connection drops
5. Consider alternative: gRPC with keepalive or plain HTTP polling for fallback

---

### Custom LLM Provider with Hardcoded Model

**Risk:** Custom LLM model defaulting to "llama3" without validation. If model unavailable, requests fail silently.

**Impact:** Feature completely non-functional if Ollama/local LLM not running.

**Migration Plan:**
1. Add health check on startup
2. Implement fallback to Anthropic if custom unavailable
3. Add model availability validation
4. Implement graceful service degradation
5. Add admin API to switch models at runtime

---

## Missing Critical Features

### No Transaction Isolation for Concurrent Updates

**Issue:** Call state updates (accept, end) use optimistic locking with SELECT + UPDATE. Concurrent requests can cause state conflicts.

**Problem:** Two accept requests for same call may both succeed, creating inconsistency.

**Blocks:** High-volume concurrent call handling scenarios.

**Fix Approach:**
1. Use database-level pessimistic locking (FOR UPDATE)
2. Implement optimistic locking with version field
3. Add unique constraint on state transitions
4. Implement transaction retries on conflicts
5. Add integration tests for concurrent scenarios

---

### No Graceful Shutdown

**Issue:** No signal handler for SIGTERM. Server terminates immediately, dropping active connections.

**Problem:** WebSocket connections abruptly closed. Outbox worker may interrupt mid-transaction. Sessions lost.

**Blocks:** Kubernetes deployments, rolling updates.

**Fix Approach:**
1. Implement context cancellation on SIGTERM
2. Add configurable shutdown timeout (e.g., 30 seconds)
3. Stop accepting new connections
4. Close existing connections gracefully
5. Wait for outbox worker to complete in-flight events

---

### No Request Correlation/Tracing

**Issue:** No request ID propagation or distributed tracing. Difficult to correlate logs across services.

**Problem:** Debugging issues requires manual log analysis. No end-to-end visibility of request flow.

**Blocks:** Monitoring and observability in production.

**Fix Approach:**
1. Implement request ID middleware
2. Add trace context propagation
3. Integrate with OpenTelemetry (already configured but not used)
4. Add structured logging with correlation IDs
5. Implement distributed tracing dashboard

---

## Test Coverage Gaps

### Concurrent Access Patterns

**Untested Area:** All concurrent access patterns in signaling service, rate limiter, and session store.

**Files:**
- `internal/services/signaling_service.go` (concurrent peer add/remove)
- `internal/llm/ratelimit/limiter.go` (concurrent rate limit checks)
- `internal/sessions/store.go` (concurrent create/get)

**Risk:** Race conditions only appear under load. Production bugs from concurrent access.

**Priority:** High - WebSocket and session operations are inherently concurrent.

---

### Error Recovery Paths

**Untested Area:** Redis connection failures, database unavailability, timeout scenarios.

**Files:**
- `internal/services/signaling_service.go` (Redis subscription failures)
- `internal/sessions/store.go` (fallback strategies)
- `internal/outbox/worker.go` (lock acquisition failures)

**Risk:** Unknown behavior under infrastructure failures. May leak resources or hang indefinitely.

**Priority:** High - Production needs robust failure handling.

---

### Edge Cases in State Machines

**Untested Area:** Invalid call status transitions, idempotency key reuse, payment status conflicts.

**Files:**
- `internal/services/call_service.go` (state transitions)
- `internal/middleware/idempotency.go` (key reuse detection)
- `internal/services/payment_service.go` (concurrent payment verification)

**Risk:** Silent failures or data corruption from invalid state changes.

**Priority:** Medium - Affects data consistency.

---

## Code Quality Issues

### Incomplete Error Messages

**Issue:** Many errors are generic ("database error", "failed to create session") without context.

**Files:** Multiple service files

**Impact:** Difficult to diagnose issues. Insufficient logging for debugging.

**Fix:** Add contextual information to errors (which operation, with what parameters).

---

### Mixed Logging Patterns

**Issue:** Both structured logger (`s.logger.Info()`) and standard lib (`log.Println()`, `log.Printf()`) used inconsistently.

**Files:** `internal/services/llm_service.go` (lines 37, 61, 82, 85, 137), `internal/llm/client.go` (lines 61, 82)

**Impact:** Inconsistent log formats. Standard lib logs not captured by log aggregation system.

**Fix:** Use logger instance consistently throughout codebase.

---

*Concerns audit: 2026-02-03*
