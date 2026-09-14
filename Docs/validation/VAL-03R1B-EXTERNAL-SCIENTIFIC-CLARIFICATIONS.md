# VAL-03R1B External Scientific Clarifications

## Status

These are prospective clarifications for a future independent recovery
experiment. They do not repair, reinterpret or rerun the contaminated VAL-03
experiment. Manifest v1.5 remains unchanged.

## Calendar/regime breadth

The breadth unit is `CALENDAR_YEAR_X_SPY_SMA200_REGIME_V1`: one distinct,
non-empty `(calendar year, SPY SMA200 regime)` cell. `UNKNOWN` is excluded.
The minimum is three cells. Duplicate observations in one cell do not increase
breadth.

## Top-five concentration

For `N > 0`, remove `max(1, ceil(0.05 * N))` highest-base-net-return primary
episodes and retain all removed IDs. This is a blocking concentration
falsification. The remainder fails if its mean primary net return or, when at
least the registered pair floor remains, its paired actual-minus-placebo effect
is non-positive. If remaining matched evidence is below the pair floor, the
disposition is `INSUFFICIENT`.

## Secondary sign diagnostic

The secondary mixed-direction diagnostic uses typed observations and 10,000
domain-separated SHA-256 counter-derived independent `+/-` assignments. It
does not shuffle observed BUY/SELL labels. The add-one p-value is retained and
the result is `INFORMATIONAL` unless a future preregistration supplies a
binary threshold. A single-direction-only stratum is `NOT_APPLICABLE`; this
diagnostic cannot rescue the primary long-only claim.

## Promotion blocking

Only dispositions with `blocking=true` can produce a promotion failure or
insufficiency. Blocking `FAIL` maps to `FAILED_VALIDATION`; blocking
`INSUFFICIENT` maps to `INSUFFICIENT_EVIDENCE`. Non-blocking informational or
not-applicable diagnostics do not block by status alone.

## Historical boundary

The trading-session overlap rule is tested using deterministic weekday and
exchange-holiday fixtures. Calendar days that are not sessions do not advance
the overlap distance. Instruments remain independent.

## Preserved history and safety

- `VAL-03 = CONTAMINATED_FOR_FORMAL_OOS`.
- Formal historical OOS run count remains `1`; `no_rerun=true` remains active.
- Contaminated artifacts and v1.5 manifest were not modified.
- No outcome, 2025/2026 data, broker action, paper activity or Phase 13 work
  occurred.
