# Jax VAL-02 Candidate Selection & Scientific Preregistration

## Starting state

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Starting HEAD: `530a90916770f680f391bfb85d0e68ef689eb5b8`
- Starting `origin/capability-reset`: `530a90916770f680f391bfb85d0e68ef689eb5b8`
- Starting divergence: `0 ahead / 0 behind`
- Starting worktree: clean
- VAL-02 is a design/freeze package. No candidate performance experiment was run.

## VAL-01 external decision

`VAL-01 = COMPLETE / GO` and `FORWARD PAPER ADMISSION = NO-GO` are recorded in
current governance. The NO-GO means that no candidate had sufficient frozen
evidence for prospective promotion; it is not a failure of the platform.

`HYP-EVENT-001A = NOT VALIDATED` remains unchanged. Real forward-paper evidence
remains `0 DAYS / 0 ORDERS`.

## Candidate pool

The allowed pool is the VAL-01 inventory. HYP-EVENT-001A is excluded from
selection because it is `CONTAMINATED_FOR_FORMAL_OOS`; its 2025 holdout remains
sealed. No new strategy was introduced.

The pool comprises:

- `rsi_momentum_v1`
- `macd_crossover_v1`
- `ma_crossover_v1`
- `same_day_earnings_drift_v1`
- `same_day_news_repricing_v1`
- `news_shock_momentum_v1`
- `opening_range_to_close_v1`
- `event_gap_continuation_v1`
- `panic_reversion_v1`
- `pairs_event_relative_v1`
- `index_flow_v1`
- `etf_news_market_panic_reversal_v1`
- `etf_news_sector_momentum_v1`
- `etf_news_rates_bonds_rotation_v1`
- `SWING_BRAIN_V1`

The documented `hyp_swing_001` and `hyp_commodity_dislocation` values are
templates/fixtures, not candidates. Candidate-routing proof artifacts are
safety evidence, not strategy-performance evidence.

## Outcome-blind selection criteria

Selection uses only pre-outcome structural feasibility and personal-product
fit. No return, P&L, Sharpe, win rate, drawdown, alpha, screenshot or OOS result
was used.

The frozen weights were set before scoring and sum to 100:

| Criterion | Weight |
| --- | ---: |
| A. Personal-product alignment | 10 |
| B. Economic plausibility | 10 |
| C. Point-in-time tractability | 12 |
| D. Data availability | 12 |
| E. Sample feasibility | 10 |
| F. Cost-model tractability | 10 |
| G. Survivorship/universe risk | 10 |
| H. Falsifiability | 8 |
| I. Parameter complexity | 6 |
| J. Operational feasibility | 6 |
| K. Model dependence | 4 |
| L. Validation cost | 2 |

Scores are ordinal feasibility scores only: `3 = HIGH`, `2 = MEDIUM`,
`1 = LOW`. Weighted totals are out of 300 and do not estimate profitability.

## Frozen selection weights

The weights above and the scoring interpretation are immutable for VAL-02.
Changing them after this record would create a new VAL-02 preregistration.

## Candidate feasibility matrix

Vector order is `A B C D E F G H I J K L`. The matrix records structural
feasibility before performance inspection.

