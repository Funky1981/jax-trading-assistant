# Jax VAL-02A Preregistration Contract Closure

## Status and scope

`VAL-02 CANDIDATE SELECTION = ACCEPTED` and the selected candidate remains
`ma_crossover_v1`. This document closes the preregistration defects identified
before VAL-03. It is a contract-only artifact: no historical performance,
return, P&L, Sharpe, hit-rate, OOS or holdout calculation was performed.

`VAL-02A = PREREGISTRATION CONTRACT CLOSURE COMPLETE — EXTERNAL REVIEW
REQUIRED`.

`VAL-03 = NOT STARTED / BLOCKED BY VAL-02A`.

The machine-readable manifest is
`Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json`.
Its UTF-8 byte SHA-256 at this closure is
`ba603f41ef815ee064d1e2a6d3d7023e05f464da584283b80170e55acbd349b4`.

## Code identity

The implementation identity is bound to the accepted repository baseline
`757878ada2177057fb38b74f31d94ff758f44be0` and to immutable Git blob IDs:

| Role | Path | Blob SHA-1 |
| --- | --- | --- |
| Strategy source | `libs/strategies/ma_crossover.go` | `fb22b9dd909a0ae5f13dd727ff2eddf40963d426` |
| Strategy/type contract | `libs/strategies/strategy.go` | `8ff03f23aed6dba929cdc83cda5c1a1153da616f` |
| Historical operational indicators and input construction | `internal/trader/signalgenerator/inprocess.go` | `4a3f0537013308f40ee94c3eb723d3a152297048` |
| Actionable signal gate and persisted first target | `cmd/trader/instance_scheduler.go` | `54244f758599e97317d31073f8c23f1d2b4ae729` |
| Cost contract implementation | `internal/modules/evaluation/costs.go` | `2cc7d92d4ab045f1295fbb0570e04ac2301fc9f5` |
| ATR algorithm contract reference | `internal/modules/quant/volatility.go` | `6fb7cee0e1643352f073022bbf999283b5a04a49` |

The generic legacy backtester in `libs/strategies/backtest.go` is explicitly
not the VAL-03 implementation: it enters on the signal bar, uses all target
levels and has no research cost binding. The `libs/dataset` indicator helper is
also not the authoritative operational input path. VAL-03 must implement the
contract below from the pinned identities rather than silently using either
legacy path.

## Candidate execution semantics

The scientific unit is the behavior Jax could actually use:

`MACrossoverStrategy.Analyze` on the historical equivalent of the operational
`AnalysisInput`, followed by the existing actionable gate:

1. signal type is not `HOLD`;
2. confidence is at least `0.60`; and
3. the normal deterministic research/risk validity checks pass.

The signal is timestamped at the regular-session close whose data formed the
input. Entry is the next regular US equity session open. This is the frozen
contract, not a new research gate.

The source’s bullish pullback branch returns confidence `0.55` (`0.65 - 0.10`)
and is therefore excluded by the existing operational `0.60` gate. Pullback
signals are not to be reintroduced after outcomes are observed. HOLD and
below-threshold signals are recorded as no-trade/abstention observations, not
silently converted into episodes.

## Confidence gate

`ACTIONABLE CONFIDENCE THRESHOLD = 0.60`.

The source confidence calculation is frozen as heuristic, not probability:

- base confidence: `0.65`;
- add `0.12` when `MarketTrend` agrees with direction;
- add `0.08` when current volume is greater than `AvgVolume20`;
- add `0.10` when directional `(SMA20 - SMA200) / SMA200` separation exceeds
  `0.05` in magnitude;
- cap at `1.00`.

For bearish signals the separation is `(SMA200 - SMA20) / SMA200`. No
additional confidence filter, calibration, threshold tuning or post-result
override is allowed. Probability calibration is `NOT APPLICABLE`.

`MarketTrend` is frozen to the operational alignment calculation: bullish when
`SMA20 > SMA50 > SMA200`, bearish when `SMA20 < SMA50 < SMA200`, otherwise
neutral. `AvgVolume20` is the arithmetic mean of the last 20 eligible volumes.

## Indicator contract

All windows include only bars available at the signal close and require enough
valid history. No provider-supplied indicator or alternate technical-analysis
library may be substituted.

- `SMA20 = arithmetic mean of the last 20 eligible closes`.
- `SMA50 = arithmetic mean of the last 50 eligible closes`.
- `SMA200 = arithmetic mean of the last 200 eligible closes`.
- `ATR14 = arithmetic mean of the last 14 true ranges`, where each true range
  is `max(high-low, abs(high-previous_close), abs(low-previous_close))`; the
  first bar without a previous close is not a true-range observation.
- `AvgVolume20 = arithmetic mean of the last 20 eligible volumes`.
- `MarketTrend` is the three-SMA alignment stated above.

