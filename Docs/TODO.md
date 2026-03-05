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

- [x] Add stronger artifact API coverage.
  - Completed: added focused handler tests for filtered listing states, promotion edge cases, and persistence-failure paths in [`cmd/trader/handlers_artifacts_test.go`](/c:/Projects/jax-trading%20assistant/cmd/trader/handlers_artifacts_test.go).
- [x] Improve golden diff behavior in [`tests/golden/compare.go`](/c:/Projects/jax-trading%20assistant/tests/golden/compare.go).
  - Completed: comparisons now normalize volatile fields and values (timestamps/UUIDs) and include unit coverage in [`tests/golden/compare_test.go`](/c:/Projects/jax-trading%20assistant/tests/golden/compare_test.go).

## Medium Priority

- [x] Consolidate repo verification helpers.
  - Completed: added missing workflow wrappers (`scripts/go-verify.ps1`, `scripts/golden-check.ps1`, `scripts/knowledge-cycle.ps1`) referenced by skill docs.
- [x] Fill remaining strategy-registry/mock-support test placeholders.
  - Completed: strategy registry now guards nil-pool access and includes concrete tests in [`internal/strategyregistry/registry_test.go`](/c:/Projects/jax-trading%20assistant/internal/strategyregistry/registry_test.go).

## Lower Priority

- [x] Continue stale-doc cleanup in non-operator historical docs that are still outside `Docs/archive/`.
  - Completed: moved legacy phase/program/report/evidence docs into structured archive folders and refreshed current indexes/runbooks.

## Execution Order

1. Expand integration coverage for ingestion/reflection operational flows.
2. Add release-grade smoke tests for artifact promotion and decision audit APIs.
3. Automate scheduled UAT runs with persisted run artifacts.
