# Project Status (Condensed)

## Snapshot

- **Active runtime layout**: `cmd/trader` serves signal generation plus the frontend-facing API surface, `cmd/research` serves orchestration and research endpoints, `services/agent0-service` remains the external planning/execution service, and `services/ib-bridge` remains the Python market bridge.
- **Core services running**: `cmd/trader`, `cmd/research`, `hindsight` (Dockerized), and the React frontend shell.
- **Major gaps**: real artifact validation still needs golden/replay-backed evidence, market-data ingestion still needs fuller validation, and Agent0/Dexter mock-vs-live boundaries still need tightening.
- **IB market data**: a production-ready Python IB bridge is documented and expected on port `8092`, but upstream integration and ingestion wiring still need validation.

## What Works (High Confidence)

- **cmd/trader frontend API**: health, risk, testing, artifact, run-history, orchestration-proxy, and trades endpoints are exposed from `cmd/trader/frontend_api.go` and `cmd/trader/codex_api.go`.
- **cmd/research**: `/orchestrate`, backtest, research project runs, and memory proxy endpoints are exposed in-process.
- **Memory tooling**: UTCP memory tools (`retain`, `recall`, `reflect`) remain backed by Hindsight via the active runtime wiring.
- **hindsight**: vector memory backend, running in Docker.
- **Frontend**: React dashboard is built and expects APIs to be available.

## Partial / Incomplete

- **Artifact gate**: artifact validation now performs integrity checks and records Gate3 trust-run evidence, but still does not execute golden/replay-backed validation directly from the artifact endpoint.
- **Dexter ingestion**: mock/live boundary is still mixed; real signal-generation and research wiring need clearer completion criteria.
- **Market data providers**: some provider methods remain unimplemented or placeholder-backed.

## Missing

- **Agent0 HTTP API hardening**: plan/execute service exists externally, but repo-local mocks and failure handling still need completion.
- **Reflection system** using `memory.reflect` as a scheduled loop.
- **Market data ingestion pipeline** consuming and persisting the IB bridge stream with stronger validation.
- **Real authentication flow** in place of the current placeholder login behavior.

## Where to Look Next

- **Roadmap**: `Docs/ROADMAP.md`
- **Tracked backlog**: `Docs/TODO.md`
- **IB setup**: `Docs/IB_GUIDE.md`
- **Implementation summary**: `Docs/IMPLEMENTATION_SUMMARY.md`
- **Archived reports**: `Docs/archive/`
