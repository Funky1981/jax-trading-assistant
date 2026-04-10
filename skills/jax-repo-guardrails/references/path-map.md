# Path Map

Use this map to route requests quickly.

## Core Areas

- `cmd/trader/`: deterministic runtime, execution path, and frontend API handlers.
- `cmd/research/`: orchestration flows, research/backtest paths, and memory tool endpoints.
- `libs/`: shared packages consumed by multiple services.
- `frontend/`: React/Vite app, hooks, data clients, UI components.
- `db/postgres/`: schema and SQL migrations.
- `tools/cmd/ingest/`: knowledge ingest pipeline.
- `tests/golden/`, `tests/replay/`: regression and determinism harnesses.

## Usually Out of Scope Unless Explicitly Requested

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
