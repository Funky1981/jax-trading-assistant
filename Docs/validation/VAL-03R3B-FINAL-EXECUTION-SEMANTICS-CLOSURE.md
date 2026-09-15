# VAL-03R3B Final Execution Semantics & One-Shot Integrity Closure

## Status

`VAL-03R3B = SUPERSEDED BY VAL-03R3C FOR STRUCTURAL RESET AND FREEZE
COMPLETENESS`.
This historical closure remains an outcome-free record. The corrected current
closure is `VAL-03R3C`; `VAL-03R4` is not authorized.

## Accepted starting state

- `VAL-03 = CONTAMINATED_FOR_FORMAL_OOS`; the original 2023-2024 run is never rerun.
- `VAL-03R3` readiness and dataset checks are accepted; R3A is superseded for the
  remaining execution defects.
- Candidate remains `ma_crossover_v1`, threshold remains `0.60`, and all strategy,
  cost, placebo, bootstrap, sample-floor and recovery-boundary terms are unchanged.
- Hash-bound local evidence remains under `.runtime/val03b/raw` and
  `.runtime/val03r3/raw`; no evidence was reacquired.

## Closure changes

The runner now applies post-split warm-up before signal/variant/placebo eligibility,
and uses the stress contract's fixed `$0.50` per order plus `10 bps` per leg on both
legs. Formal bootstrap draws are derived independently from SHA-256 counter inputs
using the parent v1.5 manifest identity. Timestamp placebo ranking uses the complete
SHA-256 domain seed rather than a truncated integer seed.

Before any future authorized execution, both hash-bound payload inventories are
revalidated immediately before data load. Runtime loading also checks exact family
synchronization, chronological sessions, positive/finite OHLCV, RAW/SPLIT scale
consistency, and SPLIT/SPLIT+SPIN-OFF identity consistency.

## One-shot lifecycle

An exclusive `jax.val-03r4.recovery-run-state/v2` `STARTED_ONCE` marker is created
after authorization, payload verification and structural validation, but before
evaluation. Any later error leaves that marker in place. Only after all result
artifacts are written is the state atomically replaced with `COMPLETED_ONCE`.
Malformed, started or completed state blocks another run. No authorization or run
state exists in this R3B package.

## Result and provenance contract

Future output uses `jax.val-03r4.recovery-oos-results/v1` and explicitly carries the
R2, R3, R3A, dataset and execution-freeze identities, parent manifest seed identity,
recovery range, reclassified former 2025 holdout state, post-boundary row count and
runner source identities. The legacy holdout field is omitted from JSON output.
Recovery quality summaries are range-filtered and retain the hash-bound warm-up
evidence identity; they contain no R3B performance values.

## Safety and outcome boundary

`ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`,
`EXECUTION_ENABLED=false`, `ExecutionAuthority=NONE`, `CreatesFill=false`, and
maximum leverage `1x` remain unchanged. No recovery performance, signals, episodes,
returns, P&L, 2025/2026 outcomes, forward paper or Phase 13 activity occurred.

The superseding machine-readable freeze is
`Docs/validation/results/VAL-03R3B-RECOVERY-OOS-EXECUTION-FREEZE.json` with status
`FROZEN_FOR_VAL03R4_EXTERNAL_AUTHORIZATION`. It is a future execution contract, not
an authorization.