| Candidate | Score vector | Weighted score / 300 | Feasibility rationale |
| --- | --- | ---: | --- |
| `rsi_momentum_v1` | `2 2 3 3 3 3 3 3 3 3 3 3` | 280 | Deterministic and cheap, but reversal mechanism and short-direction/cost interpretation are less clean. |
| `macd_crossover_v1` | `3 3 2 3 3 3 3 3 2 3 3 3` | 282 | Deterministic trend thesis, but more indicator/state complexity than MA. |
| `ma_crossover_v1` | `3 3 3 3 3 3 3 3 3 3 3 3` | 300 | Small deterministic rule surface, daily OHLCV inputs, transparent trend mechanism, low-cost replay and clear falsification. |
| `same_day_earnings_drift_v1` | `3 3 1 1 2 2 2 3 2 1 3 2` | 202 | Plausible event mechanism, but timestamped earnings, intraday data and point-in-time coverage are not frozen. |
| `same_day_news_repricing_v1` | `3 3 1 2 2 2 2 3 2 2 2 2` | 216 | Publication timing, evidence selection and intraday execution remain unresolved. |
| `news_shock_momentum_v1` | `3 3 1 2 2 2 2 3 2 2 2 2` | 216 | Historical news/event provenance and sample definition are incomplete. |
| `opening_range_to_close_v1` | `2 3 2 2 2 2 3 3 2 2 3 2` | 232 | Clear rule, but requires higher-frequency data and fragile fill/session assumptions. |
| `event_gap_continuation_v1` | `3 3 1 2 2 2 2 3 2 2 2 2` | 216 | Event-time and gap construction are not frozen. |
| `panic_reversion_v1` | `2 3 1 2 2 2 2 3 1 2 2 2` | 204 | Intraday tail/liquidity risk and parameter sensitivity are high. |
| `pairs_event_relative_v1` | `3 3 1 2 1 2 1 3 1 1 2 2` | 174 | Point-in-time peer selection and survivorship are unresolved; source marks it research-first. |
| `index_flow_v1` | `2 2 2 2 2 2 3 3 2 2 3 2` | 214 | Narrow universe helps, but intraday context and exits remain incomplete. |
| `etf_news_market_panic_reversal_v1` | `3 3 2 2 2 2 3 3 2 2 2 2` | 234 | Good fit, but news timing, intraday bars and no evidence bundle reduce feasibility. |
| `etf_news_sector_momentum_v1` | `3 3 2 2 2 2 2 3 2 2 2 2` | 224 | Sector mapping, event timing and intraday validation remain open. |
| `etf_news_rates_bonds_rotation_v1` | `3 3 1 2 2 2 2 3 2 2 2 2` | 216 | Macro-event timing/vintages and intraday execution increase data burden. |
| `SWING_BRAIN_V1` | `3 3 1 2 2 2 1 3 1 2 2 2` | 198 | Strong product alignment, but it is a multi-family assessment system rather than one frozen trading rule. |

No candidate received a score from observed performance. The high score for
`ma_crossover_v1` reflects tractability and low degrees of freedom, not an
assertion that it will outperform.

## Selected candidate

`SELECTED CANDIDATE ID = ma_crossover_v1`

`SELECTED CANDIDATE VERSION = ma_crossover_v1`

`SELECTION UNIT = MACrossoverStrategy source contract in
libs/strategies/ma_crossover.go, evaluated on a preregistered fixed ETF basket`

The candidate remains a research candidate and is not admitted to forward
paper.

## Why selected

`ma_crossover_v1` is the smallest existing candidate with deterministic
OHLCV-only inputs, no hosted model or paid inference dependency, a defensible
trend-persistence mechanism, transparent signal reasons, daily-data
compatibility, low parameter count and direct compatibility with replay, cost
and falsification contracts.

## Why other candidates were not selected first

- `rsi_momentum_v1` is deterministic but has a less conservative reversal thesis
  and greater threshold sensitivity.
- `macd_crossover_v1` has additional indicator/state complexity without a data
  advantage.
- Intraday/event candidates require unresolved timestamped event or higher
  frequency data and more fragile fill assumptions.
- `pairs_event_relative_v1` has unresolved peer and survivorship construction.
- `SWING_BRAIN_V1` is multi-family logic, not one frozen rule.
- `HYP-EVENT-001A` is excluded by its contaminated formal-OOS lineage, sealed
  holdout and accepted survivorship limitation.

## Performance-blindness proof

Selection was made from source contracts, documented metadata, data
requirements, safety/replay architecture and VAL-01 evidence gaps. No
candidate return, P&L, Sharpe, win rate, drawdown, alpha, backtest ranking or
OOS result was calculated or inspected during VAL-02.

## Prior-knowledge / contamination ledger

| Item | Classification | Treatment |
| --- | --- | --- |
| Strategy source and unit tests | `DEVELOPMENT-EXPOSED` | Used only to define the existing rule surface. |
| Generic legacy backtest implementation/tests | `DEVELOPMENT-EXPOSED` | Not treated as candidate validation. |
| VAL-01 candidate inventory | `DEVELOPMENT-EXPOSED` | Used as the fixed candidate pool. |
| Existing paper-instance JSON files | `DEVELOPMENT-EXPOSED` | Disabled instances are not paper evidence. |
| HYP-EVENT-001A 2024 lineage | `CONTAMINATED` | Excluded from selection and not repaired. |
| HYP-EVENT-001A 2025 holdout | `SEALED` | Not accessed. |
| Candidate-specific prior return results | `NOT FOUND IN INVENTORIED EVIDENCE` | No return data used for selection. |

