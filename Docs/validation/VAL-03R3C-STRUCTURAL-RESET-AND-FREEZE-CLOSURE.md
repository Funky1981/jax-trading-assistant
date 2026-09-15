# VAL-03R3C Structural Reset & Final Freeze Completeness Closure

## Status

`VAL-03R3C = COMPLETE / EXTERNAL REVIEW REQUIRED`. This is an outcome-free
continuation of R3B. `VAL-03R3B` is superseded for the corrected structural
reset and complete execution-freeze identity. `VAL-03R4` remains unauthorized.

## Scope and preserved boundary

The selected candidate remains `ma_crossover_v1`. The original 2023-2024 VAL-03
run remains `CONTAMINATED_FOR_FORMAL_OOS`, has formal run count `1`, and is never
rerun. Recovery remains bounded to `2025-01-01..2026-09-11`; no 2025/2026
performance, signals, episodes, returns or outcomes were calculated here.

All strategy parameters, the `0.60` threshold, costs, placebo/bootstrap rules,
sample floors, falsification policy, universe, safety settings and recovery
boundary remain unchanged.

## Structural reset correction

`structuralStateAt` now scans the complete session prefix through the requested
date. It does not return early when the first 200 valid sessions are reached.
Every later structural boundary therefore resets the state even when the
instrument had previously initialized. The boundary session is post-boundary
valid session `#1`; invalid synchronized sessions do not advance the count.
The frozen rule remains `200 valid synchronized sessions` before eligibility.

Focused tests cover late boundaries after prior initialization, multiple
boundaries, invalid sessions, canonical equivalence with the VAL-03C helper,
and validated boundary reporting.

## One-shot execution integrity

The persisted `STARTED_ONCE` marker now records `performance_run_count=1`, so a
crash after marker creation consumes the sole attempt exactly like
`COMPLETED_ONCE`. State validation is strict about the candidate, boundary,
freeze identity, status and run count. An existing state must match the current
R3C freeze. Exclusive file creation and a concurrent-start test prove that only
one starter can win.

`--execute` remains fail-closed before data loading unless a future exact R4
authorization exists. No authorization, run state or result artifact exists in
this package.

## R3C freeze

`Docs/validation/results/VAL-03R3C-RECOVERY-OOS-EXECUTION-FREEZE.json` is the
machine-readable freeze. It binds the parent/R2/R3/R3A/R3B/R3C identities,
candidate contract, recovery boundary, structural reset, costs, placebo and
bootstrap contracts, sample gates, falsification list, safety locks, lifecycle,
authorization path and every current runner source blob. Its status is
`FROZEN_FOR_VAL03R4_EXTERNAL_AUTHORIZATION` and its lifecycle remains
`performance_execution_started=false`, `performance_artifacts_written=false`,
`performance_run_count=0`.

The freeze is a contract, not authorization. The attempted `--execute` check
fails with `R4 authorization required; performance locked before data load`.

## Verification boundary

The focused recovery-runner, preflight, and MA command tests pass. The freeze
and runner source identities are checked before contract-audit and preflight
operation. No market-data payload was acquired or read by R3C, and the 2025
holdout remains sealed.

Safety remains `ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`,
`EXECUTION_ENABLED=false`, `ExecutionAuthority=NONE`, `CreatesFill=false`, and
maximum leverage `1x`.
