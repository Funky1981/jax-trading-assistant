# Jax VAL-03B Adjusted-Bar Equivalence & Price-Edge Contract

## Status

`VAL-03B = DATA CONTRACT BLOCKED`.

The adjusted-bar contract is internally consistent, but the authorized Alpaca
preflight did not cover the required `2015-01-01` warm-up boundary. VAL-03 has
not resumed and no strategy performance was calculated.

## External roadmap-change decision

- VAL-03A: complete at the investigation gate; the v1.3 event-ledger contract
  remains blocked.
- VAL-03 v1.3: `DATASET_BLOCKED`.
- Roadmap change: adjusted-bar pre-outcome data contract authorized.
- VAL-03B: adjusted-bar contract implemented and preflighted; dataset readiness
  blocked by missing requested-start coverage.

## Candidate and frozen terms

The selected candidate remains `ma_crossover_v1`. No candidate, strategy
parameter, cost model, sample floor, placebo, benchmark, robustness variant,
partition or holdout term changed.

The v1.3 manifest was amended only for corporate-action/data-price semantics.

## Official Alpaca adjustment semantics

Source: https://docs.alpaca.markets/us/reference/stockbars

Retrieved 2026-09-13. The documented historical-bars API supports:

- `raw`: no adjustments;
- `split`: price and volume adjusted for forward and reverse splits;
- `dividend`: price adjusted for cash dividends;
- `spin-off`: price adjusted for spin-offs;
- `all`: all documented adjustments;
- combinations such as `split,spin-off`.

The API also supports `asof=YYYY-MM-DD` for symbol/entity identification. The
VAL-03B contract uses `asof=2024-12-31` and does not use dividend or all-adjusted
bars.

## v1.4 scientific rationale

The v1.3 contract required a complete point-in-time corporate-action event
ledger that Alpaca did not establish. The v1.4 amendment instead tests a
conservative price-only claim using synchronized adjustment families:

```text
RAW + SPLIT + SPLIT,SPIN-OFF
        |
        +--> factor/structural quality checks
        +--> future VAL-03 signal/geometry inputs only from SPLIT
        +--> future VAL-03 execution/cost reference only from RAW
```

This amendment was made before any strategy signal, episode, return or OOS
result was calculated. It is not identical to the abandoned total-return
contract.

## Primary historical claim

`NET PRICE RETURN EXCLUDING CASH DISTRIBUTIONS` after the frozen base trading
cost model. Cash distributions cannot create or improve a historical signal,
entry, exit, eligibility decision, placebo match or primary return.

Future forward paper must account prospectively for actual broker-recorded
dividends, distributions, splits and position events.

## Signal bar series

`ALPACA / SIP / 1Day / adjustment=split`.

SMA20, SMA50, SMA200, ATR14, AvgVolume20, MarketTrend, confidence, stop,
target, entry validity and exit geometry will use this series if VAL-03 later
resumes.

Dividend, all-adjusted and spin-off-adjusted bars are not signal inputs.

## Raw execution-reference series

`ALPACA / SIP / 1Day / adjustment=raw`.

RAW is reserved for quoted-price reference, whole-share sizing, per-share
commission arithmetic and structural cross-checks. RAW does not independently
generate signals.

## Entity/as-of contract

All three families use `asof=2024-12-31`. No present-day mapping or 2025 as-of
date is permitted.

## Split adjustment factor contract

For each synchronized session, calculate the four price ratios:

```text
split_open  / raw_open
split_high  / raw_high
split_low   / raw_low
split_close  / raw_close
```

They must form one positive common factor within relative tolerance `5e-4`,
which is frozen to accommodate provider quote-precision rounding. Where both
volumes are non-zero, `raw_volume / split_volume` must match the same factor.

An adjacent-session factor change is a structural boundary. The detector does
not infer an action type from a market return.

## Scale-invariance proof

Deterministic synthetic tests pass for positive split-scale transformation of:

- MA ordering;
- close-versus-MA comparisons;
- inverse volume/average-volume comparison;
- ATR scaling;
- pre-entry geometry;
- stop and target touch conditions.

The tests contain no market outcomes or strategy-performance assertions.

## 200-session split reset

Preserved unchanged:

`detected split boundary -> no primary signal eligibility until 200 valid
post-split sessions exist`.

## Split-spanning episode policy

Any future episode crossing a detected structural split boundary is:

`STRUCTURAL_ACTION_INVALIDATED`.

It cannot contribute to the primary result, placebo gate or promotion evidence.

## Dividend/distribution policy

Primary historical cash credit: `NONE`.

Dividend, capital-gains and other cash distributions are not included in the
primary price return and cannot affect signal or episode eligibility.

## Structural spin-off detector

`split,spin-off` is acquired only as a structural detector. It is not used for
signals. A difference between SPLIT and SPLIT+SPIN-OFF would produce
`UNSUPPORTED_STRUCTURAL_ACTION` before strategy evaluation.

