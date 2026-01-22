# 11 — Docker Compose (Local Dev)

**Goal:** one command starts everything.

Services:
- hindsight (memory)
- jax-memory (facade)
- jax-ingest
- jax-orchestrator
- optional: postgres (if you want a relational store too)
- optional: grafana/prometheus (later)

## Environment variables
- `HINDSIGHT_URL`
- `LOG_LEVEL`
- `RUN_MODE=dev|test`

## TDD note
- docker compose is for **integration tests**, not unit tests.
