# VAL-03R4B — Single Recovery Formal OOS Execution

## Status

`VAL-03R4B = COMPLETE / EXTERNAL REVIEW REQUIRED`.

The one authorized recovery execution completed exactly once. Its terminal
classification is `FAILED_VALIDATION`. The result is not a forward-paper
admission and does not authorize Phase 13.

## Repository

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Authorization checkpoint: `7891cbbbcda50d06d03a399c8b6bda53a2647eca`
- Result-recording commit: recorded after this document and generated
  artifacts are committed

## Starting state

The package started at `c95f14306bd0a67cd7a913aad1e0cfb92aa31128`, on
`capability-reset`, with a clean worktree, local HEAD equal to
`origin/capability-reset`, and divergence `0 0`. The R4A1 freeze was the
expected predecessor.

## External authorization

The external decision authorized exactly one `VAL-03R4B` recovery OOS run for
`ma_crossover_v1` over `2025-01-01..2026-09-11`, bound to the R4A1 execution
freeze SHA
`7f1ca6532c5ab6e6fab30aed3a53179eb236ef4ed1b294a7aed40c0275f5e830`.

Authorization artifact:
`VAL-03R4B-EXECUTION-AUTHORIZATION.json`.

- Authorization value: `GO_VAL03R4B_SINGLE_RECOVERY_OOS_EXECUTION`
- Authorization SHA-256: `620962836124c54992c21286fc133572c53197f84a825c7b9e4ccaac037e6114`
- `execute_once`: `true`
- `performance_run_count_before`: `0`

## Historical pre-start attempt

The earlier R4 attempt remains `PRE-START ABORTED` and is historical. It did
not consume the recovery execution. Historical pre-start invocations: `1`.
The contaminated original 2023–2024 formal OOS remains immutable and was not
rerun.

## R4A1 freeze verification

The R4A1 source identities and freeze were verified before authorization and
again before execution. The final pre-execution contract audit and complete
structural preflight both passed. The preflight reported nine instruments,
2,688 synchronized sessions per instrument, zero canonical-factor failures,
zero SPLIT/SPLIT+SPIN-OFF differences, and zero out-of-range rows.

## Initial structural preflight

The initial and final pre-`STARTED_ONCE` structural preflights used the shared
canonical loader. No code or frozen scientific term was changed after the
execution boundary.

## R4B authorization checkpoint SHA and CI

The permission-only authorization was committed and pushed as
`7891cbbbcda50d06d03a399c8b6bda53a2647eca` before execution. Exact-SHA CI was
green: Go SUCCESS, Frontend SUCCESS, Golden Tests SUCCESS, Import Boundary
SUCCESS; Integration was intentionally skipped under existing workflow policy.

## Final pre-execution structural preflight

Final contract audit: PASS.

Final full structural preflight: PASS.

The R4A1 freeze, candidate, threshold, source identities, provider identity,
cost identities, boundary, and safety settings matched exactly.

## R4B execute invocation count

`go run ./cmd/val03r-recovery-oos --execute` invocations: `1`.

No retry, rerun, or post-result execution occurred.

## One-shot lifecycle

The run-state transitioned from the durable `STARTED_ONCE` marker to
`COMPLETED_ONCE`:

```json
{
  "performance_run_count": 1,
  "status": "COMPLETED_ONCE",
  "candidate": "ma_crossover_v1",
  "recovery_boundary": "2025-01-01..2026-09-11",
  "execution_freeze_sha256": "7f1ca6532c5ab6e6fab30aed3a53179eb236ef4ed1b294a7aed40c0275f5e830"
}
```

The result lifecycle is `performance_execution_started=true`,
`performance_artifacts_written=true`, `performance_run_count=1`, and
`recovery_oos_status=EXECUTED_ONCE`.

## Dataset integrity

The run used the accepted hash-bound recovery dataset identity
`dd4c49313016bcb45912142620bf8dcd965ec80eaf8b5be7376b3f36d98651d0`.
Alpaca SIP 1Day data covered the frozen nine-instrument universe through
2026-09-11. Each instrument had 424 RAW, SPLIT, and SPLIT+SPIN-OFF sessions;
timestamps were ordered, OHLC/volume quality checks passed, and no out-of-range
rows were present. No data was reacquired during R4B.

## Recovery structural boundaries

The recorded structural boundaries were `XLE 2025-12-05` and
`XLK 2025-12-05`. The frozen post-boundary warm-up and split-spanning
invalidation rules remained active. No SPLIT versus SPLIT+SPIN-OFF difference
or unexplained factor anomaly was reported.

## Primary sample

- Signal observations: `3,432`
- Eligible actionable long episodes: `108`
- Matched placebo pairs: `80`
- Non-empty instrument-year blocks: `16`
- Instruments: `9`
- Calendar/regime cells: `3`
- Maximum instrument share: `22.22%`
- Frozen primary population: eligible non-overlapping actionable long episodes

