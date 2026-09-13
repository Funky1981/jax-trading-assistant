# Jax VAL-02C Broker-Realism & Null-Mechanics Closure

## Status and scope

`VAL-02` candidate selection and `VAL-02A` are accepted. `VAL-02B` is a
partial closure and is superseded only for its remaining economics and
statistical-mechanics defects. This is a contract-only package for the already
selected `ma_crossover_v1`. No candidate was reselected and no historical
performance, return, P&L, OOS, 2025, broker, paid-data, hosted-AI,
forward-paper or Phase-13 action occurred.

`VAL-02C = FINAL PREREGISTRATION CONTRACT CLOSURE COMPLETE — EXTERNAL REVIEW
REQUIRED`; `VAL-03 = NOT STARTED / BLOCKED BY VAL-02C`.

## Candidate preservation and code identity

The frozen candidate is `MACrossoverStrategy.Analyze` followed by
`signal.Type != HOLD`, confidence `>= 0.60`, and existing deterministic
validity/risk checks. The source-only bullish pullback remains excluded because
it emits approximately `0.55` confidence and the operational gate rejects it.

Repository baseline: `58426c6fe5a0da85c970dccc4bb738bd48a605c5`.

| Implementation | Path | Git blob SHA-1 |
| --- | --- | --- |
| Strategy | `libs/strategies/ma_crossover.go` | `fb22b9dd909a0ae5f13dd727ff2eddf40963d426` |
| Strategy contract | `libs/strategies/strategy.go` | `8ff03f23aed6dba929cdc83cda5c1a1153da616f` |
| Operational inputs/SMA/volume | `internal/trader/signalgenerator/inprocess.go` | `4a3f0537013308f40ee94c3eb723d3a152297048` |
| ATR reference | `internal/modules/quant/volatility.go` | `6fb7cee0e1643352f073022bbf999283b5a04a49` |
| Actionable gate/first target | `cmd/trader/instance_scheduler.go` | `54244f758599e97317d31073f8c23f1d2b4ae729` |

## IBKR authoritative pricing evidence

Official sources were checked on `2026-09-13`:

