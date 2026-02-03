# Requirements: VK Backend

**Defined:** 2026-02-03
**Core Value:** Secure, reliable foundation for real-money video consultations

## v1.0 Requirements

Requirements for Security & Quality Hardening milestone. Each maps to roadmap phases.

### Security (SEC)

- [ ] **SEC-01**: SQL injection vulnerability in location filter fixed with whitelist validation
- [ ] **SEC-02**: Input validation implemented for all user-provided data (URLs, text fields, file paths)
- [ ] **SEC-03**: Rate limiting implemented on authentication endpoints (login, register, OAuth)
- [ ] **SEC-04**: Session revocation mechanism added to invalidate compromised tokens
- [ ] **SEC-05**: Session token validation strengthened against hijacking attacks
- [ ] **SEC-06**: OAuth state validation timing attack fixed with constant-time comparison
- [ ] **SEC-07**: Authorization checks verify resource ownership in all service methods
- [ ] **SEC-08**: CORS configuration added for allowed origins
- [ ] **SEC-09**: Request size limits enforced to prevent DoS attacks
- [ ] **SEC-10**: Sensitive data (PII, credentials) removed from logs or masked

### Logic & Reliability (REL)

- [ ] **REL-01**: Payment verification race condition fixed with row-level locking (SELECT FOR UPDATE)
- [ ] **REL-02**: Duplicate advocate earnings prevented with transaction isolation
- [ ] **REL-03**: Idempotency key race condition fixed with unique database constraints
- [ ] **REL-04**: Outbox event failures handled with proper error logging and metrics
- [ ] **REL-05**: OAuth user creation and session creation made atomic with transactions
- [ ] **REL-06**: Call state machine validates all transitions and prevents invalid states
- [ ] **REL-07**: Call duration calculation fixed for all call states (including unaccepted)
- [ ] **REL-08**: Database connection health check added after initialization
- [ ] **REL-09**: Session store fallback errors surfaced instead of silently masked
- [ ] **REL-10**: Graceful shutdown implemented for clean termination of requests

### Architecture & Modularity (ARCH)

- [ ] **ARCH-01**: Route handlers extracted from main.go into separate controller packages
- [ ] **ARCH-02**: Service interfaces defined for all business logic services
- [ ] **ARCH-03**: Dependency injection implemented using interfaces and constructor injection
- [ ] **ARCH-04**: Global LLM client eliminated and passed via dependency injection
- [ ] **ARCH-05**: HTTP request/response handling separated from business logic in services
- [ ] **ARCH-06**: AuthService split into separate Auth and Session services
- [ ] **ARCH-07**: Layered architecture established (handlers → services → repositories)
- [ ] **ARCH-08**: Context propagation implemented for request-scoped data
- [ ] **ARCH-09**: Centralized error handling with consistent error responses

### Testing (TEST)

- [ ] **TEST-01**: Integration tests implemented for authentication flow (register, login, OAuth)
- [ ] **TEST-02**: Integration tests implemented for payment flow (initiate, verify, earnings update)
- [ ] **TEST-03**: Integration tests implemented for call flow (initiate, accept, end, duration)
- [ ] **TEST-04**: Security test suite for SQL injection attempts
- [ ] **TEST-05**: Security test suite for authentication bypass attempts
- [ ] **TEST-06**: Security test suite for session hijacking scenarios
- [ ] **TEST-07**: Security test suite for authorization bypass attempts
- [ ] **TEST-08**: Race condition tests for concurrent payment verification
- [ ] **TEST-09**: Race condition tests for idempotency key conflicts
- [ ] **TEST-10**: Negative test cases for all API endpoints (invalid inputs, missing fields)
- [ ] **TEST-11**: Edge case tests (boundary values, null handling, empty strings)
- [ ] **TEST-12**: Load tests for payment endpoints under concurrent load
- [ ] **TEST-13**: Load tests for authentication endpoints under high request volume
- [ ] **TEST-14**: Test coverage measurement configured and enforced at 80%+ threshold
- [ ] **TEST-15**: Race detection enabled in CI with `go test -race`

## Future Requirements

Deferred to post-hardening milestones.

### Security (Deferred)

- **SEC-FUTURE-01**: Hardcoded credentials removed from repository and .env excluded from git
- **SEC-FUTURE-02**: Secret management system implemented (AWS Secrets Manager, HashiCorp Vault, or similar)

### Operations & Observability

- **OPS-01**: Distributed tracing correlation across service boundaries
- **OPS-02**: Enhanced metrics for business KPIs (conversion rates, success rates)
- **OPS-03**: Audit logging for financial transactions (compliance requirement)
- **OPS-04**: API versioning strategy for backward compatibility
- **OPS-05**: Automated deployment pipeline with rollback capability

### Developer Experience

- **DX-01**: API documentation generation from code annotations
- **DX-02**: Pre-commit hooks for code quality and security checks
- **DX-03**: Developer environment setup automation
- **DX-04**: Performance baseline and regression detection

## Out of Scope

Explicitly excluded from this milestone.

| Feature | Reason |
|---------|--------|
| New features | Focus is hardening existing functionality, not adding capabilities |
| UI/Frontend changes | Backend-only hardening milestone |
| Database migration | PostgreSQL + Redis stack is validated and sufficient |
| Alternative payment gateways | Stripe/Razorpay already integrated, focus on security |
| Microservices architecture | Premature for team size, monolith is appropriate |
| Service mesh (Istio, Linkerd) | Over-engineering for current scale |
| Event sourcing | Massive complexity, outbox pattern sufficient |
| Real-time chat | Not currently implemented, out of scope |
| Mobile app | Web-first API, mobile is future consideration |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| *(To be filled by roadmapper)* | | |

**Coverage:**
- v1.0 requirements: 44 total
- Security: 10 requirements
- Logic & Reliability: 10 requirements
- Architecture: 9 requirements
- Testing: 15 requirements
- Mapped to phases: (pending roadmap creation)
- Unmapped: 44 ⚠️ (roadmap not yet created)

---
*Requirements defined: 2026-02-03*
*Last updated: 2026-02-03 after initial definition*
