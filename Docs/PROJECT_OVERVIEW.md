# Jax Trading Assistant — Project Overview

## Purpose

Jax Trading Assistant is a modular monolith with two active Go runtimes:

- `cmd/trader`: production-facing runtime for deterministic trading flows and frontend API endpoints.
- `cmd/research`: research runtime for orchestration, backtests, and memory tools.

The system integrates with external Python services where appropriate:

- `services/ib-bridge` (market connectivity)
- `services/agent0-service` (planning/execution assistant)
- `services/hindsight` (memory backend)

## Active Runtime Topology

- `jax-trader`
  - API health and frontend API: `http://localhost:8081/health`
  - Trader runtime port: `8100`
  - Source: `cmd/trader`
- `jax-research`
  - Health: `http://localhost:8091/health`
  - Source: `cmd/research`
- `ib-bridge`
  - Health: `http://localhost:8092/health`
  - Source: `services/ib-bridge`
- `agent0-service`
  - Health: `http://localhost:8093/health`
  - Source: `services/agent0-service`
- `hindsight`
  - API: `http://localhost:8888`
  - Source: `services/hindsight`
- Frontend
  - Dev server: `http://localhost:5173`
  - Source: `frontend`

## Repository Map (Current)

- `cmd/`
  - `trader/`: production runtime + frontend-facing API handlers
  - `research/`: orchestration/research runtime + memory proxy/tools
  - `artifact-approver/`, `shadow-validator/`, `jax-utcp-smoke/`: support tooling
- `internal/`
  - Shared runtime modules (artifacts, orchestration, persistence, providers)
- `libs/`
  - Reusable clients/adapters (auth, market data, agent integrations, UTCP)
- `services/`
  - External service boundaries intentionally retained (`ib-bridge`, `agent0-service`, `hindsight`)
- `frontend/`
  - React dashboard consuming trader/research APIs
- `db/postgres/migrations/`
  - Runtime schema and migrations
- `Docs/`
  - ADRs, status, roadmap, runbooks, and archived reports

## Architecture Guardrails

- Trader must stay deterministic and must not import research-only dependencies.
- Research runtime may integrate Agent0/Dexter/Hindsight paths.
- Artifact promotion requires trust-gate evidence (Gate2 deterministic replay + Gate3 promotion checks).
- External Python services remain explicit boundaries; do not collapse them without ADR-level change.

## Validation Baseline

- Go changes: `gofmt` + targeted `go test` for touched packages.
- Frontend changes: targeted `vitest`/`e2e` around affected API-facing flows.
- Behavior-sensitive runtime changes: golden/replay verification before and after edits.

## Primary Docs

- `Docs/QUICKSTART.md` for local startup.
- `Docs/STATUS.md` for current snapshot.
- `Docs/ROADMAP.md` for active priorities.
- `Docs/TODO.md` for tracked remaining work.