- [IBKR US stocks and ETF commissions](https://www.interactivebrokers.com/en/pricing/commissions-stocks.php)
- [IBKR UK commissions and fees](https://www.interactivebrokers.co.uk/en/index.php?f=49807)
- [IBKR short-securities availability](https://www.interactivebrokers.com/en/trading/short-securities-availability.php)
- [IBKR short-sale cost](https://www.interactivebrokers.com/en/pricing/short-sale-cost.php?menu=A)

The selected primary plan is `IBKR Pro Fixed` for US exchange-listed stocks and
ETFs: `USD 0.005/share`, minimum `USD 1.00/order`, maximum `1% of trade
value`. Tiered pricing was not selected. Official pages also describe
pass-through/route/account qualifications; those remain account-level
uncertainties and are not silently treated as research facts.

## Primary base cost model

`cost_val03_ibkr_fixed_personal_base_v1`, contract
`jax.val03.personal_execution_cost/v1`, is frozen before VAL-03. Broker facts
are distinct from modelling assumptions. For a leg with whole-share quantity
`q`, price `P`, and notional `V=q*P`:

```text
commission = min(max(0.005*q, 1.00), 0.01*V)
market_friction = V*(4+5+0)/10000
leg_cost = commission + market_friction
```

The `4 bps` spread and `5 bps` slippage per leg are existing Jax daily/swing
assumptions for liquid ETFs. `0 bps` impact is an explicit bounded base
assumption because daily OHLCV cannot identify order-book impact. They are not
published IBKR fee facts. Arithmetic is high precision until final display,
which uses round-half-up cents. `ExecutionAuthority=NONE`; `CreatesFill=false`.

## Stress cost model

`cost_4c9e2fbffb41d05b537eadf05b142c8f8d00ed46d672d75bca5f83a4269404f8` is
preserved as `LEGACY DETERMINISTIC FIXTURE / CONSERVATIVE STRESS SCENARIO
ONLY`, sourced from the Phase-07 deterministic fixture: USD 0.50/order, 10 bps
commission, 5 bps spread, 10 bps slippage, 15 bps market impact per leg and a
20 bps/day borrow field. Long-only stress applies the non-borrow fields; the
preserved borrow field is not applied to the primary claim. Base is primary;
stress cannot replace it after outcomes.

## Reference capital and quantity

Reference capital is `USD 10,000` per episode. The frozen convention is whole
shares: `quantity=floor(10000/next_open)`, requiring quantity at least one.
Unused capital remains cash; entry notional is `quantity*next_open`; return
denominator is actual deployed entry notional. Fractional analytic quantity is
not permitted.

## Short-side decision

`PRIMARY PERSONAL TRADABLE CLAIM = ACTIONABLE BUY / LONG EPISODES ONLY`.
`SELL / BEARISH EPISODES = SECONDARY DIRECTIONAL RESEARCH DIAGNOSTIC ONLY`.
IBKR's official material says availability is indicative, approval applies,
and borrow rates vary by symbol and time. The repository has no approved
point-in-time 2016–2024 borrow availability/fee series. The old universal
20 bps/day assumption is not used to claim short performance. A future short
claim requires a new point-in-time borrow, financing, availability and dividend
contract. Bearish observations remain retained and cannot rescue a failed long
claim.

## Pre-entry invalidation and exit

There is no entry before the next open. A long is valid only when
`stop < next_open < target`. A short diagnostic is valid only when
`target < next_open < stop`. Otherwise record `PRE_ENTRY_INVALIDATED`,
abstention and no episode. Later opening crosses resolve at the open; later
touches resolve at the level; stop wins when both are touched. No impossible
fills, same-bar entry, scale-out or partial profit are allowed.

Stop/target levels are fixed at signal time: long stop `SMA50-ATR14`, long
target `signal_close+3*ATR14`; bearish equivalents are `SMA50+ATR14` and
`signal_close-3*ATR14`. Only `TakeProfit[0]` is used. Exit is the first valid
stop/target or the twentieth future regular-session close.

## Primary metric and comparative null

The primary unit is an eligible non-overlapping bullish/long episode:

```text
M = mean(base-cost net_return across eligible primary long episodes)
H0: M <= 0
```

The primary comparative gate is the mean paired difference between each actual
episode and its deterministic matched non-signal fixed-horizon long placebo.
The paired difference must be positive with a block-bootstrap lower bound above
zero. A positive strategy mean alone is insufficient. Secondary diagnostics are
zero return, SPY matching-date buy-and-hold, same-asset buy-and-hold, gross/net,
duration, theme and regime slices; same-interval buy-and-hold is not primary.

## Bootstrap and randomisation

Resample non-empty `(instrument, calendar year)` blocks, exactly 10,000
replicates, preserve all episodes in sampled blocks, carry actual/placebo pairs
together, and use a percentile 95% interval. The lower bound is the 2.5th
percentile; empty blocks are excluded; IID inference is prohibited. Formal OOS
requires at least 12 non-empty blocks across 6 instruments, 30 primary episodes,
30 matched pairs, no instrument above 40% and at least 3 calendar/regime slices.
Failure is `INSUFFICIENT_EVIDENCE`.

Sign permutation is `NOT APPLICABLE TO PRIMARY LONG-ONLY CLAIM`; it remains a
secondary mixed-direction diagnostic only. If run, it uses 10,000 assignments
within instrument-year strata, single-direction blocks are
`NOT_APPLICABLE`, and p-value is `(1+count(null>=observed))/(1+10000)`.
All random choices use domain-separated SHA-256 counter-derived bytes, not
wall-clock or process-global RNG. Domains are `|instrument-year-bootstrap-v2|`,
`|secondary-sign-permutation-v1|`, `|timestamp-placebo-v1|` and
`|matched-placebo|`, each derived from the frozen manifest SHA-256.

## Placebo contracts

Timestamp placebo: 10,000 replicate sets; each draw is same instrument, same
calendar year, regular session, non-actionable, outside an active episode, with
20 future valid sessions and raw/provenance records. Select the minimum
SHA-256 score of `seed|replicate|instrument|candidate_date`; never relax and
never use future returns.

Matched non-signal placebo: one per actual primary signal; same instrument,
calendar year and broad pre-signal regime (SPY close above/below available
SMA200); non-actionable, not active, 20 future valid sessions, within 60
trading sessions. Choose nearest trading-session distance, SHA-256 tie-break;
no match is recorded and does not delete the actual. Minimum pairs: 30. Placebo
outcome is whole-share long from its next open to its twentieth future close
with the same costs and point-in-time dividends; signal stops/targets are not
transferred.

## Exact robustness variants

Falsification-only variants are: SMA `(19,49,199)` and `(21,51,201)`; stop and
target distance `0.90*ATR14` and `1.10*ATR14` with ATR period fixed; remove
`max(1,ceil(0.05*N))` highest-return episodes after descending net-return
ranking while retaining identities; themes broad `{SPY,QQQ,IWM,DIA}`, sector
`{XLK,XLF,XLE}`, rates/precious `{TLT,GLD}`; and overlap sensitivity retaining
the earliest signal per instrument in each 20-session entry-date window.
Leave-one-instrument-out, SPY/SMA200 regime, zero and buy-hold baselines and
legacy stress are also fixed. No variant can replace the candidate.

## Data provider order

The only source is the existing hardened Alpaca route in
`libs/marketdata/alpaca_hardened.go`, source
`src_alpaca_stock_bars_sip_unadjusted`, SIP, `1Day`, raw/unadjusted, fixed nine
ETF basket, `2015-01-01` through `2024-12-31`. Raw bytes must precede
normalization and carry request identity, retrieval timestamp, source identity
and hash. No fallback is defined: incomplete range/feed/adjustment/provenance
is `DATASET_BLOCKED`; no provider shopping; no 2025 download.

## Holdout and execution boundary

2025 is `SEALED`; only metadata/schema/hash operations are allowed. No bars,
signals, counts, returns, calibration, semantics or debugging may access it.
HYP-EVENT-001A's contaminated 2024 lineage is not reused. VAL-03 remains
unauthorized by this manifest and this closure performs no performance
calculation. Safety is hypothetical only: no broker, order, fill, hosted AI,
paid API, paper or Phase-13 action.

## Manifest

The machine-readable contract is
`Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json`,
contract `v1.3`, status `FROZEN_FOR_EXTERNAL_REVIEW`. Its SHA-256 is recorded in
the final handover after deterministic hashing.

```text
SELECTED CANDIDATE = ma_crossover_v1
CANDIDATE RESELECTED = NO
PRIMARY BROKER COMMISSION = SOURCED FROM CURRENT AUTHORITATIVE IBKR TERMS
OLD PHASE-07 FIXTURE = LEGACY / STRESS ONLY
BASE PERSONAL COST MODEL = FROZEN
STRESS COST MODEL = FROZEN
REFERENCE CAPITAL = USD 10,000
QUANTITY CONVENTION = FROZEN
HISTORICAL SHORT ECONOMICS = NOT POINT-IN-TIME SUPPORTED / BEARISH SECONDARY ONLY
PRE-ENTRY GAP INVALIDATION = FROZEN
PRIMARY COMPARATIVE NULL = FROZEN
RANDOM SEEDS = FROZEN
TIMESTAMP PLACEBO = FROZEN
MATCHED NON-SIGNAL PLACEBO = FROZEN
ROBUSTNESS VARIANTS = EXACTLY FROZEN
DATA PROVIDER ORDER = FROZEN
FINAL HOLDOUT = SEALED
PERFORMANCE EXPERIMENT EXECUTED = NO
REAL FORWARD PAPER STARTED = NO
PHASE 13 STARTED = NO
```