Indicator algorithm identity is `jax.ma_crossover.operational-indicators/v1`
implemented by the pinned `inprocess.go` blob above. The standalone quant ATR
contract is a conformance reference only; VAL-03 must use the same simple
average true-range semantics.

## Price and corporate-action contract

The historical series must not use a retrospectively fully adjusted series
whose future split or dividend information changes an earlier decision.

- `SIGNAL PRICE MODE = RAW_AS_OBSERVED`.
- `ENTRY/STOP/TARGET PRICE MODE = RAW_PRICE_SCALE`.
- `RETURN PRICE MODE = RAW_EXECUTION_PNL_PLUS_EXPLICIT_CORPORATE_ACTION_CASH`.
- `CORPORATE ACTION MODE = POINT_IN_TIME_EFFECTIVE_EVENT_HANDLING`.
- `DIVIDEND TREATMENT = no dividend in indicators; a long receives and a short
  pays the cash distribution when held through the ex-date, using the frozen
  point-in-time corporate-action record`.
- `SPLIT TREATMENT = no retrospective back-adjustment; apply the split
  prospectively on its effective session to quantity/price bookkeeping, and
  abstain until a fresh 200-valid-session SMA200 warm-up if an indicator window
  crosses a split boundary`.

The dataset must preserve raw OHLCV and corporate-action provenance. A missing,
ambiguous, revised or unavailable corporate-action record causes a recorded
abstention rather than an after-the-fact adjustment choice. The same raw price
scale and point-in-time rules apply to indicators, signal values, entry,
stop/target evaluation and returns.

## Entry contract

The earliest permitted decision is the close of the signal session. The entry
is a market-style fill at the next regular US equity session open, strictly
after that close. The signal bar cannot be used as an entry bar. If the next
open crosses a frozen stop or target, the entry and immediate exit are both
resolved at the next-open reference without an impossible fill outside the
bar. Entry and exit adverse spread/slippage are charged by the frozen cost
contract.

The session calendar is the repository’s deterministic US regular-session
calendar with UTC storage and US/Eastern session interpretation. Holidays, DST,
missing sessions and timezone conversion are provenance-bearing inputs; no
calendar inferred from future prices is permitted.

## Stop, target and exit contract

Only the crossover/alignment branches can become actionable under the frozen
operational gate. For a bullish crossover:

- stop: `signal_SMA50 - signal_ATR14`;
- target: `signal_close + 3 * signal_ATR14`;
- the source’s second `+5 * ATR` target is ignored because the operational
  scheduler persists only `TakeProfit[0]`;
- no scale-out and no partial-profit allocation.

For a bearish crossover:

- stop: `signal_SMA50 + signal_ATR14`;
- target: `signal_close - 3 * signal_ATR14`;
- the source’s second `-5 * ATR` target is ignored;
- no scale-out and no partial-profit allocation.

Levels are calculated once from the signal-time input and do not move with
later bars. The maximum holding period is 20 future regular sessions. For each
future bar, an open crossing a level fills at the open; otherwise a touched
level fills at that level. If both stop and target are touched in one daily bar,
stop is resolved first. If neither exits the episode, the twentieth session
close is the time exit. There is no re-entry while an instrument episode is
active, and no signal-generated level may create an order or broker action.

## Bearish-direction contract

Bearish episodes are included in the primary tradable-return claim as short
direction measurements, not execution authorization. The signed direction is
`+1` for BUY and `-1` for SELL. Net return is computed symmetrically from the
same entry/exit price convention:

`directional_gross = direction_sign * (exit_value - entry_value) / entry_value`.

Short economics are frozen: spread, slippage, commission and market impact are
charged on both legs; borrow is charged at `20 bps per held trading day` on the
short notional; and any dividend liability is included as an explicit cash
outflow. If borrow availability or a required corporate-action record is not
known, the episode abstains. A bearish episode is never removed after results
are seen.

## Cost model

`COST MODEL ID = cost_4c9e2fbffb41d05b537eadf05b142c8f8d00ed46d672d75bca5f83a4269404f8`.

This is the existing `jax.cost_slippage_policy/v1` evaluation contract with
the exact pre-existing Phase-07 deterministic fixture assumptions, frozen for
VAL-03 before outcomes:

| Field | Frozen value |
| --- | ---: |
| Commission per order | `0.50 USD` |
| Commission | `10 bps` per leg notional |
| Spread | `5 bps` per leg notional |
| Slippage | `10 bps` per leg notional |
| Market impact | `15 bps` per leg notional |
| Short borrow | `20 bps per held trading day` |
| Reference-price rule | `NEXT_AVAILABLE_OBSERVATION` |
| Contract | `jax.cost_slippage_policy/v1`, policy version `v1` |
| Assumption source | existing `Phase 07 deterministic fixture` |
| Policy creation timestamp | `2026-09-07T16:00:00Z` |

