# Jax Trading Assistant — Project Overview

## Purpose

Jax Trading Assistant is a multi-service system for trading workflows that combines:

- **UTCP tools** (local and HTTP) for market, risk, storage, and backtest operations.
- **Memory** via Hindsight (vendored) with a Jax Memory facade service.
- **Research/agent ingestion** via Dexter (vendored) and Agent0 references.

Specs and build plan live in `Docs/backend/` with a step-by-step roadmap. Supporting templates/checklists live in `.serena/`.

---

## High-level architecture

This repo follows **Clean Architecture / Hexagonal** principles:

- **Service boundaries**: each service lives under `services/<name>/` and has its own `internal/` packages.
- **Shared libraries**: stable code is placed in `libs/` and should be well-tested.
- **Layering rules** (per service):
  - `internal/domain` has no dependencies on other layers.
  - `internal/app` can depend on `internal/domain` only.
  - `internal/infra` implements adapters and can depend on `internal/app` and `internal/domain`.

---

## Repository map (key areas)

- **services/**
  - `jax-api/`: HTTP API service (current focus). Entrypoint: `go run ./services/jax-api/cmd/jax-api`
    - Endpoints: `GET /health`, `POST /risk/calc`, `GET /strategies`, `POST /symbols/{symbol}/process`, `GET /trades`, `GET /trades/{id}`
  - `jax-memory/`: UTCP memory facade service. Entrypoint: `go run ./services/jax-memory/cmd/jax-memory`
    - UTCP endpoint: `POST /tools` supporting `memory.retain`, `memory.recall`, `memory.reflect`
    - Uses `HINDSIGHT_URL` if set; otherwise in-memory store.
  - `jax-orchestrator/`: pipeline service (skeleton)
  - `jax-ingest/`: ingestion service (skeleton)
  - `hindsight/`: vendored upstream memory backend (pinned; see UPSTREAM.md)

- **libs/**
  - `utcp/`: UTCP client, local tools, Postgres storage adapter
  - `contracts/`: shared DTOs/interfaces (WIP)
  - `observability/`: logging/tracing helpers (WIP)
  - `testing/`: shared fakes/fixtures (WIP)

- **config/**
  - `providers.json`: UTCP provider definitions (http/local)

- **db/postgres/**
  - `schema.sql`: Postgres schema for storage provider
  - `root-files/docker-compose.yml`: main docker compose for services

- **cmd/**
  - `jax-utcp-smoke/`: UTCP smoke test entrypoint

- **vendored/**
  - `dexter/`: Dexter repo (research agent)
  - `Agent0/`: Agent0 repo (reference/inspiration)

- **Docs/**
  - `backend/`: numbered build plan steps (01-12)
  - `frontend/`: UI documentation
- **.serena/**
  - `checklists/`: done criteria for each step
  - `templates/`: copy/paste snippets

---

## Data & tool flows (summary)

1. **UTCP providers** are defined in `config/providers.json` and used by `libs/utcp`.
2. **Jax API** serves trading-related endpoints and uses shared libs.
3. **Jax Memory** exposes UTCP memory tools, backed by Hindsight or an in-memory store.
4. **Dexter/Agent0** integration is planned for ingestion and agent workflows (see Docs/backend steps 06-08).

---

## Testing & quality

- **Primary**: `go test ./...` or `make -f root-files/Makefile test`
- **Lint**: `golangci-lint run ./...`
- **Scripted**: `scripts/test.ps1` (gofmt + lint + tests)

Dexter tests:

- `cd dexter; bun install; bun test`

---

## Local Postgres (optional)

- `docker compose -f db/postgres/docker-compose.yml up -d`
- Apply schema: `db/postgres/schema.sql`
- Example env:
  - `JAX_POSTGRES_DSN=postgres://jax:jax@localhost:5432/jax?sslmode=disable`

---

## Key docs to read next

- `Docs/backend/02_Repository_Scaffold_and_Service_Skeletons.md`
- `Docs/backend/03_Add_Hindsight_and_Memory_Service.md`
- `Docs/backend/05_go_UTCP_Memory_Tools.md`
- `Docs/backend/06_Agent0_Wiring_With_Memory.md`
- `Docs/backend/07_Dexter_Ingestion_to_Memory.md`

---

## Notes

- The repo is actively scaffolding with WIP modules; consult `Docs/backend/` for the authoritative implementation plan.
- Vendored upstreams are pinned; see each `UPSTREAM.md` for commit references.
