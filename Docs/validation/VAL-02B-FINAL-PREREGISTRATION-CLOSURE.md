# Jax VAL-02B Final Preregistration Closure

## Status and scope

`VAL-02A` was externally accepted. This bounded VAL-02B package closes the
remaining execution-economics and statistical reproducibility details for the
already-selected `ma_crossover_v1`; it does not reselect the candidate and does
not execute VAL-03.

`VAL-02B = FINAL PREREGISTRATION CLOSURE COMPLETE — EXTERNAL REVIEW REQUIRED`.

The external VAL-02B review accepted only a partial closure. The unresolved
personal-broker economics, long-only short-side boundary, pre-entry gap rule,
placebo/randomisation mechanics, exact robustness variants and provider order
are superseded by `VAL-02C-BROKER-REALISM-AND-NULL-MECHANICS-CLOSURE.md` and
manifest contract `v1.3`. This historical record is not rewritten.

The updated machine-readable manifest is
`Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json`,
contract version `v1.2`, with UTF-8 byte SHA-256
`a72e31cd8d90a700704fbc33489af20f6e0db23c4aecf76dd62917e388e20b9c`.

No historical strategy performance, return, P&L, Sharpe, hit-rate, OOS or
2025-holdout calculation was performed. No hosted inference, paid API call,
broker call, forward paper or Phase-13 action occurred.

## Frozen candidate and implementation

The selected candidate remains `ma_crossover_v1`, version `ma_crossover_v1`,
with the operational semantics already accepted in VAL-02A:

`MACrossoverStrategy.Analyze → signal type != HOLD → confidence >= 0.60 →
deterministic validity/risk checks`.

The source-only bullish pullback branch is excluded because it emits `0.55`
confidence and is rejected by the existing operational gate. The authoritative
implementation identities are those in the v1.2 machine-readable manifest,
including the accepted repository baseline and strategy, indicator, scheduler
and cost-contract blob IDs. The generic legacy backtester is not used.

## Execution unit and notional

The primary scientific unit is one non-overlapping episode with fixed analytical
notional `10,000 USD`. This is a normalized research unit, not a paper order or
broker instruction. Quantity is `10,000 / raw_entry_price`; fractional quantity
is permitted for normalization only and does not imply broker capability.

The secondary portfolio diagnostic starts at `100,000 USD` with a 1x aggregate
gross cap, equal allocation among active episodes, no resizing, and residual
cash retained. Portfolio admission cannot change the primary episode sample.

Episodes are ordered by instrument and signal-session timestamp. Only the first
eligible episode is retained while an instrument’s prior episode remains open;
signals during that interval are recorded as suppressed/no-trade observations.

## Exact execution arithmetic

For episode `i`, let `d` be `+1` for BUY and `-1` for SELL, `q` be the fixed
notional quantity, `P_e` and `P_x` be raw entry and exit references, and `D_i`
be the signed per-share dividend cash accumulated while held. The frozen net
P&L and return are:

```text
entry_notional = q * P_e
leg_cost(P) = 0.50 + q * P * (10 + 5 + 10 + 15) / 10000
short_borrow = 0 for BUY
short_borrow = entry_notional * 20 / 10000 * held_future_trading_sessions for SELL
net_pnl = d * q * (P_x - P_e) + d * q * D_i
           - leg_cost(P_e) - leg_cost(P_x) - short_borrow
net_return = net_pnl / entry_notional
```

The gross directional return is `d*q*(P_x-P_e)/entry_notional`. Costs are
charged on both legs. The primary result is net after costs; gross is retained
only as a diagnostic. The cost-policy identity is
`cost_4c9e2fbffb41d05b537eadf05b142c8f8d00ed46d672d75bca5f83a4269404f8` under
`jax.cost_slippage_policy/v1`. Long and short use the same normalized formula.

## Entry, gap and exit ordering

1. Compute indicators and the signal only from valid raw bars through the
   signal-session regular close.
2. Enter at the next regular US equity session open.
3. If that open crosses the frozen stop or target, resolve the entry and
   immediate exit at that open.
4. On later sessions, an open crossing a level resolves at the open; otherwise
   a touched level resolves at the level.
5. When stop and target are both touched in one daily bar, stop resolves first.
6. If no level resolves, exit at the close of the twentieth future regular
   session.

There are no impossible fills outside the observed bar range, no same-bar
entry, no scale-out, no partial fill model and no moving stop/target. The first
operational target only is used:

- BUY stop `signal_SMA50 - signal_ATR14`; target `signal_close + 3*signal_ATR14`.
- SELL stop `signal_SMA50 + signal_ATR14`; target `signal_close - 3*signal_ATR14`.

The second source target is ignored. The session calendar is the deterministic
Jax US regular-session calendar with UTC storage and US/Eastern interpretation.

## Short economics