The VAL-03 line starts with a new candidate-specific protocol identity. No
period is called pristine merely because source code existed before VAL-02.

## Economic hypothesis

## VAL-02A closure override

`VAL-02` candidate selection remains accepted. `VAL-02A` and `VAL-02B` are
historical closure records; the remaining broker-realism and null-mechanics
requirements are frozen by `VAL-02C`, manifest contract `v1.3`. VAL-03 remains
not started and no performance outcome has been calculated.

The original VAL-02 design text contains intentionally unresolved wording that
was identified before VAL-03. It is superseded for execution by
`Docs/validation/VAL-02A-PREREGISTRATION-CLOSURE.md` and manifest contract
version `v1.1`. That closure binds the operational signal gate, excludes the
source-only 0.55 pullback from actionable episodes, fixes raw point-in-time
price/corporate-action semantics, selects the first operational target, binds
the exact cost policy, makes the primary metric direction-symmetric, separates
episode and portfolio analyses, and freezes block-bootstrap inference. The
historical VAL-02 selection record remains unchanged; no result was calculated.

**HYP-MA-001:** For the fixed preregistered basket of liquid US-listed ETFs,
when the existing `ma_crossover_v1` rule observes a valid daily bullish or
bearish alignment at a regular-session close, entering at the next regular
session open and exiting at the first frozen stop/target or the twentieth
future trading-session close will produce a positive mean **net**
benchmark-relative signed return per non-overlapping signal episode, and will
exceed the same-interval per-asset buy-and-hold comparator in formal OOS.

The null is that the candidate does not exceed the comparator after the frozen
cost model. A positive result is not presumed and is not sufficient by itself
for forward-paper admission.

## Strategy contract

Source contract: `libs/strategies/ma_crossover.go`, ID/version
`ma_crossover_v1`.

- Universe: `SPY`, `QQQ`, `IWM`, `DIA`, `XLK`, `XLF`, `XLE`, `TLT`, `GLD`.
- This fixed ETF basket is selected before outcome inspection; no current
  equity index membership is used.
- Daily regular-session bars using a US/Eastern calendar.
- Signal is evaluated after the close using only data through that close.
- Entry is the next regular-session open strictly after the signal close.
- Existing bullish and bearish source directions are preserved.
- Existing SMA20/SMA50/SMA200 alignment and pullback rules are preserved.
- Source stop and target levels are fixed from the signal-time output.
- Exit is the first stop or target reached, otherwise the twentieth future
  session close.
- If one daily bar crosses both levels, stop is conservatively first. Gaps fill
  at the open.
- No re-entry while an instrument episode is active.
- Equal-notional measurement sizing is capped at 1x aggregate gross exposure.
- Bearish signals are signed research measurements, not short-execution
  authorization.
- Missing, stale, invalid or incomplete data produces a recorded abstention,
  never an imputed trade.
- Maximum leverage is `1x`.

## Parameter contract

| Parameter | Value | Classification |
| --- | --- | --- |
| SMA fast | 20 sessions | Fixed by existing strategy |
| SMA medium | 50 sessions | Fixed by existing strategy |
| SMA slow | 200 sessions | Fixed by existing strategy |
| Pullback lower bound | -2% | Fixed by existing strategy |
| Pullback upper bound | +1% | Fixed by existing strategy |
| Crossover stop/target ATR multipliers | 3 ATR / 5 ATR | Fixed by existing strategy |
| Pullback stop/target ATR multipliers | 1.5 ATR / 2 ATR / 3.5 ATR | Fixed by existing strategy |
| Maximum holding period | 20 sessions | Structural, frozen before results |
| Aggregate gross exposure | 1x | Safety convention |
| Parameter search | None | Not tunable in VAL-03 |

There is one permitted candidate configuration and one permitted parameter
trial. Sensitivity tests are falsification records only.

## Data requirements

