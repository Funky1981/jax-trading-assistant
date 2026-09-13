# VAL-03B v1.4 Pre-Outcome Amendment

## Amendment identity

- Candidate: `ma_crossover_v1`
- Prior contract: `v1.3`
- Amended contract: `v1.4`
- OOS run count before amendment: `0`
- Amendment timing: before any signal, episode, return or OOS calculation

## Reason

VAL-03A established that Alpaca corporate-action event history could not prove
the complete point-in-time ledger required by v1.3. The v1.3 raw-bar contract
was therefore not executable under the authorized provider and zero-new-spend
boundary.

The amendment replaces that unexecutable event-ledger dependency with a
conservative synchronized-bar contract:

- SPLIT-adjusted bars generate signal and execution geometry;
- RAW bars provide execution-reference scale and whole-share cost arithmetic;
- SPLIT+SPIN-OFF bars detect unsupported structural differences;
- primary returns exclude cash-distribution credit.

This tests a price edge, not a total-return edge. It does not claim the two
economic definitions are identical.

## Preserved terms

The following were not changed:

- selected candidate;
- strategy parameters;
- confidence gate and pullback exclusion;
- costs;
- benchmark/placebo design;
- sample floors;
- partitions;
- robustness suite;
- long-only primary claim;
- 2025 seal.

## Preflight outcome

All nine ETFs returned synchronized RAW, SPLIT and SPLIT+SPIN-OFF family data
through 2024-12-31 with no 2025 rows and no spin-off differences. However, all
families began at 2016-01-04 rather than the required 2015-01-01 warm-up.

No provider fallback is authorized. VAL-03 remains blocked until that coverage
requirement is satisfied or an external decision changes the data boundary.

No strategy calculation was performed.