SELL episodes remain in the primary directional research claim. The frozen
assumption is that the nine fixed ETFs are borrow-eligible for this analytical
study, with a conservative `20 bps per held future trading session` borrow
charge. This is not a statement about a live borrow entitlement. Dividend
liability is charged explicitly when a short is held through an ex-date. If a
required corporate-action record is missing or ambiguous, the episode abstains.
No dynamic borrow feed, broker borrow call or execution authority is added.

## Price and corporate-action semantics

- Signals, indicators, levels and execution references use `RAW_AS_OBSERVED`
  OHLCV from the approved unadjusted daily-bar source.
- No retrospective split or dividend adjustment is allowed.
- Splits are handled prospectively on the effective session; a window crossing
  a split requires a new 200-valid-session warm-up.
- Dividends are excluded from indicators and included only as point-in-time
  cash flows in returns.
- Missing/ambiguous corporate-action provenance causes abstention.

## Statistical protocol

The single primary statistic is:

`M = arithmetic mean of net_return over all eligible non-overlapping episodes`.

The primary null is `H0: M <= 0` (no positive net directional edge). The
one-sided decision threshold is `alpha = 0.05`. The primary result requires:

- positive observed `M`;
- the percentile 95% instrument-year block-bootstrap lower bound above zero;
- the one-sided within-instrument-year sign-permutation p-value below `0.05`;
- all sample, concentration, data-quality and falsification gates passing.

The sign-permutation test preserves instrument-year blocks and episode count;
directions are permuted within each block and the same frozen net-return
accounting is applied. Its p-value is:

`(1 + number of null statistics >= observed M) / (1 + 10,000)`.

No secondary diagnostic can replace the primary test. Same-interval asset
buy-and-hold, SPY buy-and-hold, gross/net, holding duration, regime and
concentration are descriptive diagnostics only.

## Block bootstrap and effective sample

The resampling unit is a non-empty `(instrument, calendar year)` block. Let `K`
be the number of observed non-empty blocks. Each bootstrap replicate samples
exactly `K` blocks with replacement, concatenates all episodes in sampled
blocks, and recomputes the episode-weighted mean `M`. Within-block temporal
dependence and unequal episode counts are retained; empty blocks are excluded.

Use exactly `10,000` replicates and a percentile 95% interval. The seed is the
first eight big-endian bytes of:

`SHA-256(manifest_sha256 || "|instrument-year-bootstrap-v1|")`.

Formal OOS requires at least 12 non-empty instrument-year blocks across at
least 6 instruments, at least 30 episodes, no instrument above 40% of episodes,
and at least 3 calendar/regime slices. Otherwise the classification is
`INSUFFICIENT_EVIDENCE`; IID inference is prohibited.

## Multiplicity and falsification

The registered selection family contains exactly one candidate, one fixed
configuration and one primary metric. No parameter, threshold, model or prompt
search is permitted. The registered falsification list is fixed before VAL-03:

- within-block direction/sign permutation;
- constrained timestamp permutation;
- matched non-signal placebo dates;
- top-five-percent contribution exclusion;
- leave-one-instrument-out;
- predefined SPY/SMA200 regime split;
- predefined theme/sector reporting;
- one-session SMA perturbation;
- 10% ATR perturbation;
- 2x cost stress;
- overlap/deduplication sensitivity;
- null, asset buy-and-hold and SPY baselines.

Failed variants and all null/placebo results must be retained. No new test may
be used to repair a failed OOS result, and no OOS rerun is permitted after
inspection.

## Partition and holdout rules

- Warm-up: `2015-01-01` through `2015-12-31`, indicator warm-up only.
- Development: `2016-01-01` through `2020-12-31`.
- Validation: `2021-01-01` through `2022-12-31`.
- Formal OOS: `2023-01-01` through `2024-12-31`, evaluated once after freeze.
- Final holdout: `2025-01-01` through `2025-12-31`, `SEALED`.

The 2025 partition was not accessed. It remains prohibited for bars, signals,
episode counts, returns, debugging, calibration and narrative confirmation.
HYP-EVENT-001A remains `NOT VALIDATED` and its contaminated 2024 lineage is
not reused.

## VAL-03 execution boundary

VAL-03 may run only after external acceptance of this v1.2 manifest. It must
use the exact implementation IDs, cost arithmetic, price semantics, episode
ordering, statistical test, partitions and falsification list above. A positive
metric alone cannot produce `FORWARD_PAPER_ELIGIBLE`.

This closure grants no recommendation mutation, paper order, broker, live,
hosted-inference or Phase-13 authority.

## Required status

```text
SELECTED CANDIDATE = ma_crossover_v1
CANDIDATE RESELECTED = NO
VAL-02A = EXTERNALLY ACCEPTED
VAL-02B = FINAL PREREGISTRATION CLOSURE COMPLETE — EXTERNAL REVIEW REQUIRED
VAL-03 = NOT STARTED
PERFORMANCE EXPERIMENT EXECUTED = NO
REAL FORWARD PAPER STARTED = NO
PHASE 13 STARTED = NO
FINAL HOLDOUT = SEALED
```
