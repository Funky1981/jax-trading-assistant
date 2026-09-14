# Jax VAL-03R1 Harness Integrity Recovery

## Status

`VAL-03R1 = COMPLETE / EXTERNAL REVIEW REQUIRED`

This package repairs and tests the VAL-03 harness only. It does not authorize
or execute a new historical evaluation.

## Preserved contaminated execution

The original VAL-03 result remains immutable:

- formal OOS run count: `1`;
- final classification: `CONTAMINATED_FOR_FORMAL_OOS`;
- no-rerun: `true`;
- raw result artifacts and their recorded identities are preserved.

The original defect was a formal evaluator call using confidence gate `1.0`
while the frozen manifest required `0.60`. The raw zero-episode result is not
interpreted as strategy failure or success.

## Harness correction

`cmd/val03-ma-crossover` now loads `FrozenExperimentConfig` from the hash-bound
v1.5 preregistration. It validates candidate identity, strategy version,
confidence threshold, indicator periods, exit parameters, universe,
partitions, holdout, cost identities, sample floors, bootstrap count and
execution authority before any outcome-bearing path can run.

Primary development, validation and formal-OOS paths all consume the same
typed actionable-confidence threshold. Registered robustness variants may
change only their explicitly registered SMA/ATR parameters; they cannot
silently change eligibility.

The old contaminated execution identity is fail-closed through the immutable
integrity record. A new outcome-bearing execution identity is not created by
VAL-03R1.

## Preflight-only path

Command:

`go run ./cmd/val03-ma-crossover --preflight-only`

It validates manifest identity/configuration and the contaminated-run guard,
then reports `outcomes_calculated=false`. It does not load market bars or
calculate signals, episodes, returns or OOS results.

## Result-schema correction

Future result manifests now distinguish data-only output from an output-bearing
run. Data-only quality artifacts use `performance_output_generated=false`; a
run manifest written after result output uses `true`. The historical
contaminated artifact is not rewritten.

## Test coverage

Deterministic tests cover manifest/config drift, shared threshold binding,
date and holdout guards, contaminated-run refusal, entry/exit geometry,
whole-share sizing, cost arithmetic, placebo/seed determinism, bootstrap
determinism, block/concentration accounting and the output marker. Existing
VAL-03C initialization tests continue to cover the 200-session warm-up,
invalid-session handling and structural reset.

## Scientific and safety boundaries

- Strategy source and v1.5 manifest were not changed.
- No historical outcome was rerun.
- No development, validation or OOS outcome was newly calculated by VAL-03R1.
- No 2025 or 2026 data was requested or accessed.
- No provider data was newly acquired.
- `ExecutionAuthority=NONE`; `CreatesFill=false`.
- Forward paper and Phase 13 remain not started.

## Recovery decision

Any future independent performance boundary requires external technical-lead
authorization and a new preregistration/execution identity. See
`Docs/validation/VAL-03R1-INDEPENDENT-RECOVERY-OPTIONS.md`.
