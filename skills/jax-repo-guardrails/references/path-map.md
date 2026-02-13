# Path Map

Use this map to route requests quickly.

## Core Areas

- `services/jax-api/`: API gateway, auth, orchestration HTTP handlers.
- `services/jax-orchestrator/`: orchestration flows and client wiring.
- `services/jax-signal-generator/`: strategy signal generation and orchestration trigger path.
- `services/jax-trade-executor/`: execution, position sizing, risk-enforced order flow.
- `services/jax-memory/`: UTCP memory facade.
- `libs/`: shared packages consumed by multiple services.
- `frontend/`: React/Vite app, hooks, data clients, UI components.
- `db/postgres/`: schema and SQL migrations.
- `tools/cmd/ingest/`: knowledge ingest pipeline.
- `tests/golden/`, `tests/replay/`: regression and determinism harnesses.

## Usually Out of Scope Unless Explicitly Requested

- `services/hindsight/` (vendored)
- `Agent0/` (vendored)
- `dexter/` (vendored)
- `Docs/archive/` (historical)
- `node_modules/` and binary artifacts

## Quick Validation Mapping

- service-local Go edit:
  - `go test ./services/<name>/...`
- shared `libs/*` edit:
  - `go test ./libs/<lib>/...`
  - `go test ./services/...` for known dependents
- API contract edit:
  - backend package tests + frontend data/hook tests
- behavior-sensitive logic edit:
  - `go test -v ./tests/golden/... -tags=golden`
  - `go test ./tests/replay/...`
