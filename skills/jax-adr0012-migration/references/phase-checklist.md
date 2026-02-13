# ADR-0012 Phase Checklist

Primary source: `Docs/ADR-0012-two-runtime-modular-monolith.md`.

## Phase 0

- baseline golden/replay harness present
- behavior locked before boundary removal

## Phase 1-2

- collapse orchestration HTTP seams incrementally
- preserve API contract shapes

## Phase 3

- introduce `cmd/trader` and reusable composition root
- keep deterministic execution path

## Phase 4

- add artifact tables and lifecycle checks
- enforce approved-only loading in Trader

## Phase 5

- remove obsolete internal service process wiring
- retain only justified external boundaries

## Cross-Phase Safety Checks

- no Trader import of research-only integrations
- golden/replay non-regression unless intentionally updated
- explicit rollback condition noted for each phase ticket