The sample, block, instrument, breadth, and concentration floors passed. The
paired-placebo floor also passed (`80 >= 30`).

## Abstentions

The retained abstention counts were:

- HOLD: `1,541`
- POST_SPLIT_WARMUP: `384`
- active-position suppression: `1,129`
- below-confidence: `403`
- no matched placebo: `28`
- structural_action_invalidated: `17`

Bearish diagnostic observations: `234`.

## Base-cost result

Primary model: `cost_val03_ibkr_fixed_personal_base_v1`.

- Mean gross return per eligible long episode: `1.008262%`
- Mean net return after base costs: `0.806967%`
- Mean stress-cost net return: `0.194035%`

The positive base mean is not sufficient for promotion.

## Bootstrap result

The registered instrument-year block bootstrap used 10,000 replicates and the
frozen deterministic seed contract.

- Mean net return 95% interval: `[0.111356%, 1.472851%]`
- Paired actual-minus-placebo mean: `-1.002592%`
- Paired 95% interval: `[-2.182375%, 0.091127%]`

The primary mean lower bound was positive, but the paired edge lower bound was
not positive.

## Matched placebo result

The matched non-signal placebo produced `80` valid pairs. The mean actual minus
placebo difference was `-1.002592%`, with bootstrap lower bound `-2.182375%`.
This fails the preregistered direction-symmetric comparative gate.

## Timestamp placebo result

The deterministic constrained timestamp placebo completed with 10,000
replicates and 160,000 selections. It used seed SHA-256
`37114c390002d7f19e2fabb24dd4957a53e5c365f91afef2632025525acad3ea` and
selection digest
`d7055fea1675d5e804f56da9a9292abcc68c97c59a8a03d9b935dc5d261cc79d`.
It was retained as informational and was not used to replace the primary
comparative null.

## Stress-cost result

The legacy Phase-07 fixture
`cost_4c9e2fbffb41d05b537eadf05b142c8f8d00ed46d672d75bca5f83a4269404f8` was
used only as the registered stress scenario. Its mean net return was
`0.194035%`. It did not replace the base model.

## Top-5% exclusion

The frozen `CEIL_5_PERCENT_V1` rule removed six highest-return episodes:
`GLD-2026-01-30`, `XLK-2025-10-13`, `GLD-2025-04-14`, `GLD-2025-11-13`,
`IWM-2026-05-20`, and `GLD-2026-01-27`.

- Mean before: `0.806967%`
- Mean after: `0.445822%`
- Remaining episodes: `102`
- Remaining pairs: `77`
- Remaining paired difference: `-1.362795%`
- Disposition: `FAIL` / blocking

## Leave-one-instrument-out

All leave-one-instrument-out means remained positive, ranging from `0.312315%`
when GLD was excluded to `0.947603%` when QQQ was excluded. These are
descriptive robustness outputs only and do not override the failed paired gate
or top-5% exclusion.

## Regime slices

- `SPY_ABOVE_SMA200`: `103` episodes, mean net `0.770600%`, contribution
  `79.3718%`
- `SPY_BELOW_SMA200`: `5` episodes, mean net `1.556140%`, contribution
  `7.7807%`

The registered regime diagnostic completed. The small below-regime slice is
reported descriptively, not treated as independently significant.

## Theme slices

- broad_equity: `61` episodes, mean net `0.438747%`, contribution `26.7636%`
- sector_equity: `20` episodes, mean net `0.170185%`, contribution `3.4037%`
- rates_precious_metal: `27` episodes, mean net `2.110563%`, contribution
  `56.9852%`

The result is materially concentrated in the rates/precious-metal theme for
contribution purposes, although the frozen instrument concentration ceiling
passed.

## SMA perturbations

- 19/49/199: `109` episodes, mean net `0.774454%`
- 21/51/201: `103` episodes, mean net `0.581518%`

Both remained falsification variants and were not eligible replacements.

## ATR perturbations

- 0.90 x ATR14: `113` episodes, mean net `0.638921%`
- 1.10 x ATR14: `100` episodes, mean net `0.715612%`

Both remained falsification variants and were not eligible replacements.

## Overlap sensitivity

The frozen earliest-signal 20-session overlap sensitivity produced `70`
episodes with mean net return `0.417884%`. It was informational.

## Secondary bearish diagnostics

Bearish observations remained secondary directional diagnostics only. The
secondary sign-permutation diagnostic used 10,000 assignments, with `82`
observations used, `54` excluded single-direction observations, `10` mixed
strata, and p-value `0.857814`. No short capability was claimed.

## Concentration

The instrument concentration gate passed: maximum instrument contribution was
`22.2222%`, below the frozen `40%` ceiling. However, the top-five exclusion
failure and the theme contribution profile show that the positive mean is not
robust enough for promotion.

## Full falsification matrix