VAL-03 requires daily historical bars for the fixed basket from
`2016-01-01` through `2024-12-31`; the first provider session is expected to be
`2016-01-04`. Initialization is per instrument and requires 200 valid
synchronized SPLIT-adjusted sessions before development scoring. The final
2025 holdout remains sealed. The source is the existing approved Alpaca
historical route. VAL-02 acquired no scored data.

Required fields include RAW and SPLIT OHLCV plus the SPLIT+SPIN-OFF structural
detector, adjustment-family provenance,
symbol identity/history, UTC and US/Eastern timestamps, trading calendar,
completeness/staleness flags, dataset identity and content hash.

## Universe / survivorship contract

The fixed ETF basket avoids silently using today's equity survivors or current
index membership. Inception, closure, renaming, split and dividend history
must be recorded. Closed instruments are not replaced. The inference population
is limited to this basket and cannot be generalized to all equities.

## Temporal / knowability contract

- Signal cutoff: regular-session close.
- Earliest entry: next regular US equity session open.
- No same-bar entry.
- Indicator windows contain only bars available at the cutoff.
- Corporate-action treatment must not use future knowledge.
- Calendar, DST and UTC conversion are deterministic and auditable.

## Partition map

| Partition | Dates | Permitted use |
| --- | --- | --- |
| Initialization | First provider session through each instrument's 200th valid synchronized SPLIT session | `WARMUP_ONLY`; not scored. |
| Development | 2016-01-01 through 2020-12-31, scorable only after initialization | Pipeline and descriptive development checks. |
| Validation | 2021-01-01 through 2022-12-31 | Protocol verification and registered diagnostics. |
| Formal OOS | 2023-01-01 through 2024-12-31 | One frozen candidate evaluation. |
| Final holdout | 2025-01-01 through 2025-12-31 | Sealed; no VAL-02/ordinary VAL-03 access. |

No HYP-EVENT-001A partition is reused as this line's formal OOS.

## Expected sample

Expected workload before outcomes is approximately 9 instruments × 2,264
available sessions across 2016–2024, with the first 200 valid sessions per
instrument reserved for initialization. The final 2025 holdout is sealed.
Exact signal counts are not computed in VAL-02.

Pre-registered floors for VAL-03 are:

- all 9 instruments have valid session/calendar coverage;
- at least 90 non-overlapping development episodes;
- at least 30 validation episodes;
- at least 30 formal OOS episodes;
- at least 15 metadata-verified holdout observations before any later external
  unsealing decision;
- no instrument contributes more than 40% of scored episodes;
- at least three calendar/regime slices contain observations.

Failure yields `INSUFFICIENT_EVIDENCE`; periods are not merged and filters are
not loosened.

## Benchmark hierarchy

- Primary: same-interval per-asset buy-and-hold.
- Market: SPY buy-and-hold over the same dates.
- Null: zero systematic benchmark-relative effect.
- Random: registered within-instrument/year sign and timestamp permutations.

## Cost model

Bind to the accepted Phase-11 versioned cost model before VAL-03 execution.
Record fees, spread, slippage, latency, session, fill and partial-fill
semantics. Report gross and net results. If signed bearish measurement cannot
be represented honestly, stop as `INSUFFICIENT_EVIDENCE` rather than omit costs.

## Primary metric

`PRIMARY METRIC = mean net benchmark-relative signed return per non-overlapping
episode`, compared with same-interval per-asset buy-and-hold in formal OOS.

Uncertainty uses an instrument-year block bootstrap with a 95% interval,
preserving within-block temporal dependence. Naive IID confidence is not used.
Primary success requires a positive candidate-minus-comparator estimate and a
95% lower bound above zero, with no rejection condition.

Secondary diagnostics are gross/net return, holding duration, drawdown, count,
instrument/year/regime slices, cost stress and concentration.

## Falsification suite

1. Within-instrument/year randomized signal signs.
2. Within-instrument/year constrained timestamp permutation.
3. Matched non-signal placebo dates.
4. Exclusion of the top 5% of episode contributions.
5. Leave-one-instrument-out sensitivity.
6. Predefined SPY-above/below-SMA200 regime split.
7. Predefined sector/theme subset reporting.
8. One-session SMA and 10% ATR perturbations as robustness records only.
9. Two-times spread/slippage stress.
10. One episode per instrument per 20-session overlap sensitivity.
11. Buy-hold, SPY and null baseline comparisons.