Observed preflight result: no SPLIT versus SPLIT+SPIN-OFF boundaries across the
completed nine-symbol family comparisons.

## Hardened Alpaca adapter changes

`libs/marketdata/alpaca_hardened.go` now supports only these explicit research
identities in addition to the retained legacy identity:

- RAW (`raw`);
- SPLIT (`split`);
- SPLIT+SPIN-OFF (`split,spin-off`).

The request carries an optional explicit `asof` date. Adjustment identity is
included in source IDs, normalizer IDs/versions, external IDs and observation
fingerprint seeds. Generic `ADJUSTED`, dividend and all-adjusted modes remain
unsupported.

The adapter persists exact provider bytes before normalization through the
existing raw-payload store. It has no broker or trading authority.

## Source/provenance identities

- Provider: `pvd_alpaca_market_data`
- Feed: SIP
- Timeframe: 1Day
- As-of: 2024-12-31
- Families: `raw`, `split`, `split,spin-off`
- Raw payloads: persisted locally under ignored `.runtime/val03b/raw/`
- No raw payloads are committed

## Test coverage

Focused deterministic tests cover:

- RAW, SPLIT and SPLIT+SPIN-OFF request construction;
- `asof` propagation;
- unsupported adjustment rejection;
- adjustment-bearing source identity separation;
- date-range and returned-date holdout rejection;
- common split-factor validation;
- structural boundary detection;
- MA, volume, ATR, stop, target and pre-entry scale invariance.

## Bar-family acquisition

The dedicated command `cmd/val03b-bar-preflight` acquired, through the hardened
route, all 27 permitted family requests: nine ETFs × RAW/SPLIT/SPLIT+SPIN-OFF.

No DIVIDEND or ALL request was made. No market-bar request included a date on or
after 2025-01-01.

## Data-quality results

For all 27 family results:

- first returned session: 2016-01-04;
- last returned session: 2024-12-31;
- records: 2,264 per family;
- duplicate sessions: 0;
- non-monotonic sessions: 0;
- OHLC validity: PASS;
- volume validity: PASS;
- synchronized session count: 2,264 per instrument;
- returned 2025 rows: 0.

The requested range begins 2015-01-01. All 27 families failed the requested
start-coverage check because the earliest returned session was 2016-01-04.
This is a dataset blocker, not a strategy result.

## Structural factor boundaries

No unexplained RAW/SPLIT factor anomalies remained under the frozen `5e-4`
quote-precision tolerance. No structural factor boundary was recorded in the
completed comparisons.

## Unsupported structural events

No SPLIT versus SPLIT+SPIN-OFF difference was observed in the completed family
comparisons. This does not waive the missing 2015 coverage blocker.

## Dataset-readiness artifact

`Docs/validation/results/VAL-03B-DATASET-READINESS.json`

Artifact status:

`BLOCKED_INCOMPLETE_DATE_COVERAGE`

It contains only provider/family quality metadata and hashes. It contains no
signals, episodes, returns, P&L or performance metrics.

## VAL03B_BAR_FAMILY_SHA256

Artifact SHA-256:

`78c1fbeec2122eebba1a8dbe21c2edfc3c569377770561afe9e128cd633037e5`

## Manifest v1.4

The manifest was advanced to `v1.4` and remains:

`FROZEN_FOR_EXTERNAL_REVIEW`

It freezes the adjusted-bar price-edge semantics, but VAL-03 cannot resume until
the required requested-start data coverage is resolved under the authorized
Alpaca/no-fallback boundary.

## Manifest SHA-256

`6a7cab890f5300d13a05912c19298b9839b4fea68fdf3fc48604766fb2898709`

## Why amendment is pre-outcome

- OOS run count before amendment: 0
- No strategy performance calculated
- No signals or episodes generated
- Candidate unchanged
- Parameters unchanged
- Costs unchanged
- Sample floors unchanged
- Placebo design unchanged
- Partitions unchanged
- Robustness suite unchanged
- 2025 untouched

## Exact VAL-03B status

`STRUCTURAL BAR CONTRACT = BLOCKED`

The adjusted-bar semantics and adapter tests pass, but the full readiness gate
fails because the required 2015-01-01 warm-up coverage is unavailable from the
authorized Alpaca response.

`VAL-03B = DATA CONTRACT BLOCKED`

## Safety

- `ALLOW_LIVE_TRADING=false` preserved
- `BROKER_EXECUTION_ALLOWED=false` preserved
- `EXECUTION_ENABLED=false` preserved
- Maximum leverage remains `1x`
- `ExecutionAuthority = NONE`
- `CreatesFill = false`
- No broker calls or mutations
- No orders, fills, approvals, trades or positions
- No hosted inference
- No paid data purchase
- No 2025 access
- No forward paper
- Phase 13 not started
