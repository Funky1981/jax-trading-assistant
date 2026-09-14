# Jax VAL-03R1A Full Manifest Semantic Conformance Closure

## Status

`VAL-03R1A = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This is an outcome-free harness correction. No historical bars were loaded by
the audit path, no strategy results were calculated, and no new OOS execution
identity was created.

## Preserved scientific history

`VAL-03` remains `CONTAMINATED_FOR_FORMAL_OOS`. Its one historical formal OOS
run, raw artifacts, hashes, and `no_rerun=true` integrity record remain
unchanged. The old output is invalid for both promotion and rejection.

The v1.5 preregistration remains authoritative and unchanged:

- Manifest: `Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json`
- Version: `v1.5`
- SHA-256: `96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`
- Candidate: `ma_crossover_v1`

## Conformance matrix

The baseline matrix is recorded in
`Docs/validation/VAL-03R1A-MANIFEST-CONFORMANCE-MATRIX.md`. Before correction,
31 material requirements were assessed as 9 PASS, 10 PARTIAL and 12 FAIL.
After correction, all 31 are PASS, with zero PARTIAL, FAIL or unresolved
ambiguities.

## Frozen configuration binding

`FrozenExperimentConfig` now binds and validates:

- candidate and strategy identity;
- actionable confidence threshold;
- SMA20/SMA50/SMA200, ATR14 and AvgVolume20 periods;
- stop/target and 20-session holding semantics;
- nine-symbol universe and all partition dates;
- sealed 2025 holdout;
- base and stress cost identities and assumptions;
- development, validation and OOS episode floors;
- matched-placebo floor;
- effective instrument-year block floor;
- instrument breadth floor;
- 40% instrument concentration ceiling;
- minimum three regime/calendar slices;
- 10,000 bootstrap replicates;
- 20-trading-session overlap window;
- execution authority `NONE`.

Material manifest drift fails closed. The outcome-free contract audit also
checks the exact registered falsification list, comparator coverage,
fail-closed promotion admission, old-run protection and holdout protection.

## Matched non-signal placebo

Matching is deterministic, one-to-one and uses only pre-outcome information:

- same instrument and calendar year;
- same signal-time SPY SMA200 regime;
- non-actionable candidate date;
- not the actual signal date;
- not within any active primary interval, inclusive of signal through exit;
- at least 20 future valid sessions;
- valid structural data;
- no more than 60 trading-session distance;
- nearest valid date first;
- SHA-256 tie-break for equal distances;
- no future return matching;
- identical frozen cost semantics and no cash-distribution credit;
- fixed-horizon placebo outcome with no transferred stop/target geometry.

Synthetic tests prove wrong-regime closer dates lose to farther valid dates,
actionable dates are rejected, active dates are rejected, matching is
one-to-one, and the distance boundary uses trading-session indices.

## Timestamp placebo

The registered deterministic procedure now produces 10,000 replicate sets. Each
replicate evaluates each available instrument/year independently and selects
from valid same-year dates using SHA-256 ordering over the frozen seed domain.
It enforces non-actionable, non-active, valid-provenance and future-horizon
constraints without relaxation. A deterministic digest of selections is
available for future result artifacts.

Synthetic tests prove replicate count, determinism, domain separation,
eligibility restrictions and independence from map iteration order.

## Secondary sign permutation

The implementation provides the registered 10,000-replicate,
instrument-year-stratified diagnostic. Mixed-direction strata are shuffled
deterministically while preserving direction labels within strata; the
registered add-one p-value is used. Single-direction data returns
`NOT_APPLICABLE` because the applicability condition is absent. This remains
secondary and cannot establish the primary long claim.

## Regime and theme slices

Regime is assigned from SPY close versus the available SPY SMA200 at the
observation timestamp, with no future information. The exact theme groups are
aggregated independently:

- `broad_equity`: SPY, QQQ, IWM, DIA
- `sector_equity`: XLK, XLF, XLE
- `rates_precious_metal`: TLT, GLD

Synthetic tests prove signal-time regime assignment and distinct group
aggregation rather than relabelling one aggregate result.

## Overlap and active intervals

Active intervals are represented as inclusive trading-session index ranges from
signal date through exit date. The registered overlap sensitivity retains the
earliest entry per instrument when a later entry is fewer than 20 trading
sessions away; a 20-session separation is retained. Calendar weekends and
holidays do not count, and instruments are independent.

## Benchmarks and diagnostics

The harness now has numeric helpers for:

- zero-return baseline;
- same-asset buy-and-hold over matching entry/exit dates;
- SPY buy-and-hold over matching dates;
- holding duration and gross/net diagnostic output.

These remain descriptive unless the frozen manifest makes them gates.
Top-five-percent exclusion retains removed episode IDs. Leave-one-instrument-
out produces one result per frozen instrument.

## Falsification and promotion order

Registered falsification outputs use the typed statuses:

`PASS`, `FAIL`, `INSUFFICIENT`, `INFORMATIONAL`, and `NOT_APPLICABLE`.

No threshold is invented for diagnostics whose preregistration defines no
binary threshold; those remain informational. Explicit failures block
promotion, and required insufficiency yields `INSUFFICIENT_EVIDENCE`.

The promotion classifier is now called only after the complete falsification
result is constructed. It requires the sample, dependence, concentration,
primary mean, bootstrap and paired-placebo gates in addition to the registered
falsification dispositions. Synthetic tests prove that an empty/insufficient
falsification result cannot promote, an explicit failure blocks promotion, and
informational robustness cannot rescue a failed primary gate.

## Outcome-free conformance audit

Command:

`go run ./cmd/val03-ma-crossover --contract-audit`

Result:

`VAL03_CONTRACT_AUDIT=PASS candidate=ma_crossover_v1 falsifications=14 no_outcomes_loaded=true no_2025=true no_2026=true`

The command reads only the frozen manifest and immutable integrity record. It
does not load bar payloads, calculate signals, generate episodes, calculate
returns, expose 2023–2024 results, or access 2025/2026 data.

## Test and safety boundary

Focused synthetic tests cover placebo eligibility and distance, timestamp
replicates, sign permutation, regime/theme aggregation, overlap sessions,
benchmarks, config mutation, promotion ordering, holdout protection and
result-output markers. Existing warm-up tests remain active.

Safety remains unchanged:

- `ALLOW_LIVE_TRADING=false`
- `BROKER_EXECUTION_ALLOWED=false`
- `EXECUTION_ENABLED=false`
- `ExecutionAuthority=NONE`
- `CreatesFill=false`
- maximum leverage `1x`

No broker calls, orders, fills, approvals, positions, paid data, hosted
inference, forward paper or Phase 13 activity occurred.

## Final governance state

- `VAL-03 = CONTAMINATED_FOR_FORMAL_OOS`
- `VAL-03R1 = PARTIAL CLOSURE`
- `VAL-03R1A = COMPLETE / EXTERNAL REVIEW REQUIRED`
- `NEW INDEPENDENT OOS = NOT AUTHORIZED`
- `2025 HOLDOUT = SEALED`
- `FORWARD PAPER = NOT STARTED`
- `PHASE 13 = NOT STARTED`

Manifest v1.5 was not changed. Strategy source and parameters were not
changed. No new historical evaluation is authorized by this package.
