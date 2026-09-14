# Jax VAL-03R1D Sign-Permutation Null Integrity Closure

## Status

`VAL-03R1D = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This is a narrow, outcome-free correction. The first VAL-03 run remains
`CONTAMINATED_FOR_FORMAL_OOS`; it was not rerun. No independent OOS is
authorized, no market outcomes were loaded, and manifest v1.5 remains unchanged.

## Defect and correction

R1C correctly wired production BUY/SELL diagnostic observations, but the first
null implementation changed `Direction` while `signedMean` preferred the already
populated `DirectionalEffect`. Null draws could therefore reuse the observed
effect. R1D removes that ambiguity with separate functions:

- observed statistic: `mean(DirectionalEffect)`;
- null statistic: `mean(assigned_sign * AbsoluteMagnitude)`.

`SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1` is the frozen null policy. The original
directional effect sign is never read by null-statistic calculation.

## Determinism and traceability

Signs use `SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1`, with the existing
domain-separated seed and 10,000 replicates. For each canonical
instrument-year/index assignment, the first bit of the raw SHA-256 digest is
used: zero means `+1`, one means `-1`. The null-statistic sequence is hashed
with replicate number and IEEE-754 float bits and persisted as a digest.

Observations carry the canonical `instrument-year` `Stratum` field. A stored
non-empty stratum that disagrees with the derived identity fails closed. Only
mixed strata under `MIXED_INSTRUMENT_YEAR_ONLY_V1` are applicable; single-
direction strata remain excluded and explicitly reported.

## Evidence

Synthetic tests prove exact reference signs, null values differing from the
observed statistic, non-degenerate p-values, repeated-call and input-order
determinism, excluded-stratum invariance, applicable-magnitude sensitivity,
stratum mismatch rejection, production BUY/SELL observation fields, and
production evaluator-to-builder wiring. R1B/R1C regression tests remain green.

## Preserved boundaries

- Manifest v1.5 SHA-256 remains
  `96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`.
- Contaminated integrity artifact is unchanged; formal OOS run count is `1` and
  `no_rerun=true`.
- No 2023–2024 rerun, 2025/2026 access, paid data, hosted inference, broker
  action, paper activity, or Phase 13 work occurred.
- Safety remains `ExecutionAuthority=NONE`, `CreatesFill=false`, live trading
  disabled, and maximum leverage `1x`.

## Governance

- `VAL-03R1 = PARTIAL CLOSURE`.
- `VAL-03R1A = SUPERSEDED`.
- `VAL-03R1B = SUPERSEDED`.
- `VAL-03R1C = SUPERSEDED BY R1D`.
- `VAL-03R1D = COMPLETE / EXTERNAL REVIEW REQUIRED`.
- `NEW INDEPENDENT OOS = NOT AUTHORIZED`.
- `2025 HOLDOUT = SEALED`.
- `FORWARD PAPER = NOT STARTED`.
- `PHASE 13 = NOT STARTED`.
