# Roadmap: VK Backend v1.0 Security & Quality Hardening

## Overview

Transform functional Go backend into production-ready platform through systematic hardening across four critical dimensions: security vulnerabilities (SQL injection, rate limiting, session management), logic bugs (payment race conditions, state machines), comprehensive testing (integration, security, race conditions), and architectural modularity (dependency injection, layered structure). The journey prioritizes security first, establishes reliability, captures baseline behavior in tests, then refactors for maintainability — ensuring no breaking changes to existing API contracts.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3, 4): Planned milestone work
- Decimal phases (e.g., 2.1): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Security Hardening** - Fix critical vulnerabilities before any other work
- [ ] **Phase 2: Logic & Reliability** - Resolve race conditions and state machine issues
- [ ] **Phase 3: Testing Foundation** - Establish comprehensive test coverage as refactoring safety net
- [ ] **Phase 4: Architecture & Modularity** - Refactor into maintainable layered structure

## Phase Details

### Phase 1: Security Hardening
**Goal**: Establish security baseline by eliminating critical vulnerabilities in authentication, authorization, input validation, and session management

**Depends on**: Nothing (first phase)

**Requirements**: SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06, SEC-07, SEC-08, SEC-09, SEC-10

**Success Criteria** (what must be TRUE):
  1. SQL injection attacks fail when attempted on location filter and all other user inputs
  2. Brute force attacks are blocked by rate limiting on login, register, and OAuth endpoints
  3. Compromised session tokens can be revoked immediately via admin action
  4. Authorization checks prevent users from accessing or modifying resources they don't own
  5. Request size limits prevent DoS attacks via large payload submissions

**Plans**: TBD

Plans:
- [ ] *Plans to be created during phase planning*

### Phase 2: Logic & Reliability
**Goal**: Eliminate race conditions and logic bugs that cause financial inconsistencies or invalid application states

**Depends on**: Phase 1

**Requirements**: REL-01, REL-02, REL-03, REL-04, REL-05, REL-06, REL-07, REL-08, REL-09, REL-10

**Success Criteria** (what must be TRUE):
  1. Concurrent payment verification webhooks cannot duplicate advocate earnings
  2. Idempotency keys prevent duplicate operations even under concurrent requests
  3. Call state machine rejects invalid transitions and calculates correct duration for all states
  4. OAuth user creation and session creation succeed or fail atomically together
  5. Health check endpoints accurately report database and Redis connectivity status

**Plans**: TBD

Plans:
- [ ] *Plans to be created during phase planning*

### Phase 3: Testing Foundation
**Goal**: Establish comprehensive test coverage capturing current behavior as safety net for architectural refactoring

**Depends on**: Phase 2

**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06, TEST-07, TEST-08, TEST-09, TEST-10, TEST-11, TEST-12, TEST-13, TEST-14, TEST-15

**Success Criteria** (what must be TRUE):
  1. Integration tests verify authentication, payment, and call flows produce correct API responses
  2. Security test suite demonstrates SQL injection, session hijacking, and authorization bypass attempts fail
  3. Race condition tests verify concurrent payment operations maintain financial integrity
  4. Test coverage reaches 80%+ overall and 95%+ for security-critical services
  5. CI pipeline enforces race detection and coverage thresholds on every commit

**Plans**: TBD

Plans:
- [ ] *Plans to be created during phase planning*

### Phase 4: Architecture & Modularity
**Goal**: Refactor 800-line main.go into layered architecture with dependency injection, verified by Phase 3 tests

**Depends on**: Phase 3

**Requirements**: ARCH-01, ARCH-02, ARCH-03, ARCH-04, ARCH-05, ARCH-06, ARCH-07, ARCH-08, ARCH-09

**Success Criteria** (what must be TRUE):
  1. Route handlers live in separate controller packages, not main.go
  2. Services accept dependencies via constructor injection with interfaces
  3. Tests can run in parallel without global state conflicts or flaky failures
  4. All Phase 3 integration tests still pass with identical API behavior
  5. Error responses follow consistent structure across all endpoints

**Plans**: TBD

Plans:
- [ ] *Plans to be created during phase planning*

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Security Hardening | 0/TBD | Not started | - |
| 2. Logic & Reliability | 0/TBD | Not started | - |
| 3. Testing Foundation | 0/TBD | Not started | - |
| 4. Architecture & Modularity | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-03*
*Last updated: 2026-02-03*