Commission, spread, slippage and impact are applied separately at entry and
exit using each leg’s reference price and unit-notional quantity. Borrow is
applied only to SELL/short episodes for the number of held future trading
sessions; long episodes receive zero borrow despite the generic cost function’s
holding-day parameter. Dividends are separate point-in-time cash flows. Gross
and net values must both be retained, but primary decisions use net values.
This policy is not `diagnostic-cost-model-v1`, which is a paper-venue
diagnostic model and does not contain the required research borrow/impact
semantics.

## Primary metric and comparator

The earlier benchmark-relative signed-return wording is superseded. It is
directionally asymmetric for bearish signals and is not the primary test.

`PRIMARY SCIENTIFIC UNIT = one non-overlapping episode with fixed unit notional`.

`PRIMARY METRIC = arithmetic mean of net directional return per eligible episode`.

For episode `i`, net return is the directional gross return minus round-trip
costs and signed corporate-action cash flows, divided by the fixed entry
notional. BUY and SELL use the same formula and the same cost treatment.

`PRIMARY NULL = no positive net directional edge (mean <= 0)`.

The primary one-sided decision test is a 95% instrument-year block-bootstrap
lower confidence bound for the mean, supplemented by the preregistered
within-instrument-year sign-permutation randomization test. Success requires a
positive net mean, the lower confidence bound above zero, and the observed mean
to exceed the 95th percentile of the frozen sign-permutation null distribution.
This is a scientific rejection/acceptance rule, not a guarantee of promotion.

Same-interval per-asset buy-and-hold, SPY buy-and-hold, and zero/null returns
remain secondary diagnostics only. They cannot replace or redefine the primary
metric after results are visible.

## Episode-level and portfolio-level analysis

The primary analysis is episode-level and is independent of concurrent
portfolio allocation. Each eligible non-overlapping episode contributes one
fixed unit-notional return; dependence is handled by the block contract below.

A portfolio-level simulation is secondary only. It uses `100,000 USD` starting
capital, a 1x aggregate gross cap, equal allocation among currently active
episodes, no resizing of existing positions, residual cash retained, and no
new portfolio episode when its full unit allocation would breach the cap. Long
and short notionals both count toward gross exposure. Portfolio mechanics must
never alter the primary episode observations.

## Bootstrap and effective sample contract

The resampling unit is the non-empty `(instrument, calendar year)` block. All
episodes in a sampled block are retained together, preserving within-block
dependence and unequal episode counts. Empty blocks are excluded. Use exactly
`10,000` replicates, a percentile interval, and a deterministic seed derived as
the first eight bytes of `SHA-256(manifest_sha256 || "|instrument-year-bootstrap-v1")`.
The seed is recorded with results, not chosen from outcomes.

The minimum effective sample is `12` non-empty instrument-year blocks across at
least `6` instruments in the formal OOS partition, in addition to the existing
episode floors. If this floor is not met, the result is `INSUFFICIENT_EVIDENCE`
and inference must not fall back to IID observations.

## Holdout seal

The final holdout remains `2025-01-01` through `2025-12-31`, with dataset
identity and seal metadata only permitted. No bars, signals, episode counts,
returns, debugging, calibration, narrative confirmation or other semantic
research use is allowed. No VAL-02A action accessed the 2025 partition.

## VAL-03 decision boundary

VAL-03 may execute only after external review of this closure. It must use the
exact candidate, code identities, operational gate, price/corporate-action
rules, partitions, cost policy, metric, bootstrap and falsification suite in
the machine-readable manifest. It must evaluate historical performance only
then, once, without modifying recommendation logic or creating paper/live
orders.

## Control statements

```text
SELECTED CANDIDATE = ma_crossover_v1
CANDIDATE RESELECTED = NO
OPERATIONAL SIGNAL SEMANTICS FROZEN = YES
INDICATOR SEMANTICS FROZEN = YES
PRICE / CORPORATE-ACTION MODE FROZEN = YES
EXIT SEMANTICS FROZEN = YES
PRIMARY METRIC DIRECTION-SYMMETRIC = YES
COST MODEL FROZEN = YES
BEARISH ECONOMICS FROZEN = YES
BOOTSTRAP CONTRACT FROZEN = YES
FINAL HOLDOUT = SEALED
PERFORMANCE EXPERIMENT EXECUTED = NO
REAL FORWARD PAPER STARTED = NO
PHASE 13 STARTED = NO
```

## VAL-02B continuation

VAL-02A was externally accepted. The final execution-economics and statistical
details are now frozen by
`Docs/validation/VAL-02B-FINAL-PREREGISTRATION-CLOSURE.md` and manifest contract
version `v1.2`. VAL-02B supersedes this document only for the exact notional,
cash-flow arithmetic, short-cost application, statistical test and resampling
details; it does not change the selected candidate or reopen any result.
