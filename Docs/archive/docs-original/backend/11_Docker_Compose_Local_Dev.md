# 11 — Docker Compose (Local Dev)

**Goal:** one command starts the core backend stack.

Primary compose file: `docker-compose.yml` at repo root.

Default services:
- hindsight (memory backend)
- jax-memory (facade)
- jax-api (HTTP API)

Profiles:
- `db`: postgres (optional relational store)
- `jobs`: jax-ingest and jax-orchestrator (batch jobs)

Notes:
- `jax-ingest` expects `./data/dexter.json` (mounted into the container at `/data/dexter.json`).
- `jax-orchestrator` uses `JAX_SYMBOL` (defaults to `AAPL`) when run via the `jobs` profile.

## Environment variables
- `HINDSIGHT_API_LLM_PROVIDER`
- `HINDSIGHT_API_LLM_API_KEY`
- `HINDSIGHT_URL`
- `JAX_SYMBOL`

## Commands
- `docker compose up -d`
- `docker compose --profile db up -d`
- `docker compose --profile jobs up`

## TDD note
- docker compose is for **integration tests**, not unit tests.
