# Jax VAL-03C Warm-Up & Manifest Consistency Amendment

## Status

VAL-03C is a pre-outcome warm-up and manifest-consistency closure. It does not
authorize VAL-03 performance execution.

- VAL-03B adjusted-bar engineering contract: ACCEPTED.
- VAL-03B data readiness: blocked only by the obsolete standalone 2015 warm-up
  requirement.
- VAL-03C: complete / external review required.
- VAL-03 performance: NOT STARTED.
- OOS run count: 0.
- `ma_crossover_v1`: unchanged and not reselected.

## Amendment reason

The VAL-03B preflight returned complete synchronized RAW, SPLIT and
SPLIT+SPIN-OFF bar families beginning on the first 2016 market session,
2016-01-04, through 2024-12-31. It did not return a 2015 session. The 2015
calendar year was only intended to initialize the frozen SMA200 lookback, not
to provide a scored research population.

The amendment replaces the artificial calendar-year warm-up with a deterministic
per-instrument valid-session initialization contract. The request boundary is
2016-01-01, the provider's first expected session is 2016-01-04, and the first
development-scored observation is the earliest session after 200 valid
synchronized SPLIT-adjusted sessions are available.

No calendar validation or formal OOS boundary moved.

## Terms explicitly unchanged

- Candidate and strategy parameters.
- SMA20, SMA50, SMA200 and ATR14.
- 0.60 actionable-confidence gate and pullback exclusion.
- Long-only primary claim and bearish secondary diagnostic treatment.
- Fixed nine-ETF universe.
- Next-session-open entry, stop/target, first target and 20-session exit.
- USD 10,000 reference capital and whole-share sizing.
- Base and stress cost models.
- Matched-placebo design, bootstrap, sample floors and concentration limits.
- Validation: 2021-01-01 through 2022-12-31.
- Formal OOS: 2023-01-01 through 2024-12-31.
- Final holdout: 2025-01-01 through 2025-12-31, SEALED.

## Pre-outcome boundary

Before this amendment:

- OOS run count: 0.
- Strategy signals calculated: NO.
- Candidate episodes generated: NO.
- Development profitability viewed: NO.
- Validation profitability viewed: NO.
- OOS profitability viewed: NO.
- Historical returns calculated: NO.
- 2025 data accessed: NO.

The change was caused solely by provider start-date feasibility and manifest
semantic consistency. It was not selected from performance, profitability,
sample counts, or any result artifact.

## v1.5 contract consequences

The manifest now binds:

- `requested_data_start = 2016-01-01`;
- `provider_data_start = 2016-01-04`;
- per-instrument valid-session initialization;
- 200 required sessions;
- `WARMUP_ONLY` before initialization completion;
- development scoring only after initialization completion;
- SPLIT-adjusted indicator inputs;
- RAW execution-reference role;
- no primary cash-distribution credit;
- validation data-contract source identities;
- the deterministic initialization helper identity.

The readiness helper only evaluates session validity and initialization state.
It cannot calculate MA direction, confidence, signals, episodes, returns or
performance.

## Readiness evidence

The revalidated bar-family evidence contains 2,264 synchronized sessions per
instrument for RAW, SPLIT and SPLIT+SPIN-OFF, from 2016-01-04 through
2024-12-31. Every instrument reaches initialization on 2016-10-17, the 200th
valid session, leaving 199 WARMUP_ONLY sessions. The machine-readable
readiness artifact is
`Docs/validation/results/VAL-03C-DATASET-READINESS.json` with SHA-256
`b7081deb3bf55be5a23c1f1fcafbe44ff90c4fcee924de47f738cdcad86861ce`.

The bar-family payload hashes and per-family source mapping remain bound by
the hash-verified VAL-03B evidence artifact referenced by that readiness
artifact. No signals, episodes, returns or performance outputs are present.

## Result

The existing bar-family evidence is sufficient under the corrected warm-up
contract, subject to external authorization before VAL-03 execution. No
performance runner was executed and no market outcome was produced.
