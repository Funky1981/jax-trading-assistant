# VAL-03R4A1 — Full Execution-Path Preflight Closure

## Status

`VAL-03R4A1 = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This package corrects the active preflight dispatch and creates a superseding
execution freeze. It does not authorize R4B or run recovery performance.

## Preserved R4A evidence

The R4A attempt, forensic finding, data-integrity preflight and R4A freeze are
historical and unchanged:

- attempt integrity: `8dc3874be5d4593c82fc77c3b1090a6fa55b4787b72e0a09a23bbd8066e05d96`
- forensic: `70941d082cd1d430b684017129341370b25e7fe845730169dfd5a3a847d85624`
- data-integrity preflight: `2d5f7fc479b6777854987fe1954114a74a528015d0952ed086666aad635b777f`
- R4A freeze: `f359db05cd3a4bfe2a96b0dc2a3b993a710410695d15329db60cf8de8f8d0eef`

The old R4 authorization remains historical and not reusable. The formal
recovery run state and R4 result artifacts remain absent.

## Defect and correction

Before this package, `--preflight-only` validated payload hashes but did not
reconstruct and validate the combined data used by future execution. The new
`r3aRunDataIntegrityPreflight` function performs payload hash validation and
the complete combined structural load. Both `--preflight-only` and the future
execution path call this function before any `STARTED_ONCE` transition.

The shared path validates exact payload families, synchronization, date bounds,
bar validity, canonical RAW/SPLIT factors through
`marketdata.DeriveVAL03BSplitFactor`, canonical boundaries through
`marketdata.IsVAL03BStructuralFactorBoundary`, and SPLIT versus
SPLIT+SPIN-OFF equality. It does not calculate indicators, signals, episodes,
returns, P&L, placebos, bootstrap statistics or falsification outcomes.

## Structural preflight result

The official local command:

`go run ./cmd/val03r-recovery-oos --preflight-only`

returned:

`VAL03R4A1_PREFLIGHT=PASS instruments=9 sessions_per_instrument=2688 canonical_factor_failures=0 spin_off_differences=0 out_of_range_rows=0 structural_boundaries=2 performance_execution_started=false performance_run_count=0`

The known structural boundaries remain XLK `2025-12-05` and XLE
`2025-12-05`. The accepted XLK `2016-02-11` rounded frame passes the canonical
ratio-tolerance contract. No prices or strategy outcomes were printed.

## Regression and fail-closed behavior

Deterministic dispatch coverage proves that `--preflight-only` invokes the
shared structural loader and never invokes evaluation or run-state creation.
The XLK regression remains covered: canonical validation passes while the
legacy absolute-price validator fails.

`go run ./cmd/val03r-recovery-oos --execute` remains fail-closed with
`VAL-03R4B AUTHORIZATION REQUIRED`. No R4B authorization was created, no
`STARTED_ONCE` marker was written, and no result artifact was created.

## R4A1 freeze

`VAL-03R4A1-RECOVERY-OOS-EXECUTION-FREEZE.json` is contract
`jax.val-03r4a1.recovery-oos-execution-freeze/v1`, status
`FROZEN_FOR_VAL03R4B_EXTERNAL_AUTHORIZATION`, and binds the current runner,
shared-loader and canonical helper identities. Its SHA-256 is:

`7f1ca6532c5ab6e6fab30aed3a53179eb236ef4ed1b294a7aed40c0275f5e830`

The freeze preserves the candidate, 0.60 threshold, strategy parameters, cost
models, placebos, bootstrap, sample gates, falsification suite, recovery
boundary and safety controls. Future R4B authorization must bind this R4A1
freeze SHA, not the historical R4A freeze SHA.

## Safety and outcome state

`ExecutionAuthority=NONE`, `CreatesFill=false`, live trading, broker execution
and execution are disabled, and maximum leverage remains `1x`. Formal recovery
run count is `0`; `STARTED_ONCE` is absent; outcomes are not exposed; the
2023–2024 contaminated run is not rerun; forward paper and Phase 13 remain not
started.
