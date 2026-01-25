# Scalability Plan

This document captures near-term steps to make the system scale cleanly later.

## Stateless Services
- Move signaling state out of process memory (Redis Pub/Sub or dedicated signaling service).
- Store auth sessions in Redis or move to signed JWTs.
- Avoid correctness-critical in-process state across all services.

## Data Layer
- Adopt a migration tool and disable AutoMigrate in production.
- Add indexes based on observed query patterns.
- Add PgBouncer or managed pooling.
- Plan for read replicas and partitioning for `calls` and `payments`.

## Async Workflows
- Introduce a queue + worker service for payments, call lifecycle, and email.
- Add outbox + idempotency keys for payment safety.

## Infrastructure Scaling
- Build immutable container images and deploy with autoscaling.
- Use sticky sessions for WebSockets.
- Use managed Postgres and Redis.

## Observability
- Add OpenTelemetry tracing, Prometheus metrics, and structured logs.
- Standardize request IDs across services.
- Define SLIs and run load tests.

## LLM Isolation
- Run the LLM service as its own deployment with autoscaling.
- Use caching and per-user budgets for cost control.
