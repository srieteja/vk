# Setup Guide

## Quick Start

1.  **Prerequisites:** Ensure Go 1.24+ and Docker are installed.
2.  **Clone & Enter:** `cd <repo-root>`
3.  **Config:** `cp .env.example .env` and fill in your API keys (Google, Anthropic, etc.). Set `WEBSOCKET_ALLOWED_ORIGINS` to your frontend origin and set `OAUTH_STATE_SECRET`.
4.  **Dependencies:** `go mod download`
5.  **Infrastructure:** `make docker-up` (Starts Postgres & Redis)
6.  **Database:** Postgres runs `migrations/init.sql` automatically on first boot. If you need to re-run, use `docker-compose down -v` or execute the SQL manually.
7.  **Run:** `make run`
8.  **Optional Worker:** `make worker` (processes outbox events)

Server will be running at: http://localhost:8080

## LLM Setup (Optional)

To use the AI Chat features:
1.  **Local Model:** Install [Ollama](https://ollama.com) and run `ollama serve`.
2.  **Pull Model:** `ollama pull llama3` (or your custom model).
3.  **Config:** Ensure `CUSTOM_LLM_BASE_URL` in `.env` matches your Ollama URL.

See `plans/model_training.md` for details on training custom models.

## Commands

*   `make run`: Start server
*   `make test`: Run unit tests
*   `make build`: Build binary
*   `make docker-down`: Stop infrastructure

## Testing

*   **Health Check:** `curl http://localhost:8080/health`
*   **LLM Check:** `curl http://localhost:8080/api/llm/health`
