# VAL-03R3A Executable Recovery Runner and Final Freeze Closure

## Status

`VAL-03R3A = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This package repairs the executable recovery-runner contract after the R3
static freeze was found not to be a complete future execution path. The
corrected recovery dataset artifact explicitly records that 2025 data is
present for the recovery boundary and that no strategy output was generated.

The selected candidate remains `ma_crossover_v1`. The original 2023-2024
formal run remains `CONTAMINATED_FOR_FORMAL_OOS` with run count `1` and is never
rerun. Recovery performance remains unstarted.

## Bound contracts

- Parent v1.5 manifest and R2/R3 identities are hash-checked before any mode.
- The corrected R3A recovery readiness artifact is required and hash-checked.
- The recovery boundary is `2025-01-01..2026-09-11`; no extension is allowed.
- Threshold `0.60`, long-only primary population, cost IDs, placebo IDs,
  bootstrap, floors, concentration ceiling, and falsification identities are
  loaded from the frozen parent/R2/R3 contracts and checked exactly.
- `--contract-audit` and `--preflight-only` perform no strategy evaluation.
- `--execute` requires the future R4 authorization artifact and matching R3A
  freeze SHA before loading data. No authorization artifact was created here.
- A completed R4 run-state or any completed R4 result artifact blocks rerun.

## Runner and behavioural conformance

The new `cmd/val03r-recovery-oos` package contains the future recovery data
loader, frozen evaluation, placebo, bootstrap, falsification and classification
pipeline, while the CLI remains fail-closed until R4 authorization. The old
contaminated `cmd/val03-ma-crossover` runner was not re-enabled.

Deterministic tests compare representative readiness-harness outputs directly
with `strategies.NewMACrossoverStrategy().Analyze`: bullish alignment receives
the source confidence boosts, the pullback remains `0.55` and below the `0.60`
gate, HOLD is non-actionable, and SELL is retained only as a secondary
diagnostic. Tests also enforce strict authorization typing, one-shot run-state
handling, date and cost guards, placebo determinism, bootstrap determinism,
and the inherited falsification contract.

## Future execution freeze

`Docs/validation/results/VAL-03R3A-RECOVERY-OOS-EXECUTION-FREEZE.json` is the
superseding freeze. Its status is
`FROZEN_FOR_VAL03R4_EXTERNAL_AUTHORIZATION`; it binds the final runner source
blobs, all inherited scientific identities, the corrected dataset identity,
future command, authorization schema/path, one-shot lifecycle, and safety
state. Its current SHA-256 is
`7338909b98d0165e351079c4b82c42bf99f7464817586b1e37ba8b51923ceec9`. It does
not authorize execution.

The authorization file
`Docs/validation/results/VAL-03R4-EXECUTION-AUTHORIZATION.json` and run-state
file are intentionally absent. No R4 result artifact exists.

## Safety and outcome boundary

`ExecutionAuthority=NONE`, `CreatesFill=false`,
`ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`,
`EXECUTION_ENABLED=false`, and maximum leverage `1x` remain unchanged. No
broker, order, fill, approval, position, forward-paper, or Phase-13 activity
occurred. No 2023-2024 rerun, recovery performance calculation, or 2025/2026
candidate outcome inspection occurred.

The next package is `VAL-03R4 — SINGLE REPLACEMENT FORMAL OOS EXECUTION`,
which requires a separate external authorization.
