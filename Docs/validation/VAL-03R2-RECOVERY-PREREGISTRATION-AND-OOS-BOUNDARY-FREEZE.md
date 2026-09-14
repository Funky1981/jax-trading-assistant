# Jax VAL-03R2 Recovery Preregistration & Independent OOS Boundary Freeze

## Status

`VAL-03R2 = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This is an outcome-free preregistration and boundary package. It does not
acquire recovery data, load 2025 or 2026 bars, calculate signals or outcomes,
rerun the contaminated VAL-03 experiment, start forward paper, or begin Phase
13.

## Historical boundaries preserved

- `VAL-03 = CONTAMINATED_FOR_FORMAL_OOS`.
- The contaminated artifact remains immutable: formal run count `1`,
  `no_rerun=true`.
- `VAL-03R1D = COMPLETE / GO`; its harness corrective loop is closed for the
  identified defects.
- Parent manifest v1.5 remains unchanged with SHA-256
  `96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`.

## Recovery identity

The new machine contract is:

`Docs/validation/manifests/VAL-03R2-ma_crossover_v1-RECOVERY-PREREGISTRATION.json`

with contract identity `jax.val-03r2.ma-crossover-recovery-preregistration/v1`.
It references, rather than modifies, the v1.5 parent and preserves
`ma_crossover_v1`, its strategy parameters, bullish long-only primary claim,
0.60 actionable threshold, 20/50/200 SMA periods, ATR14, AvgVolume20, frozen
stop/target/holding rules, USD 10,000 whole-share sizing, base IBKR cost model,
legacy stress model, and `ExecutionAuthority=NONE`.

## Inherited R1 policies

The recovery contract inherits these prospective policy identities:

- `CEIL_5_PERCENT_V1`
- `CALENDAR_YEAR_X_SPY_SMA200_REGIME_V1`
- `SECONDARY_DIRECTIONAL_PRICE_EFFECT_V1`
- `SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1`
- `MIXED_INSTRUMENT_YEAR_ONLY_V1`
- `FAIL_CLOSED_UNKNOWN_STATUS_V1`
- `SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1`

Historical v1.5 is not back-edited.

## Independent recovery boundary

The new boundary is exactly:

- Start: `2025-01-01`
- End: `2026-09-11`
- Valid US regular sessions only
- `2026-09-14` excluded because it was incomplete at freeze time
- All dates after `2026-09-11` excluded
- Extension after results: prohibited; a new experiment identity is required

This boundary was selected structurally, before outcome access. Nine instruments
can provide at most nine instrument-year blocks in 2025 and at most eighteen
across 2025 and 2026. The registered effective-block floor is twelve. No actual
signal, episode, regime, or block count was inspected.

## Warm-up and readiness boundary

Previously accepted pre-recovery evidence may provide only the minimum valid
indicator lookback required to initialize the frozen strategy. Those observations
are `WARMUP_ONLY`; no pre-recovery signal, episode, return, or performance value
contributes to recovery OOS. The contaminated 2023–2024 performance result is
never reused.

Before a later package opens the recovery holdout, it must verify eligibility
floors without reporting returns or other performance statistics:

- Development floor: `90`
- Validation floor: `30`
- Recovery OOS floor: `30`

Failure is `RECOVERY READINESS = INSUFFICIENT_EVIDENCE` and stops before 2025
access. R2 freezes this rule but does not execute the readiness calculation.

## Validation-floor correction

The prior typed `FrozenExperimentConfig` bound development and formal OOS floors
but omitted the manifest's validation floor. R2 adds `ValidationSampleFloor`,
loaded directly from:

`sample_and_dependence.episode_floors.validation`

The required typed values are development `90`, validation `30`, and OOS `30`.
This is a contract-binding correction, not a performance or strategy change.

## Provider and acquisition contract

Any later acquisition is fixed to:

- Provider: Alpaca
- Feed: SIP
- Timeframe: `1Day`
- Universe: `SPY`, `QQQ`, `IWM`, `DIA`, `XLK`, `XLF`, `XLE`, `TLT`, `GLD`
- Adjustment families: RAW, SPLIT, SPLIT+SPIN-OFF
- Route: existing hardened Alpaca historical route
- Fallback: NONE
- New paid spend: USD 0
- Raw bytes persisted before normalization with request/source identity,
  retrieval timestamp, adjustment identity, and content hash

No provider call is authorized by R2.

## Promotion gates

The recovery OOS gates remain frozen: at least 30 primary episodes, 30 matched
pairs, 12 non-empty instrument-year blocks, six instruments, no instrument over
40%, at least three calendar-year × SPY-SMA200-regime cells, positive base-cost
mean, positive block-bootstrap lower bounds for actual and paired differences,
passing top-five concentration, blocking falsifications, and data quality.
Unknown material status fails closed. Secondary sign permutation is informational
and non-blocking.

## No post-result salvage

After the first recovery result, no tuning, threshold change, universe change,
date extension, instrument removal/addition, cost change, placebo change,
bootstrap change, falsification change, regime change, or top-five policy change
is allowed. A material change requires a new preregistration identity.

## Outcome-free audit

The dedicated command is:

`go run ./cmd/val03-ma-crossover --recovery-contract-audit`

It reads only the recovery contract, the hash-bound parent manifest, source
identities, static safety/boundary fields, and the immutable contaminated-run
guard. It refuses local recovery data and emits no signal, episode, block,
regime, return, or outcome counts.

The existing runner remains fail-closed against the contaminated VAL-03
identity; R2 does not create a path that silently executes the old experiment.

## Safety and status

- `ALLOW_LIVE_TRADING=false`
- `BROKER_EXECUTION_ALLOWED=false`
- `EXECUTION_ENABLED=false`
- `ExecutionAuthority=NONE`
- `CreatesFill=false`
- Maximum leverage: `1x`
- New independent OOS execution: `NOT AUTHORIZED`
- Recovery data acquisition: `NOT AUTHORIZED`
- 2025 and 2026: `SEALED / NOT ACCESSED`
- Forward paper: `NOT STARTED`
- Phase 13: `NOT STARTED`

## Exact R2 status

`VAL-03R2 = COMPLETE / EXTERNAL REVIEW REQUIRED`.
