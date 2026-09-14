# Jax VAL-03R1C Secondary-Diagnostic & Audit-Traceability Closure

## Result

`VAL-03R1C = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This package closes the bounded defects identified after VAL-03R1B. It does not
repair or rerun the contaminated VAL-03 formal OOS. The v1.5 preregistration,
candidate identity, strategy parameters, and contaminated result artifact remain
unchanged.

## Closed findings

| Finding | Closure evidence | Status |
|---|---|---|
| Evaluator did not populate `SecondaryDiagnostics` | `evaluatePartition` now runs `simulateDirectional` for actionable BUY and SELL signals with a separate active interval and abstention accounting | PASS |
| Production sign path was empty | `run` → `evaluatePartition` → `buildFalsification` passes the typed production collection; integration test proves both directions reach it | PASS |
| Manual injection was the only test | `TestSecondaryDiagnosticsArePopulatedByProductionEvaluator` exercises the evaluator wiring on deterministic synthetic data | PASS |
| Single-direction strata entered mixed permutation | `MIXED_INSTRUMENT_YEAR_ONLY_V1` uses only mixed strata and records excluded IDs/counts | PASS |
| Regime cells were counted but not persisted | `CalendarRegimeCells` is persisted in partition output and populated by `finishPartition` | PASS |
| Unknown non-blocking statuses could pass | `classifyPromotion` recognizes the closed status set and fails any unknown status closed | PASS |

## Preserved boundary

`VAL-03 = CONTAMINATED_FOR_FORMAL_OOS`; formal run count remains 1 with
`no_rerun=true`. `NEW INDEPENDENT OOS = NOT AUTHORIZED`. No development,
validation, OOS, 2025, or 2026 outcomes were accessed; no broker, paper, hosted
AI, paid-provider, order, fill, approval, or Phase-13 activity occurred.

## Verification

Focused evaluator, sign-permutation, breadth-persistence, and closed-status
tests pass. The outcome-free `--contract-audit` remains required to verify
manifest v1.5, holdout protection, contaminated-run protection, execution
authority, R1B identities, and R1C production-path identities.

## Governance

- `VAL-03R1 = PARTIAL CLOSURE`.
- `VAL-03R1A = SUPERSEDED`.
- `VAL-03R1B = SUPERSEDED BY R1C`.
- `VAL-03R1C = COMPLETE / EXTERNAL REVIEW REQUIRED`.
- `2025 HOLDOUT = SEALED`.
- `FORWARD PAPER = NOT STARTED`.
- `PHASE 13 = NOT STARTED`.
