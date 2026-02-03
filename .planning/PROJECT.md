# VK Backend

## What This Is

Enterprise-grade Go backend API serving a dual-user video calling platform. Advocates (service providers) and Clients (consumers) connect for video consultations with integrated payments via Stripe/Razorpay. Includes WebRTC signaling, OAuth authentication, session management, and event-driven architecture with outbox pattern.

## Core Value

Secure, reliable foundation for real-money video consultations between advocates and clients — if payment or auth security fails, the entire platform fails.

## Current Milestone: v1.0 Security & Quality Hardening

**Goal:** Transform codebase from functional to production-ready through comprehensive security hardening, logic bug fixes, architectural improvements, and test coverage.

**Target outcomes:**
- All critical security vulnerabilities fixed (SQL injection, exposed credentials, missing rate limiting, weak session management)
- Race conditions and logic bugs resolved (payment verification, session handling, OAuth flows)
- Modular architecture with proper separation of concerns
- Comprehensive test coverage (>70%) including security, edge cases, and integration tests
- Production-ready deployment configuration

## Requirements

### Validated

<!-- Existing functionality that works -->

- ✓ Dual user system (Advocates & Clients)
- ✓ Email/Password + Google OAuth authentication
- ✓ Session management (Redis + PostgreSQL hybrid)
- ✓ WebRTC video calling with signaling service
- ✓ Payment integration (Stripe & Razorpay)
- ✓ Idempotency middleware for payment operations
- ✓ Outbox pattern for event publishing
- ✓ Observability (OpenTelemetry, Prometheus metrics)
- ✓ LLM service with circuit breaker and rate limiting

### Active

<!-- Current milestone scope -->

**Security Hardening:**
- [ ] Remove hardcoded credentials from repository
- [ ] Fix SQL injection vulnerability in location filter
- [ ] Implement rate limiting on authentication endpoints
- [ ] Strengthen session token validation and add revocation
- [ ] Add input validation for all user-provided URLs/data
- [ ] Fix timing attack in OAuth state validation
- [ ] Add authorization checks before payment operations

**Logic & Reliability:**
- [ ] Fix payment verification race condition with proper locking
- [ ] Implement proper error handling for outbox failures
- [ ] Add transactions for OAuth user creation flows
- [ ] Fix call duration calculation for all call states
- [ ] Resolve idempotency key expiration race condition
- [ ] Add database connection health checks

**Architecture & Modularity:**
- [ ] Extract route handlers from 800-line main.go into controllers
- [ ] Eliminate global LLM client state
- [ ] Create service interfaces for dependency injection
- [ ] Separate HTTP concerns from business logic in LLM service
- [ ] Split auth service responsibilities (auth vs sessions)
- [ ] Add request size limits and CORS configuration

**Testing:**
- [ ] Achieve >70% test coverage across all services
- [ ] Add integration tests for critical flows
- [ ] Create security-focused test suite (injection, bypass, hijacking)
- [ ] Add tests for race conditions and concurrent operations
- [ ] Implement load testing for payment and auth paths
- [ ] Add negative test cases for all endpoints

### Out of Scope

- New features (focus is hardening existing functionality)
- UI/Frontend changes
- Database migration from PostgreSQL
- Alternative payment gateways beyond Stripe/Razorpay
- Real-time chat (not currently implemented)

## Context

**Current State:**
- Functional codebase with core features working
- 47 Go files, 9 test files (~19% file coverage)
- Security audit revealed 9 critical/high vulnerabilities
- Architecture has modularity issues (tight coupling, global state)
- Missing test coverage in critical paths (payments, auth, race conditions)

**Technical Environment:**
- Go 1.24
- PostgreSQL (primary datastore)
- Redis (caching, session store)
- WebRTC for P2P video
- Gin web framework
- GORM ORM
- OpenTelemetry + Prometheus observability

**Known Issues:**
- `.env` file contains hardcoded database credentials and weak secrets
- SQL LIKE concatenation in advocate filter creates injection risk
- No rate limiting allows brute force attacks on login/registration
- Payment verification lacks proper locking mechanism
- Session validation doesn't prevent hijacking scenarios

## Constraints

- **Tech Stack**: Must remain Go + PostgreSQL + Redis (validated in production environment)
- **Backward Compatibility**: Pre-production, can make breaking changes to API structure
- **Timeline**: Systematic approach prioritized over speed
- **Dependencies**: Must maintain current OAuth, payment gateway, and WebRTC integrations
- **Security**: All changes must pass security review before deployment

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Hybrid session store (Redis + DB) | Fast lookups with persistent fallback | ✓ Good — provides resilience |
| Outbox pattern for events | Ensures event delivery guarantees | ✓ Good — prevents lost events |
| Idempotency middleware | Prevents duplicate payments | ⚠️ Revisit — has race condition bug |
| Global LLM client | Simplicity of access | ⚠️ Revisit — creates testing/concurrency issues |
| 800-line main.go with inline handlers | Rapid development | ⚠️ Revisit — needs modularization |

---
*Last updated: 2026-02-03 after v1.0 milestone initialization*
