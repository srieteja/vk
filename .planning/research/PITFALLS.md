# Pitfalls Research

**Domain:** Go Backend Security & Quality Hardening
**Researched:** 2026-02-03
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: SQL Injection During Refactoring

**What goes wrong:**
Developers refactor string concatenation to use GORM's query builder but miss unsafe patterns like `LIKE` with string interpolation. In your codebase, line 142 of `advocate_service.go` uses `query = query.Where("LOWER(location) LIKE ?", "%"+filter.Location+"%")` which appears safe with placeholders but the string concatenation of wildcards happens in Go code before parameterization. If `filter.Location` isn't validated, special SQL characters in user input could still cause issues or bypass filters.

**Why it happens:**
The pattern looks safe because it uses `?` placeholders, creating false confidence. Developers focus on fixing obvious string concatenation (`fmt.Sprintf`) but miss that GORM's parameterization doesn't protect against all injection vectors when wildcards are involved.

**How to avoid:**
- Always sanitize user input before any string manipulation, even for parameterized queries
- For LIKE queries, explicitly escape SQL wildcard characters (`%`, `_`) in user input
- Use GORM's built-in escaping: sanitize the input string to remove or escape `%` and `_` before concatenating wildcards
- Add input validation layer that rejects unexpected characters (e.g., only allow alphanumeric + spaces for location searches)

**Warning signs:**
- Any LIKE query with user input
- String concatenation before passing to Where() clause
- Missing input validation on filter parameters
- Test cases don't include SQL special characters in inputs

**Phase to address:**
Phase 1 (Security Hardening - Input Validation)

---

### Pitfall 2: Race Condition in Payment Verification "Fixed" Without Testing

**What goes wrong:**
The payment verification in `payment_service.go` (lines 83-139) uses a database transaction to prevent race conditions when updating payment status and advocate earnings. However, there's no row-level locking. If two webhook callbacks arrive simultaneously for the same transaction, both could read `payment.Status != "completed"` (line 101) before either writes, causing duplicate earnings updates.

**Why it happens:**
Developers see "use a transaction" as the fix for race conditions, but GORM transactions don't automatically lock rows. The check-then-update pattern (read status, modify if not completed) creates a race window. Without explicit locking (`SELECT ... FOR UPDATE`), the transaction only ensures atomicity of the write, not isolation during the read.

**How to avoid:**
- Use pessimistic locking: `tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&payment, ...)`
- Or implement idempotency at the endpoint level using the `idempotency` middleware already in codebase
- Add unique constraint on `transaction_id` with status to prevent duplicate completions at database level
- Write concurrent integration tests that hammer the endpoint with simultaneous requests

**Warning signs:**
- Transaction wraps check-then-update logic without locking
- Payment webhooks lack idempotency keys
- Test suite has no concurrency tests for payment flows
- Logging shows duplicate "Payment verified successfully" messages for same txnID

**Phase to address:**
Phase 2 (Logic & Reliability - Race Conditions)

---

### Pitfall 3: Session Token Validation Without Revocation Support

**What goes wrong:**
Session validation in `middleware/auth.go` (lines 46-56) and `sessions/store.go` checks token expiration but has no revocation mechanism. If a user's session is compromised, there's no way to invalidate it before expiration. The hybrid store (lines 127-156) doesn't support deletion, only creation and retrieval.

**Why it happens:**
Session stores are implemented with the "happy path" in mind (create, validate, expire). Developers forget that security incidents require immediate revocation. The Store interface (lines 19-22) was designed without a `Delete` or `Revoke` method.

**How to avoid:**
- Add `Revoke(ctx, token) error` and `RevokeAllForUser(ctx, userID, userType) error` to Store interface
- Implement in all three stores (db, redis, hybrid)
- Add logout endpoint that calls Revoke
- Add admin endpoint to force-revoke user sessions
- For Redis store, deletion is trivial (DEL command); for DB store, delete row; for hybrid, do both
- Add session version or `revoked_at` timestamp to detect stolen tokens

