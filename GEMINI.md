# Project Context: Enterprise Go Backend API

## Overview
This project is a production-ready backend API built in Go, designed for a platform connecting Providers ("Advocates") and Consumers ("Clients"). It features a dual-user authentication system, P2P video calling capabilities, payment processing, and LLM-powered chat functionality.

## Tech Stack
*   **Language:** Go (1.24+)
*   **Web Framework:** Gin
*   **Database:** PostgreSQL (via GORM)
*   **Caching/Rate Limiting:** Redis
*   **Authentication:** JWT, Google OAuth
*   **LLM Integration:** OpenAI, Anthropic
*   **Containerization:** Docker, Docker Compose

## Architecture
The project follows a standard layered architecture:
*   `cmd/main.go`: Application entry point and wiring.
*   `internal/config`: Configuration management using environment variables.
*   `internal/database`: Database connection and schema initialization.
*   `internal/models`: Data structures and database models.
*   `internal/services`: Business logic implementation.
*   `internal/middleware`: HTTP middleware (Authentication, Logging).
*   `internal/llm`: LLM integration service with a provider abstraction.
*   `internal/logger`: Custom logging package.

## Key Features & Routes
*   **Authentication:**
    *   `/api/auth/advocate/*`: Registration/Login for Advocates.
    *   `/api/auth/client/*`: Registration/Login for Clients.
    *   `/api/auth/google/*`: Google OAuth flow.
*   **User Management:**
    *   `/api/advocate/*`: Profile, availability, and earnings management.
    *   `/api/client/*`: Profile management, finding advocates.
*   **Communication:**
    *   `/api/calls/*`: WebRTC signaling (initiate, accept, end calls, token generation).
    *   `/api/llm/chat`: LLM-powered chat interface.
*   **Payments:**
    *   `/api/client/payment/*`: Payment initiation and verification.

## Development Workflow

### Prerequisites
*   Go 1.24+
*   Docker & Docker Compose
*   PostgreSQL (if not using Docker)

### Setup
1.  **Environment:** Copy `.env.example` to `.env` and configure secrets.
2.  **Dependencies:** Run `go mod download`.
3.  **Database:** Start the database using `make docker-up` or configure a local instance. Initialize schema using `migrations/init.sql`.

### Build & Run
*   **Run Locally:** `make run` (Starts server on port 8080 by default)
*   **Build Binary:** `make build`
*   **Docker:** `make docker-up` (Starts Postgres and Redis) / `make docker-down`

### Testing
*   **Run All Tests:** `make test`
*   **Test Coverage:** `make test-coverage`

## Coding Conventions
*   **Logging:** Use the `internal/logger` package. Do not use standard `log` or `fmt.Print` for application logging.
*   **Configuration:** access configuration via `config.LoadConfig()`. Do not hardcode values.
*   **Error Handling:** Services return errors; Handlers map errors to appropriate HTTP status codes and JSON responses.
*   **Database:** Use GORM for database interactions. Ensure models are defined in `internal/models`.

## Important Files
*   `go.mod`: Project dependencies.
*   `Makefile`: Command shortcuts.
*   `internal/config/config.go`: Environment variable mappings.
*   `internal/llm/providers/provider.go`: Interface definition for LLM providers.
*   `docs/API.md`: Detailed API documentation.
