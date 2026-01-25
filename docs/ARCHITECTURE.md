# Architecture

## Overview
This project follows a standard **Layered Architecture** (Controller-Service-Repository) pattern, implemented in Go using the Gin web framework. It is designed for high cohesion and low coupling, making it testable and maintainable.

## Directory Structure
*   `cmd/`: Entry point (`main.go`). Wires up dependencies.
*   `internal/config`: Configuration loaded from env vars.
*   `internal/database`: DB connection and schema migration.
*   `internal/models`: Database structs and request/response DTOs.
*   `internal/services`: Business logic (Auth, Payments, Calls, LLM).
*   `internal/middleware`: HTTP middleware (Auth, Logging).
*   `internal/llm`: Specialized module for AI integration.
*   `plans/`: **[NEW]** Stores generated roadmaps and feature plans (e.g. model training guides).

## Core Systems

### 1. Authentication & User Management
*   **Dual User System:** Distinct `advocates` (providers) and `clients` (consumers) tables.
*   **OAuth Strategy:** Google OAuth flow via `internal/services/oauth_service.go`.
    *   Uses `google_id` (nullable) to link accounts.
    *   Auto-creates accounts if they don't exist.
    *   State is HMAC-signed and time-limited (`OAUTH_STATE_SECRET`, `OAUTH_STATE_TTL_SECONDS`).
*   **Session Management:** Opaque session tokens stored in `sessions` table (not JWTs).

### 2. Communication (WebRTC)
*   **Signaling:** REST endpoints (`/api/calls/*`) handle call initiation, acceptance, and ending.
*   **WebSocket Signaling:** `/api/ws/signal` fans out SDP/ICE messages to peers.
    *   Clients authenticate using WebRTC tokens from `/api/calls/token`.
    *   Origin checks are enforced via `WEBSOCKET_ALLOWED_ORIGINS` (default: same-host).
*   **ICE Servers:** Generates tokens for TURN/STUN usage.

### 3. LLM Integration (`internal/llm`)
A robust, fault-tolerant AI service designed for production.
*   **Provider Pattern:** Interfaces allow hot-swapping models.
    *   **Primary:** Custom Local Model (Ollama/vLLM).
    *   **Fallback:** Anthropic Claude (or OpenAI).
*   **Resilience:**
    *   **Circuit Breaker:** Prevents cascading failures if a provider goes down.
    *   **Rate Limiter:** Token bucket algorithm to control usage.
    *   **Fallback Logic:** Automatically switches to backup providers on error.
*   **Caching:** Redis-based caching for frequent queries.

### 4. Payments
*   Tracks transactions between Clients and Advocates.
*   Calculates platform fees and advocate earnings.

### 5. Scalability & Observability
*   **Session Store:** Redis-backed or hybrid session storage via `SESSION_STORE_MODE`.
*   **Signaling Pub/Sub:** Redis fan-out for WebSocket signaling across instances.
*   **Outbox:** Persistent outbox events with a dedicated worker (`cmd/worker`).
*   **Idempotency:** `Idempotency-Key` support for safe retries on write endpoints.
*   **Telemetry:** Prometheus metrics at `/metrics` and optional OpenTelemetry tracing.

## Data Flow
`Request` -> `Middleware` -> `Handler` -> `Service` -> `Repository/DB`

## Database Schema
(See `internal/models/models.go` or `migrations/init.sql` for latest)
*   **Users:** `advocates`, `clients`
*   **Auth:** `sessions`
*   **Business:** `calls`, `payments`
