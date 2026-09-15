# Jax VAL-03R3 Recovery Readiness and Execution Freeze

> **VAL-03R3A supersession note:** The static R3 readiness and recovery data
> remain accepted historical evidence, but the original R3 future execution
> freeze is superseded for execution-path purposes by
> `VAL-03R3A-RECOVERY-OOS-EXECUTION-FREEZE.json`. R3A corrects the recovery
> dataset status wording, binds a complete future runner and keeps performance
> locked until separate R4 authorization. No recovery performance was run.

## Status

`VAL-03R3 = RECOVERY READINESS + DATASET + EXECUTION FREEZE COMPLETE /
EXTERNAL REVIEW REQUIRED`.

The static contract-conformance stage passed, and the local hash-bound
pre-recovery evidence was verified without reacquisition. Outcome-free
development/validation readiness passed. Recovery market data was acquired
only for structural data-quality validation under the frozen Alpaca contract;
no strategy signals, episodes, returns or performance artifacts were produced.

The dedicated recovery runner is performance-locked. R3 may complete only by
freezing the future execution contract; VAL-03R4 remains separately gated.

The frozen execution artifact is
`Docs/validation/results/VAL-03R3-RECOVERY-OOS-EXECUTION-FREEZE.json`.
Its SHA-256 is
`4fc66036cdbc9218bbe37ca23d8dbd6366a6f062ad68090e0164ce9f70c51ee7`.
Its lifecycle is `performance_execution_started=false`,
`performance_artifacts_written=false`, and `performance_run_count=0`.

## Immutable scientific boundary

- Original 2023-01-01..2024-12-31 formal OOS: `CONTAMINATED_FOR_FORMAL_OOS`,
  formal run count `1`, never rerun.
- Recovery boundary: `2025-01-01..2026-09-11`, with 2026 explicitly partial.
- Candidate: `ma_crossover_v1`; no reselection or parameter change.
- 2025/2026 recovery data is not candidate performance evidence.
- Forward paper and Phase 13 remain not started.

## Readiness and data quality

Development readiness is `326 >= 90`; validation readiness is `104 >= 30`.
The recovery data contract and the RAW/SPLIT/SPLIT+SPIN-OFF bar-family
readiness both pass. The nine instruments have 424 synchronized sessions per
family, zero duplicates, monotonic timestamps, valid OHLC/volume, no out-of-
boundary rows, and no detected spin-off adjustment boundary. Structural
factor boundaries are retained as data-quality metadata only.

## Performance lock

`cmd/val03r-recovery-oos/` accepts only `--contract-audit` and
`--preflight-only`. It verifies the frozen parent, R2, R3, data-contract and
dataset identities, but has no performance mode or result writer. Any future
recovery OOS requires a separately authorized VAL-03R4 package and a new
execution-freeze decision.

## Safety

`ExecutionAuthority=NONE`, `CreatesFill=false`, live trading disabled, broker
execution disabled, execution disabled, and maximum leverage remains `1x`.
No broker calls, orders, fills, approvals, positions, hosted inference or
paid data access are permitted by this package.
