# Project Context: Enterprise Go Backend API (vk)

## Overview
A production-ready Go backend connecting Providers ("Advocates") and Consumers ("Clients"). Includes P2P WebRTC signaling, GORM-based accounting, and a resilient LLM router.

## Tech Stack
*   **Core:** Go 1.24+, Gin (Web Framework), GORM (PostgreSQL)
*   **Auth:** JWT (Sessions), Google OAuth2
*   **Infrastructure:** Redis (LLM Cache/Rate Limit), Docker
*   **Communication:** WebRTC (Signaling + STUN/TURN tokenization)
*   **AI:** Custom Provider (Ollama/vLLM) -> Anthropic -> OpenAI

## Critical Domain Nuances (Token Efficiency)

### 1. Authentication & OAuth
*   **GoogleID:** Stored as `*string` (nullable) in `Advocate` and `Client` models. **Reason:** Avoids unique constraint violations for email/password users (who have NULL) vs OAuth users.
*   **Sessions:** Token-based, stored in `sessions` table, validated via `AuthMiddleware`.

### 2. Accounting & Payments (`PaymentService`)
*   **Commission:** Default is **20% platform fee** (defined in `config.Config`).
*   **Verification Logic:** Uses `db.Transaction` to atomically:
    1. Update payment status to `completed`.
    2. Increment Advocate `earnings` using `gorm.Expr` (prevents race conditions).
*   **Idempotency:** `VerifyPayment` checks if status is already `completed` to prevent double-crediting.

### 3. LLM Integration (`LLMService`)
*   **Provider Chain:** `Primary: Custom` (Local/Ollama) -> `Fallback: Anthropic`.
*   **Resilience:** Implements Circuit Breaker (stops requests on failures) and Rate Limiting (Token Bucket).
*   **Caching:** Redis-backed caching for identical prompts to save costs/latency.

### 4. Communication (`CallService` / `WebRTCService`)
*   **Flow:** `Initiate` -> `Accept` -> `Get Token` -> `End`.
*   **States:** `initiated`, `accepted`, `completed`, `scheduled`.
*   **Duration:** Calculated at `EndCall` based on `StartedAt` timestamp.
*   **Scheduling:** `Call` model supports `ScheduledAt` for future appointments. Clients can view advocate schedules via `GetAdvocateSchedule`.
*   **Signaling:** Implemented via WebSocket (`/api/ws/signal`). Peers authenticate with a WebRTC token and exchange SDP/ICE messages.

## Directory Map
*   `internal/models`: GORM structs (look here for schema).
*   `internal/services`: Business logic (Accounting, Auth, LLM).
*   `internal/llm`: AI logic (Circuit breakers, Providers).
*   `plans/`: Long-term roadmaps (e.g., `model_training.md`).
*   `tests/unit`: Uses SQLite in-memory (`setup_test.go`).

## Key Environment Variables
*   `CUSTOM_LLM_BASE_URL`: For local model serving (e.g., `http://localhost:11434/v1`).
*   `PLATFORM_COMMISSION_PERCENTAGE`: Controls accounting math (default 20.0).

## Testing Patterns
*   Always use `setupTestDB()` from `setup_test.go` for unit tests.
*   Verify side effects (e.g., checking DB earnings after payment verification).