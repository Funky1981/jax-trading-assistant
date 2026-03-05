# Repository Todo

This list is based on the live codebase on branch `work` as of 2026-03-03. It tracks real remaining work, not the older `services/jax-*` layout that no longer reflects the active runtime.

## In Progress

- [x] Artifact validation uses real Gate3 execution evidence.
  - Completed: `/api/v1/artifacts/{id}/validate` now attaches the persisted Gate3 trust-run result, stores the gate report URI on the approval record, and blocks promotion if Gate3 fails.
- [x] Artifact validation now executes Gate2 deterministic replay/golden evidence plus Gate3 promotion evidence.
  - Completed: `/api/v1/artifacts/{id}/validate` now requires both Gate2 (`deterministic_replay`) and Gate3 (`artifact_promotion`) trust gates to pass before `DRAFT -> VALIDATED`.

## High Priority

- [x] Keep trust-gate evidence and artifact approval evidence linked.
  - Completed: artifact validation now records gate run evidence (`replayRun`, `gateRun`) and persists best available run/report references on the approval row.
- [ ] Remove contract drift between docs and runtime layout.
  - `Docs/STATUS.md` and some older memories still describe missing `services/jax-api` and `services/jax-orchestrator`, while live endpoints are now in `cmd/trader`, `cmd/research`, and `cmd/trader/frontend_api.go`.
- [x] Replace placeholder authentication in [`libs/auth/handlers.go`](/c:/Projects/jax-trading%20assistant/libs/auth/handlers.go).
  - Completed: login now uses persisted `auth_users` credentials (bcrypt + lockout policy) instead of accepting any non-empty username/password.
- [x] Finish market-data provider gaps in [`libs/marketdata/ib/provider.go`](/c:/Projects/jax-trading%20assistant/libs/marketdata/ib/provider.go) and [`libs/marketdata/provider_polygon.go`](/c:/Projects/jax-trading%20assistant/libs/marketdata/provider_polygon.go).
  - Completed: IB now supports polling quote streams and derived trade history from 1-minute bars; Polygon now supports trades, earnings (financials-backed), and tier-aware polling quote streams.

## Medium Priority

- [ ] Reduce N+1 approval lookups in the artifact API.
  - `GET /api/v1/artifacts` still fetches approvals row-by-row after listing artifacts.
- [ ] Add stronger artifact API coverage.
  - Missing focused tests for filtered state listing, promotion edge cases, and failed persistence paths.
- [ ] Improve golden diff behavior in [`tests/golden/compare.go`](/c:/Projects/jax-trading%20assistant/tests/golden/compare.go).
  - Current TODO: ignore expected volatile fields like timestamps and UUIDs more intelligently.
- [x] Stabilize Agent0 and Dexter mock adapters as deterministic test shims.
  - Completed: [`libs/agent0/mock.go`](/c:/Projects/jax-trading%20assistant/libs/agent0/mock.go) and [`libs/dexter/mock.go`](/c:/Projects/jax-trading%20assistant/libs/dexter/mock.go) now return deterministic non-error defaults with focused unit tests.

## Lower Priority

- [ ] Clean stale docs that still describe superseded service topology.
- [ ] Consolidate repo verification helpers.
  - Skill docs mention scripts that are not present in the repo root anymore.
- [ ] Fill in remaining strategy-registry/mock-support test placeholders.

## Execution Order

1. Remove API/store inefficiencies and add tests around artifact workflows.
2. Clean runtime/docs drift so operators and future work use the correct entrypoints.
3. Tackle remaining subsystem gaps: non-critical provider enhancements and operational hardening.