| Diagnostic | Status | Blocking | Result |
| --- | --- | --- | --- |
| Matched non-signal placebo | PASS as executed | Yes | Comparative edge was negative and failed the primary gate |
| Timestamp placebo | INFORMATIONAL | No | 10,000 deterministic replicates completed |
| Secondary sign permutation | INFORMATIONAL | No | p-value `0.857814` |
| Top-5% exclusion | FAIL | Yes | Paired effect remained negative |
| Leave-one-instrument-out | INFORMATIONAL | No | All descriptive means positive |
| Regime slices | PASS as executed | Yes | Descriptive slices retained |
| Theme slices | INFORMATIONAL | No | Rates/precious-metal contribution dominated |
| SMA 19/49/199 | INFORMATIONAL | No | Frozen variant completed |
| SMA 21/51/201 | INFORMATIONAL | No | Frozen variant completed |
| ATR 0.90 | INFORMATIONAL | No | Frozen variant completed |
| ATR 1.10 | INFORMATIONAL | No | Frozen variant completed |
| Legacy stress costs | INFORMATIONAL | No | Mean net `0.194035%` |
| 20-session overlap | INFORMATIONAL | No | Mean net `0.417884%` |
| Zero/buy-hold baselines | INFORMATIONAL | No | Descriptive baseline output retained |

## Terminal classification

`FAILED_VALIDATION`.

The sample and primary base-cost lower-bound gates passed, but the
direction-symmetric matched-placebo gate failed and the blocking top-5%
exclusion failed. The positive mean alone therefore does not support
`FORWARD_PAPER_ELIGIBLE`.

## Scientific interpretation

The recovery result is evidence about the frozen recovery sample only. It does
not validate the contaminated 2023–2024 run, does not reopen 2025 as a future
holdout, and does not demonstrate a trading edge. The observed positive
base-cost mean was not superior to the preregistered matched non-signal
explanation, and the result was sensitive to removal of a small set of extreme
contributors. No strategy, parameter, cost, placebo, bootstrap, or boundary
change was made after observing the result.

## Result artifact hashes

- `VAL-03R4B-EXECUTION-AUTHORIZATION.json`:
  `620962836124c54992c21286fc133572c53197f84a825c7b9e4ccaac037e6114`
- `VAL-03R4-RUN-STATE.json`:
  `3c6badcf508938770560f4e02c3ed634135ddffbec9a7af92d6533ed05773e60`
- `VAL-03R4-PRIMARY.json`:
  `ec30fdd097e096de1bb18b4b6db4799322ddf60295a24d362863407d16b80ab7`
- `VAL-03R4-FALSIFICATION.json`:
  `3af5b36f11f942cc2d5f273a08afd17588ba9dd61ee87d62c4c3602daeb9d325`
- `VAL-03R4-RUN-MANIFEST.json`:
  `0f74a933b94768e4d8786fb74c20c0bd637d7cbcef2172a42bab2b8dde320aec`

The generated artifacts bind parent manifest SHA
`96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a`, dataset
SHA `dd4c49313016bcb45912142620bf8dcd965ec80eaf8b5be7376b3f36d98651d0`, and
R4A1 execution freeze SHA
`7f1ca6532c5ab6e6fab30aed3a53179eb236ef4ed1b294a7aed40c0275f5e830`.

## Files changed

- `Docs/validation/results/VAL-03R4-RUN-STATE.json`
- `Docs/validation/results/VAL-03R4-PRIMARY.json`
- `Docs/validation/results/VAL-03R4-FALSIFICATION.json`
- `Docs/validation/results/VAL-03R4-RUN-MANIFEST.json`
- this handover
- current roadmap and decision log status entries

## Verification

Post-run verification is required on the result-recording commit:

- result JSON syntax and schema references: PASS
- artifact SHA-256 recomputation: PASS
- no 2023–2024 rerun: PASS
- no second R4B execution: PASS
- no broker calls, orders, fills, or positions: PASS
- forward paper: NOT STARTED
- Phase 13: NOT STARTED

## Safety

`ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`,
`EXECUTION_ENABLED=false`, `ExecutionAuthority=NONE`, `CreatesFill=false`,
execution worker disabled, and maximum leverage `1x` were preserved. No broker
or account mutation occurred.

## Adversarial review

- Selected candidate remained `ma_crossover_v1`; no reselection.
- R4B execute invocation count was exactly `1`; no rerun after inspection.
- The original contaminated 2023–2024 run was not rerun.
- Recovery boundary remained `2025-01-01..2026-09-11`; no later row entered.
- Frozen threshold `0.60`, parameters, costs, placebo, bootstrap and sample
  floors were unchanged.
- Bearish diagnostics did not rescue the long-only claim.
- The failed paired-placebo and top-five exclusion results were retained.
- The positive primary mean was not treated as proof of edge.
- Forward paper and Phase 13 remained stopped.

`Adversarial VAL-03R4B review = PASS`.

## Exact VAL-03R4B status

`VAL-03R4B = COMPLETE / FAILED_VALIDATION / EXTERNAL REVIEW REQUIRED`.

## Recommended external decision

`NO-GO FORWARD PAPER / FAILED HISTORICAL VALIDATION`
