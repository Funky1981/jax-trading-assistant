# Jax VAL-03 Formal OOS Integrity Review

## Status

VAL-03 execution is closed for external review as:

`CONTAMINATED_FOR_FORMAL_OOS`

This is an execution-integrity classification, not a claim that the strategy
failed or passed scientifically.

## Frozen boundary

- Candidate: `ma_crossover_v1`.
- Preregistration: v1.5, manifest SHA-256
  `96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`.
- Pre-OOS freeze: `Docs/validation/results/VAL-03-OOS-EXECUTION-FREEZE.json`.
- Frozen runner commit: `4ae446299418fde8bb5173b02bfe670da0266235`.
- Formal OOS boundary: `2023-01-01..2024-12-31`.
- Final holdout: `2025-01-01..2025-12-31`, sealed.

The pre-OOS freeze was pushed and passed its exact-SHA CI checkpoint before the
formal command was run.

## Execution record

The exact frozen command was run once:

```text
go run ./cmd/val03-ma-crossover
```

Run count: `1` attempted formal pipeline execution. No rerun occurred.

The first invocation stopped on a loader/schema defect before producing a
result. After the loader was corrected and the correction was committed and
green at the pre-OOS checkpoint, the single formal pipeline invocation
produced the raw result artifacts. No narrative interpretation was used to
change the runner after that invocation.

## Material integrity defect

The frozen primary operational gate is actionable confidence `>= 0.60`.
The formal partition calls in `cmd/val03-ma-crossover/main.go` passed a gate
value of `1.0` instead. The frozen robustness helper used `0.60`, so the
primary pipeline and its registered variants did not execute the same
candidate semantics.

Consequences visible in the raw artifacts include zero primary actionable-long
episodes in the primary partition outputs, while some robustness variants
were evaluated with the correct threshold. The raw `INSUFFICIENT_EVIDENCE`
classification is therefore not scientifically interpretable as a result for
the preregistered candidate.

The raw run manifest also records `DataQuality.performance_output_generated`
as false even though result artifacts were written. This is retained as an
artifact-schema defect and is not treated as evidence that no output existed.

## Required no-rerun disposition

Because formal OOS output existed before the defect was fully identified, the
post-freeze rule applies: a material bug discovered after OOS output becomes
available must not be fixed and rerun. The execution is therefore not eligible
for scientific promotion or rejection from its numeric outputs.

The correct terminal disposition is:

`CONTAMINATED_FOR_FORMAL_OOS`

The defect, raw artifacts and exact identities are preserved. Any future
attempt requires a separately authorized new frozen execution boundary; this
package does not authorize that attempt.

## Data and safety integrity

- No 2025 bars, signals, episodes, benchmark paths or holdout outputs were
  accessed.
- No broker, order, fill, position, approval or execution state was created.
- No paid provider or hosted inference call was made.
- The candidate and preregistered strategy parameters were not changed.
- No forward paper or Phase 13 activity started.

## Related machine-readable record

`Docs/validation/results/VAL-03-HYP-MA-001-INTEGRITY-REVIEW.json` records the
raw-versus-final classification, defect, no-rerun rule and exact frozen
identities without replacing the raw run artifacts.