Placebo equivalence, cost removal, instability, concentration or failure of the
primary uncertainty rule rejects or leaves the candidate unpromoted.

## Multiple-testing budget

- Candidate variants: 1.
- Parameter trials used for selection: 1.
- Model/prompt variants: 0.
- Validation retries changing the candidate: 0.
- Falsification/robustness variants: exactly the registered list.
- Failed trials may not be renamed or deleted.

## Calibration / resolution plan

The source confidence field is heuristic, not a calibrated probability.
Probability calibration is `NOT APPLICABLE` for this preregistration. VAL-03
may report ordinal confidence resolution descriptively; confidence cannot be
used as a post-result filter.

## Abstention plan

Record no-signal/hold observations, invalid or incomplete data, stale-data
abstentions, risk/sizing vetoes and active-position suppression. Coverage and
abstention rates must be reported beside episode results.

## Concentration plan

Report instrument, theme/sector, calendar-year, regime, overlap and top-N
contribution. Failure occurs if one instrument exceeds 40% of episodes, the top
5% supplies the effect, or the result exists only in one unregistered slice.

## Walk-forward protocol

`NO WALK-FORWARD REFITTING` is frozen. Parameters remain fixed across all
periods. Formal OOS is scored once after candidate freeze. Any material change
creates a new candidate version and independent evaluation requirement.

## Final holdout seal

- Date range: `2025-01-01` through `2025-12-31`.
- Status: `SEALED`.
- Before external authorization, only existence/schema/hash metadata may be
  checked.
- Bars, returns, tuning, debugging, calibration and narrative confirmation
  are prohibited.

## Success / rejection criteria

VAL-03 cannot classify the candidate above `RESEARCH_CANDIDATE` unless all
applicable gates pass: reproducibility, point-in-time correctness, sample,
costs, OOS, falsification, concentration, uncertainty and no contamination.
`HYPOTHESIS REJECTED` and `NO ROBUST EDGE` are valid outcomes.

## VAL-03 execution contract

VAL-03 may execute only after external review of this preregistration. It must
use the exact candidate, data, partitions, metric, cost model and falsification
list above. It must not calculate 2025 outcomes, change recommendations or
create paper/live orders.

## Safety

```text
ALLOW_LIVE_TRADING=false
BROKER_EXECUTION_ALLOWED=false
EXECUTION_ENABLED=false
EXECUTION WORKER=DISABLED
MAXIMUM LEVERAGE=1x
HOSTED INFERENCE=NOT AUTHORIZED
PAID DATA/API SPEND=$0
REAL FORWARD PAPER=NOT STARTED
PHASE 13=NOT STARTED
```

## VAL-02 control statements

```text
ONE CANDIDATE SELECTED = YES
SELECTED CANDIDATE = ma_crossover_v1
SELECTION USED OBSERVED PERFORMANCE = NO
SELECTION CRITERIA FROZEN BEFORE SCORING = YES
STRATEGY CONTRACT FROZEN = YES
PARTITION MAP FROZEN = YES
FALSIFICATION SUITE FROZEN = YES
TRIAL BUDGET FROZEN = YES
FINAL HOLDOUT = SEALED
HISTORICAL PERFORMANCE EXPERIMENT EXECUTED = NO
REAL FORWARD PAPER STARTED = NO
PHASE 13 STARTED = NO
SAFETY BOUNDARIES PRESERVED = YES
```

## VAL-03B continuation reference

The selected candidate remains unchanged. VAL-03B amended only the blocked
corporate-action/data-price contract to v1.4 before outcomes and documented the
adjusted-bar equivalence design. Its Alpaca preflight established complete
2016-01-04 through 2024-12-31 bar-family coverage but did not satisfy the then-
required standalone 2015 warm-up.

## VAL-03C continuation reference

VAL-03C corrects that obsolete standalone 2015 warm-up requirement using a
per-instrument 200-valid-session initialization contract. The selected
`ma_crossover_v1`, all strategy/statistical terms, validation/OOS dates and the
2025 seal remain unchanged. No performance was executed; the corrected data
contract is ready for separate external VAL-03 authorization.