**Warning signs:**
- No logout endpoint in API
- Session store interface lacks delete methods
- No way to invalidate sessions in emergency
- Security audit flags "no session revocation"

**Phase to address:**
Phase 1 (Security Hardening - Session Management)

---

### Pitfall 4: OAuth State Validation Timing Attack

**What goes wrong:**
In `oauth_service.go` line 262, the state signature validation uses `hmac.Equal()` which is timing-safe, BUT the subsequent checks (lines 271-278) for expiration and payload validity use standard comparisons. An attacker could potentially learn about payload structure through timing differences, though this is a lower severity issue than the SQL injection.

**Why it happens:**
Developers correctly use `hmac.Equal` for the signature but forget that all subsequent validation steps also leak information through timing. The `time.Now().Unix() > payload.ExpiresAt` comparison is timing-safe, but the structure checks might reveal format details.

**How to avoid:**
- Validate all checks before returning any error (collect all failures)
- Return a generic "invalid state" error for all failure modes
- Don't return different error messages for "invalid format", "expired", "invalid signature"
- Add rate limiting to OAuth endpoints to prevent timing attack attempts

**Warning signs:**
- Multiple different error messages returned from ValidateState
- No rate limiting on OAuth callback endpoint
- Error messages reveal internal state structure
- Responses return before all validation complete

**Phase to address:**
Phase 1 (Security Hardening - OAuth Security)

---

### Pitfall 5: Breaking Existing Functionality During Refactoring

**What goes wrong:**
When extracting handlers from 800-line `main.go` into controllers, developers change error handling behavior, status codes, or response formats. Clients that depend on specific error messages or status codes (like mobile apps checking for 401 vs 403) break in production.

**Why it happens:**
No integration tests capture the existing API contract. Developers focus on code structure without documenting current behavior first. Refactoring without characterization tests means you don't know what you're preserving.

**How to avoid:**
- Write integration tests BEFORE refactoring that capture current behavior
- Use golden test files to snapshot current responses (including error cases)
- Run tests with `-update` flag to capture baseline, then refactor, then verify no changes
- Create API contract tests that verify status codes, error message formats, response structures
- Use feature flags to test new handlers alongside old ones before switching
- Monitor error rates in production after deployment

**Warning signs:**
- Refactoring PR has no new tests, only code movement
- Integration test coverage <50%
- Tests only check "happy path" responses
- No tests for error handling behavior
- CI doesn't run integration tests before merge

**Phase to address:**
Phase 3 (Testing - Integration Tests) — MUST run BEFORE Phase 4 (Architecture Refactoring)

---

### Pitfall 6: Global LLM Client Creates Testing Nightmare

**What goes wrong:**
The `var llmClient *llm.Client` global in `llm_service.go` (line 19) makes unit tests impossible to isolate. Tests can't mock the LLM client without affecting other tests. Parallel tests fail unpredictably. Initialization order matters (Start must be called before HandleChat).

**Why it happens:**
Global state seems convenient during rapid development ("just call llmClient.Complete anywhere"). Developers don't realize the testing implications until writing the test suite. By then, the global is used throughout the codebase.

**How to avoid:**
- Pass dependencies explicitly through function/method parameters
- Use dependency injection: make LLMService hold the client, pass service to handlers
- For HTTP handlers, use middleware to inject service into context: `c.Set("llm_service", service)`
- Or use struct-based handlers with dependencies: `type Handlers struct { llmService *LLMService }`
- Never initialize in `Start()` method — use constructor injection
- Test should create fresh service instance with mock client

**Warning signs:**
- `var clientName *Type` at package level in service files
- Tests need to call initialization functions before running
- Tests fail when run in parallel
- Test setup modifies global state
- Mocking requires global variable replacement

**Phase to address:**
Phase 4 (Architecture & Modularity - Dependency Injection)

---

### Pitfall 7: Missing Authorization Checks Before Payment Operations

