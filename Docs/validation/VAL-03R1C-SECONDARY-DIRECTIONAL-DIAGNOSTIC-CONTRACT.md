# VAL-03R1C Secondary Directional Diagnostic Contract

> Supersession note: R1D corrected the null-statistic implementation described
> below so null draws use `SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1` and never reuse a
> populated observed directional effect.

## Status

`VAL-03R1C = COMPLETE / EXTERNAL REVIEW REQUIRED`. This is a prospective,
outcome-free correction for the future recovery harness. The contaminated VAL-03
run remains immutable, no new OOS is authorized, and manifest v1.5 is unchanged.

## Policy

`SECONDARY_DIRECTIONAL_PRICE_EFFECT_V1` is a non-blocking diagnostic. It cannot
rescue or replace the primary long-only promotion claim. It is produced by the
same production evaluator that constructs frozen MA signals, after the existing
0.60 actionable-confidence gate, with valid history, next-session entry, and
structural validity.

For each instrument, the diagnostic has its own active interval. A qualifying
BUY or SELL signal is simulated at the next regular-session open using its
signal-time SMA50/ATR stop and three-ATR target, opening-cross handling,
conservative stop-first intrabar ordering, structural invalidation, and the
20-session maximum holding period. It creates no order, fill, position,
approval, or trading-state mutation. Pre-entry geometry or incomplete-data
failure records an abstention; later signals are suppressed only while the
diagnostic interval is active.

## Directional observation

The observation records instrument, calendar year, direction, signal/entry/exit
dates, directional effect, absolute magnitude, and its instrument-year stratum.
The raw underlying price effect is `raw_exit/raw_entry - 1`. BUY effect equals
that value; SELL effect equals its negative. No commission, borrow, financing, or
cash-distribution credit is applied. The observation is descriptive and is not a
tradable short claim.

## Mixed-stratum sign diagnostic

The secondary sign permutation uses
`SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1`, 10,000 replicates, and the existing
domain-separated deterministic seed. Under `MIXED_INSTRUMENT_YEAR_ONLY_V1`,
only instrument-year strata containing both BUY and SELL observations enter the
permutation. Single-direction strata are explicitly listed and excluded with
their observation counts. Each null value is the assigned deterministic sign
times the observation's fixed absolute magnitude under
`SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1`; it never reads the observed effect sign.
No label shuffling occurs. The add-one p-value is informational only; no mixed
stratum yields `NOT_APPLICABLE`.

## Auditability

The production evaluator persists the typed diagnostic collection into the
falsification builder, and the sign summary records all strata, mixed-stratum
IDs, excluded single-direction IDs, observations used/excluded, seed, algorithm,
and policy. Partition output persists exact calendar-year × SPY-regime cell
identities. Unknown falsification statuses fail closed as
`INSUFFICIENT_EVIDENCE`, including when marked non-blocking.

## Safety

`ExecutionAuthority=NONE`, `CreatesFill=false`, live trading remains disabled,
the contaminated run is not rerun, no market outcomes or 2025/2026 data are
accessed, and Phase 13 remains not started.
