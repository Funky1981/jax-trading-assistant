# Jax VAL-03R1B Falsification & Promotion Integrity Closure

> Current status note: the remaining audit-traceability defects identified
> after this package are closed by `VAL-03R1C`. This document preserves the R1B
> evidence and is not the final R1C handover.

## Status

`VAL-03R1B = COMPLETE / EXTERNAL REVIEW REQUIRED`.

This is an outcome-free corrective package. No historical performance was
executed, no market outcomes were inspected, and no 2025 or 2026 data was
accessed.

## Preserved identity

The selected candidate remains `ma_crossover_v1`. Manifest v1.5 remains
unchanged at SHA-256
`96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`.
The contaminated VAL-03 run remains immutable: formal run count `1`,
`no_rerun=true`, classification `CONTAMINATED_FOR_FORMAL_OOS`.

## Corrective closure

- Top-five removal is `max(1, ceil(0.05*N))`, with exact removed IDs retained.
- Top-five concentration is a blocking disposition. Remaining primary or
  paired effect failure produces `FAIL`; remaining pair evidence below the
  floor produces `INSUFFICIENT`.
- Secondary sign permutation receives an explicit typed observation collection.
  It uses independent deterministic +/- assignments, not BUY/SELL label
  shuffling, with 10,000 replicates and the registered add-one p-value.
- Breadth is the number of distinct non-empty
  `(calendar year, SPY SMA200 regime)` cells; `UNKNOWN` is excluded and the
  floor remains three.
- Promotion classification respects `Blocking`: blocking failures and
  insufficiency block; non-blocking informational diagnostics do not.
- Weekend and exchange-holiday session fixtures prove session-index overlap
  semantics, including the 19-session conflict and 20-session allowance.

## Contract audit

`go run ./cmd/val03-ma-crossover --contract-audit` passed with:

`VAL03_CONTRACT_AUDIT=PASS candidate=ma_crossover_v1 falsifications=14 no_outcomes_loaded=true no_2025=true no_2026=true`

The audit additionally verifies the R1B rule identities, explicit sign
observation wiring, blocking semantics, contaminated-run guard, holdout guard
and `ExecutionAuthority=NONE`.

## Explicit disposition rules

| Rule | Status | Effect |
|---|---|---|
| Required primary episode, paired-placebo, effective-block, instrument or breadth floor is below its bound | `INSUFFICIENT` | blocking; terminal `INSUFFICIENT_EVIDENCE` |
| Blocking matched-placebo diagnostic has insufficient pairs | `INSUFFICIENT` | blocking; terminal `INSUFFICIENT_EVIDENCE` |
| Blocking top-five remainder loses its primary or paired effect | `FAIL` | blocking; terminal `FAILED_VALIDATION` |
| Blocking top-five remainder has fewer than the pair floor | `INSUFFICIENT` | blocking; terminal `INSUFFICIENT_EVIDENCE` |
| Blocking calendar-year × regime breadth is below three cells | `INSUFFICIENT` | blocking; terminal `INSUFFICIENT_EVIDENCE` |
| Primary base mean or either registered bootstrap lower bound is non-positive | n/a | terminal `FAILED_VALIDATION` after adequate evidence |
| Non-blocking robustness or benchmark diagnostic | `INFORMATIONAL` | retained; does not block by status alone |
| Secondary sign permutation with no mixed-direction stratum | `NOT_APPLICABLE` | retained; does not block |
| Unknown/unsupported diagnostic status | n/a | fail closed as `INSUFFICIENT_EVIDENCE` |

The table is prospective for a future recovery experiment. It does not alter
the contaminated result or rewrite manifest v1.5. Earlier package wording that
treated top-five exclusion as informational is superseded by this R1B
clarification.

## Verification boundary

- `go test ./... -count=1`: PASS.
- `go vet ./...`: PASS.
- `git diff --check`: PASS.
- `go run ./cmd/val03-ma-crossover --preflight-only`: PASS.
- No development, validation or OOS outcomes were calculated.
- No OOS rerun was performed.
- No candidate reselection, strategy change or optimization occurred.

## Governance

- `VAL-03 = CONTAMINATED_FOR_FORMAL_OOS`.
- `VAL-03R1 = PARTIAL CLOSURE`.
- `VAL-03R1A = NOT ACCEPTED AS FINAL CLOSURE / SUPERSEDED BY R1B`.
- `VAL-03R1B = COMPLETE / EXTERNAL REVIEW REQUIRED`.
- `NEW INDEPENDENT OOS = NOT AUTHORIZED`.
- `2025 HOLDOUT = SEALED`.
- `FORWARD PAPER = NOT STARTED`.
- `PHASE 13 = NOT STARTED`.

Safety remains `ALLOW_LIVE_TRADING=false`,
`BROKER_EXECUTION_ALLOWED=false`, `EXECUTION_ENABLED=false`,
`ExecutionAuthority=NONE`, `CreatesFill=false`, maximum leverage `1x`.