**What goes wrong:**
`payment_service.go` creates and verifies payments but doesn't check if the clientID or advocateID making the request matches the payment's client/advocate. An attacker could verify another user's payment transaction if they guess the transaction ID.

**Why it happens:**
Business logic services focus on data operations, assuming authorization happens in HTTP handlers. But authorization logic scattered across handlers is easy to miss during refactoring. The service layer should enforce authorization rules, not just the HTTP layer.

**How to avoid:**
- Add userID context to all service methods: `VerifyPayment(txnID, requestingUserID)`
- Check authorization in service layer before any operation: "Is requestingUserID the client or advocate for this payment?"
- Return "not found" instead of "unauthorized" to prevent enumeration
- Add authorization middleware that extracts userID and passes to service
- Write security tests that attempt operations with wrong userID

**Warning signs:**
- Service methods don't take requesting user as parameter
- Authorization checks only in HTTP handlers, not service layer
- Tests don't verify authorization failures
- Endpoints return different errors for "not found" vs "unauthorized"
- Security audit flags "broken object level authorization"

**Phase to address:**
Phase 1 (Security Hardening - Authorization Checks)

---

### Pitfall 8: Idempotency Key Race Condition on Expiration

**What goes wrong:**
The `idempotency/store.go` (lines 34-36) has a `Save` method that creates idempotency keys but no locking. If a request arrives, checks for key (not found), and starts processing while a second request for the same key arrives before the first saves, both could proceed. Additionally, `DeleteExpired` (line 38) could delete a key between check and save.

**Why it happens:**
Check-then-insert pattern without atomic operations. GORM's `Create` doesn't use INSERT ... ON CONFLICT DO NOTHING or upsert patterns. The expiration cleanup job runs concurrently without coordination.

**How to avoid:**
- Add unique constraint on `key` column in database
- Use `INSERT ... ON CONFLICT DO NOTHING` or GORM's `Clauses(clause.OnConflict{DoNothing: true})`
- Check the error: if unique constraint violation, fetch existing record and return its response
- Use `First()` with `FOR UPDATE` lock, then check expiration before inserting
- Stop using time-based expiration; use TTL or reference counting
- Add database index on (key, expires_at) for efficient cleanup queries

**Warning signs:**
- No unique constraint on idempotency key in schema
- DeleteExpired runs on timer without coordination
- Concurrent requests with same key both succeed
- Tests don't verify idempotency under concurrent load
- Duplicate operations logged for single idempotency key

**Phase to address:**
Phase 2 (Logic & Reliability - Idempotency Hardening)

---

### Pitfall 9: Call Duration Calculated Incorrectly for Non-Started Calls

**What goes wrong:**
In `call_service.go` line 151-153, duration is only calculated if `call.StartedAt != nil`. But calls in status "initiated" or "accepted" that never reach "started" could still end up with status "completed" if `EndCall` is called, resulting in duration=0 and no charge calculation.

**Why it happens:**
Call state machine wasn't fully designed. The code assumes accepted → started → completed, but actual user flows (call initiated but never accepted, or accepted but receiver never joined) create edge cases.

