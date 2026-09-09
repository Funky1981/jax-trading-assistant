# Phase 12 Scientific Completion Handover

## Repository

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Final HEAD: `6413ab1cae4f934f218471e736d3781c1d40547b`
- Scientific artifact: private `scientific-results-v3/report.json`
- Scientific artifact SHA-256: `50f8622f8013708f0879c43c8eded4030f448fcfcd450d3c3b5ed3c3d4a7bac4`
- Parent dataset: `hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1`
- Parent manifest SHA-256: `db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d`
- Evidence dataset: `hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2`
- Evidence manifest SHA-256: `967dfcb18eff8b6a3f4ec39fedd3898537446a284dc2a868631926dc1f41408c`

## Roadmap state

- Phase 10: `COMPLETE / GO`.
- Phase 11: `COMPLETE / GO`.
- Phase 12: `COMPLETE / CONDITIONAL GO — SCIENTIFIC EVALUATION COMPLETE`.
- Phase 13: `NOT STARTED`.
- Actual forward-paper evidence: `0 DAYS / 0 ORDERS`.
- Scientific status: `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`.

## Hypothesis and frozen design

`HYP-EVENT-001A — SEC evidence-backed issuer-event reaction` was evaluated as
registered. The frozen contract is `jax.hyp-event-001a.direction/v1` with
prompt identity
`d0b09acf412eb73ff97fc2e9bd5f4fc609dc57d5bd1dbd17d57f986b158d4143`. The
classifier used only same-accession SEC evidence available at the acceptance
cutoff and returned `POSITIVE`, `NEGATIVE`, `NEUTRAL` or
`INSUFFICIENT_EVIDENCE`; later outcomes could not create labels. The event
family was non-amended Form 8-K, with entry at the next eligible regular US
session open strictly after SEC availability, primary horizon five trading
days, secondary diagnostics one and three days, and SPY as benchmark.

Partitions were frozen as development 2016–2021, validation 2022–2023, formal
OOS 2024, and final holdout 2025. The final holdout was not semantically read
or scored. The private parent/evidence identities above were verified before
the experiment.

## Direction assessment and hosted inference

GPT-5.6 Luna was used with `reasoning_effort=none`, one result per event and a
maximum of one bounded retry. The run produced 939 unique labels:

| Direction | Count |
|---|---:|
| POSITIVE | 483 |
| NEGATIVE | 162 |
| NEUTRAL | 259 |
| INSUFFICIENT_EVIDENCE | 35 |

The persisted successful-response usage was 12,721,590 input tokens including
99,989 cached tokens, 126,229 output tokens and 0 reasoning tokens. The
historical `$2.677034` is preserved as `ORIGINAL INTERNAL COST ESTIMATE`.
The later request-level recomputation is `$2.678539` for the 939 persisted
responses, but three bounded validation retries lack retained provider usage,
so the complete HYP-EVENT cost remains unresolved. See
`PHASE-12-SCIENTIFIC-CLOSURE-FORENSIC.md`. No Alpaca data, returns, SPY data,
credentials or 2025 semantics were sent to the provider.

## Experiment results

The immutable report records 645 valid observations: 391 development, 162
validation and 92 formal 2024 OOS; 294 rows were skipped for missing market
coverage or a window crossing the frozen year boundary. The deterministic
direction-only and evidence-quality-conditioned baselines were both evaluated
with `cost_phase11_v1`. The conditioned threshold was `anchor_count / 5`,
capped at 1, with threshold `0.8`.

The formal 2024 cost-adjusted descriptive means were:

| Candidate | Observations | Mean cost-adjusted signed return | Hit rate |
|---|---:|---:|---:|
| Direction-only | 92 | 0.00419212 | 55.43% |
| Evidence-conditioned | 59 | 0.00853903 | 59.32% |

The corresponding development means were `-0.00118495` and `-0.00373331`;
validation means were `0.00028249` and `-0.00189598`. These are descriptive
results only. The report explicitly makes no IID significance claim and
retains issuer-cluster, deduplication, cost-stress and other sensitivity
diagnostics.

The candidate was frozen before formal OOS as experiment
`exp_c08c8bf01dc4c29b0f2c657057ab0dc12e9736d551d7a1f215ccf8dcfea9d258`.
The report retains 22 falsification records across the 11 registered tests and
both development/validation partitions, including label/date shuffles,
placebo proxy, evidence permutation, source removal, issuer/event clustering,
liquidity sensitivity, cost stress, regime split and top-issuer exclusion.

## Scientific decision

The 2024 conditioned comparison is retained as a
`PROMISING RESEARCH CANDIDATE — NOT PROMOTED`. This is not a claim of
profitability, statistical edge, production readiness or live readiness.
Promotion is deterministically `PROMOTION_CLOSED` because the 2025 holdout is
sealed, the accepted dataset remains current-ticker/survivorship limited, and
actual forward-paper evidence is zero. No advanced model was trained, no
recommendation logic changed, and no paper/live authority changed.

## Verification and adversarial review

The corrected harness enforces UTC session-open/close timestamps, exact parent
and evidence identities, non-holdout labels, raw-only traversal of sealed
post-2024 market rows, one-way OOS artifact creation and the complete
registered falsification plan. Focused classifier, experiment, hypevidence and
advancedquant tests pass. At final HEAD `ccf3944`, `go test ./... -count=1`,
`go vet ./...`, focused golden/replay/contract tests, `git diff --check` and
192-entry manifest validation pass. Fresh adversarial review found zero
blocking findings: `Blocking findings remaining: 0`; `Adversarial scientific
review: PASS`.

The accepted Phase-10/11 race result remains the existing Docker verification:
`CGO_ENABLED=1 go test -race ./internal/modules/workflow
./internal/modules/papertrading -count=1` passed. Native Windows cgo remains
unavailable; this is not a claim of native-host race verification.

## Safety, spend and limitations

- `ALLOW_LIVE_TRADING=false`.
- `BROKER_EXECUTION_ALLOWED=false`.
- Live worker disabled; maximum leverage remains `1x`.
- No approval, order, trade, fill, portfolio mutation or paper-policy change
  was created by the experiment.
- No new credentials or paid data were introduced.
- Raw SEC/Alpaca data and credentials remain private and outside Git.
- Current-ticker survivorship, inactive-symbol/delisted coverage and corporate
  action limitations remain.
- The 2025 holdout remains sealed and is available only for integrity checks.

## Recommended external decision

`CONDITIONAL GO PHASE 12`

The phase capability is complete for review, but the candidate remains
research-only and promotion remains closed. Phase 13 is not started.
