# Repository Todo

This list is based on the live codebase on branch `work` as of 2026-03-05. It tracks remaining work for the active `cmd/trader` + `cmd/research` runtime layout.

## Recently Completed

- [x] Artifact validation requires real Gate2 + Gate3 evidence before `DRAFT -> VALIDATED`.
- [x] Trust-gate evidence and artifact approval records are linked for auditability.
- [x] Replace placeholder authentication in [`libs/auth/handlers.go`](/c:/Projects/jax-trading%20assistant/libs/auth/handlers.go).
  - Login now uses persisted `auth_users` credentials (bcrypt + lockout policy).
- [x] Finish market-data provider gaps in [`libs/marketdata/ib/provider.go`](/c:/Projects/jax-trading%20assistant/libs/marketdata/ib/provider.go) and [`libs/marketdata/provider_polygon.go`](/c:/Projects/jax-trading%20assistant/libs/marketdata/provider_polygon.go).
- [x] Reduce N+1 approval lookups in artifact listing APIs.
- [x] Stabilize Agent0 and Dexter mock adapters as deterministic test shims.
- [x] Remove docs/runtime contract drift in core operator docs (`QUICKSTART`, `STATUS`, `ROADMAP`, `PROJECT_OVERVIEW`).

## High Priority

- [ ] Add stronger artifact API coverage.
  - Missing focused tests for filtered state listing, promotion edge cases, and failed persistence paths.
- [ ] Improve golden diff behavior in [`tests/golden/compare.go`](/c:/Projects/jax-trading%20assistant/tests/golden/compare.go).
  - Current TODO: ignore expected volatile fields (timestamps/UUIDs) more intelligently.

## Medium Priority

- [ ] Consolidate repo verification helpers.
  - Skill docs mention scripts/helpers that are no longer present at expected repo locations.
- [ ] Fill remaining strategy-registry/mock-support test placeholders.

## Lower Priority

- [ ] Continue stale-doc cleanup in non-operator historical docs that are still outside `Docs/archive/`.

## Execution Order

1. Complete artifact API test hardening.
2. Improve golden comparison resilience for deterministic replay workflows.
3. Finish remaining verification-helper and placeholder test cleanup.