**How to avoid:**
- Design explicit state machine with valid transitions: initiated→(accepted|rejected|timeout), accepted→(started|timeout), started→completed
- Prevent invalid transitions: EndCall should fail if status is "initiated" (can't end what never started)
- Add timeout mechanism: scheduled job marks calls as "timeout" if accepted but not started within 5 minutes
- Calculate charge differently for different end states (timeout should not charge)
- Add validation: can't mark completed unless StartedAt exists

**Warning signs:**
- State transitions not documented
- No validation of current state before status change
- Tests don't cover all state transition paths
- Calls stuck in intermediate states in database
- Support tickets about "charged for calls that never happened"

**Phase to address:**
Phase 2 (Logic & Reliability - Call State Machine)

---

### Pitfall 10: Outbox Pattern Errors Silently Ignored

**What goes wrong:**
Throughout services (payment_service.go line 71, 127; call_service.go line 52, 106, 160), outbox.Enqueue errors are ignored with `_ = s.outboxService.Enqueue(...)`. If event publishing fails, the payment completes but downstream systems never get notified, causing data inconsistency.

**Why it happens:**
Developers assume outbox is reliable infrastructure and don't want event failures to block primary operations. The outbox pattern is supposed to handle this, but ignoring errors means you don't know when the outbox itself fails (e.g., database connection issue).

**How to avoid:**
- Log outbox errors at ERROR level: `if err := s.outboxService.Enqueue(...); err != nil { log.Error(...) }`
- Add metrics for outbox failures: `outbox_enqueue_failures_total` counter
- For critical events (payment.completed), consider failing the operation if outbox fails
- Implement circuit breaker for outbox: if enqueueing consistently fails, alert but don't fail operations
- Add monitoring: alert if outbox has unprocessed events older than threshold
- Add admin endpoint to manually replay failed events

**Warning signs:**
- All outbox errors ignored with blank identifier
- No metrics on outbox failure rates
- Monitoring doesn't track outbox processing lag
- Events never arrive at downstream systems but operations succeed
- Support finds "completed payments with no notification events"

**Phase to address:**
Phase 2 (Logic & Reliability - Outbox Error Handling)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Generic error messages to clients | Faster development | Hard to debug client issues; poor UX | MVP only — add specific errors in Phase 1 |
| No rate limiting during refactor | Focus on core bugs | Production brute-force/DoS vulnerability | Never — add before any deployment |
| Skip migration rollback scripts | Faster iteration | Can't rollback deployments safely | Development only — required for production |
| Test after refactor instead of before | Perceived speed gain | Unknown current behavior, likely to break compatibility | Never for hardening — breaks are invisible without baseline |
| Defer transaction isolation testing | Simpler test setup | Race conditions only found in production | Never for payment/financial logic |
| Keep global state "temporarily" | Easier refactor planning | Tests remain flaky and coupled | Never — removing globals harder the longer they exist |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Stripe/Razorpay webhooks | Process webhook without signature verification | Always verify webhook signature before processing; reject invalid signatures |
| OAuth callback | Trust state parameter from URL without validation | Validate HMAC signature, expiration, and nonce; store state in session or signed JWT |
| Redis session store | Assume Redis always available | Use hybrid store pattern (already implemented); handle Redis failures gracefully |
| WebRTC signaling | Store signaling state in HTTP handler context | Use Redis pub/sub for signaling across multiple server instances |
| Payment gateway idempotency | Use request ID as idempotency key | Use payment-specific key (clientID+advocateID+amount+timestamp hash) to prevent duplicate charges |
| Database connection pool | Use default GORM pool settings | Set `SetMaxIdleConns`, `SetMaxOpenConns`, `SetConnMaxLifetime` for connection reuse and health |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| N+1 queries in advocate listings | Slow API response as advocate count grows | Use `Preload()` for relationships or optimize with JOIN queries | >1000 advocates |
| No database connection pooling limits | Exhausted PostgreSQL connections | Set MaxOpenConns to ~80% of PostgreSQL max_connections; add connection health checks | >100 concurrent users |
| Session lookup for every request without caching | High Redis/DB load; slow auth middleware | Already using hybrid store — ensure Redis is properly configured for low latency | >500 req/s |
| Loading full call history without pagination | Memory exhaustion; slow client rendering | Add limit/offset or cursor-based pagination to schedule endpoints | >10k calls per advocate |
| Synchronous LLM calls blocking HTTP handlers | Request timeout; poor user experience | Use async job queue for LLM processing; return job ID immediately; poll for results | LLM latency >5s |
| Outbox worker batch size too small | High CPU usage processing events one-by-one | Batch process 50-100 events per iteration; add backpressure if queue grows too large | >1k events/min |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Storing plaintext secrets in `.env` in repository | Credential exposure if repo is leaked | Use secrets manager (AWS Secrets Manager, HashiCorp Vault); rotate exposed secrets immediately |
| Bcrypt cost too low (<12) | Fast brute-force attacks on leaked password hashes | Use bcrypt.DefaultCost (14 in Go) for new passwords; add migration to re-hash old passwords |
| No rate limiting on auth endpoints | Account enumeration and brute-force attacks | Add per-IP and per-email rate limiting (e.g., 5 login attempts per 15 minutes) |
| Session tokens with predictable format | Session hijacking through token guessing | Use cryptographically secure random (crypto/rand, 32+ bytes); already implemented correctly in auth_service.go |
| CORS allows all origins in production | XSS and CSRF attacks | Whitelist specific frontend origins; never use `Access-Control-Allow-Origin: *` in production |
| Missing input validation on URLs (profile images, webhooks) | SSRF attacks allowing internal network scanning | Validate URL schemes (only https://); block private IP ranges (RFC1918); validate domain against allowlist |
| Password reset tokens without expiration | Indefinite token validity | Use short-lived tokens (15-60 minutes); one-time use; tie to current password hash |
| Logging sensitive data (passwords, tokens, payment details) | Credential leakage through logs | Sanitize logs; never log plaintext passwords or full tokens (log last 4 chars only) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Generic "database error" returned to client | User doesn't know what went wrong or how to fix | Return specific errors: "Email already registered", "Invalid credentials", "Call not found" (already doing this well) |
| Payment verification webhook fails silently | User charged but call not enabled | Add retry mechanism; user notification if payment processing delayed; support dashboard to manually verify |
| Call "accepted" but advocate never joins | Client waits indefinitely | Add timeout (5 minutes); auto-cancel and refund if advocate doesn't start call |
| Session expires during active call | Call disconnects; poor experience | Extend session TTL on activity; refresh token mechanism for long sessions |
| OAuth flow fails with no error message | User stuck on blank screen | Return error query param to redirect URL; show user-friendly error page; log detailed error server-side |
| Profile image upload fails with 500 error | User thinks service is broken | Validate file type and size before upload; return specific error: "File too large (max 5MB)" or "Invalid image format" |

## "Looks Done But Isn't" Checklist

- [ ] **OAuth implementation:** Often missing state validation, token refresh, and error handling for revoked tokens — verify HMAC signature and expiration
- [ ] **Payment verification:** Often missing duplicate webhook handling and reconciliation — verify idempotency keys and row-level locking
- [ ] **Session management:** Often missing revocation mechanism — verify logout endpoint and force-revoke functionality exist
- [ ] **Input validation:** Often missing on "safe" fields like email, name, location — verify regex validation and SQL character escaping
- [ ] **Rate limiting:** Often only on "obvious" endpoints like login — verify also on registration, password reset, OAuth initiation, expensive queries
- [ ] **Error handling:** Often returns 500 for all errors — verify specific status codes (400, 401, 403, 404, 409, 422) and error messages
- [ ] **Database transactions:** Often used without locking — verify SELECT FOR UPDATE or optimistic locking for check-then-update patterns
- [ ] **Metrics and monitoring:** Often only for uptime — verify business metrics (payments completed, calls started, failed logins) and error rates
- [ ] **Integration tests:** Often only happy path — verify negative tests, concurrent requests, timeout handling, and external service failures
- [ ] **Migration scripts:** Often only "up" migrations — verify rollback scripts exist and are tested

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| SQL injection in production | HIGH | 1. Deploy input validation fix immediately. 2. Audit database for suspicious queries in logs. 3. Check for data exfiltration. 4. Reset credentials if compromised. 5. Public disclosure if PII leaked. |
| Payment race condition caused duplicate charges | HIGH | 1. Deploy locking fix immediately. 2. Query database for duplicate payments (same txnID with multiple completions). 3. Issue refunds for duplicates. 4. Contact affected advocates/clients. 5. Add monitoring to detect future occurrences. |
| Session revocation missing during security incident | MEDIUM | 1. Deploy session revocation feature immediately. 2. Manually delete session records from database/Redis. 3. Force password reset for affected users. 4. Notify users of security incident. |
| OAuth timing attack discovered | MEDIUM | 1. Deploy timing-safe validation. 2. Add rate limiting. 3. Audit logs for suspicious OAuth attempts. 4. Rotate OAuth state secret. 5. Monitor for unusual patterns. |
| Global LLM client causing test failures | LOW | 1. Refactor to dependency injection pattern. 2. Update all tests to use mocks. 3. Add linter rule to prevent new globals. 4. Document dependency injection pattern in CONTRIBUTING.md. |
| Authorization missing on payment endpoint | HIGH | 1. Deploy authorization checks immediately. 2. Audit logs for unauthorized payment operations. 3. Contact affected users. 4. Add security tests to CI. 5. Security code review for other missing authz. |
| Outbox errors silently ignored | MEDIUM | 1. Add error logging and metrics. 2. Query outbox table for stuck events. 3. Implement manual replay mechanism. 4. Process backlog of failed events. 5. Add monitoring alerts for outbox lag. |
| Breaking API changes deployed during refactor | MEDIUM | 1. Identify breaking changes. 2. Deploy rollback or forward fix. 3. Notify API consumers. 4. Add integration tests to prevent recurrence. 5. Document API contract. |
| Call duration miscalculated | LOW | 1. Add state machine validation. 2. Query for calls with duration=0 but status=completed. 3. Manually review for incorrect charges. 4. Issue refunds if needed. 5. Add tests for all state transitions. |
| Idempotency race condition | MEDIUM | 1. Add unique constraint on key column. 2. Update code to handle constraint violations. 3. Query for duplicate operations. 4. Fix data inconsistencies. 5. Add concurrent integration tests. |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| SQL Injection (LIKE wildcards) | Phase 1 | Security test suite attempts injection; penetration testing |
| Payment race condition | Phase 2 | Concurrent integration tests pass; load test with simultaneous webhooks |
| Session revocation missing | Phase 1 | Logout endpoint exists; revoke API tested; security audit passes |
| OAuth timing attack | Phase 1 | All validation errors return generic message; rate limiting active |
| Breaking changes during refactor | Phase 3 (before Phase 4) | Integration tests capture baseline; golden tests show no changes after refactor |
| Global LLM client | Phase 4 | All services use dependency injection; tests run in parallel successfully |
| Missing authorization | Phase 1 | Authorization tests for all endpoints pass; security scan shows no broken object access |
| Idempotency race condition | Phase 2 | Unique constraint exists; concurrent tests with same key handle correctly |
| Call duration miscalculation | Phase 2 | State machine tests cover all transitions; invalid transition attempts fail |
| Outbox errors ignored | Phase 2 | Metrics show outbox failure rates; monitoring alerts on lag; error logs exist |

## Sources

This research is based on analysis of the existing codebase:
- `/Users/srie/work/vk/internal/services/` (auth_service.go, payment_service.go, advocate_service.go, oauth_service.go, call_service.go, llm_service.go)
- `/Users/srie/work/vk/internal/middleware/auth.go`
- `/Users/srie/work/vk/internal/sessions/store.go`
- `/Users/srie/work/vk/internal/idempotency/store.go`
- `/Users/srie/work/vk/.planning/PROJECT.md`

Common Go security and refactoring patterns drawn from:
- Go's database/sql and GORM documentation for SQL injection prevention
- OWASP Top 10 for API Security (2023)
- Go concurrency patterns and common race condition pitfalls
- Payment processing best practices (idempotency, webhooks, locking)
- OAuth 2.0 security best practices (RFC 6819)

All findings are HIGH confidence based on direct code analysis of the existing codebase.

---
*Pitfalls research for: VK Backend Security & Quality Hardening*
*Researched: 2026-02-03*
