# VAL-03R1A Manifest Conformance Matrix

Status: baseline recorded before corrective implementation. This matrix is
outcome-free; it does not load market bars or inspect VAL-03 results.

| Frozen requirement | Manifest/source | Intended semantics | Implementation / test | Baseline | Corrective action |
|---|---|---|---|---|---|
| Candidate identity/version | `selected_candidate` | `ma_crossover_v1` only | `loadFrozenExperimentConfig` | PASS | retain |
| Confidence gate | `confidence_contract.actionable_threshold` | exact `0.60`, shared by all paths | `actionableByConfig` | PASS | retain |
| Indicator calculations | `indicator_contract` | frozen SMA/ATR/volume formulas | `buildSignal`, indicator tests | PARTIAL | bind and test all inputs |
| Entry/exit semantics | `protocol`, `exit_contract` | next session, fixed levels, 20 sessions | `simulateLong` | PARTIAL | add boundary tests and config binding |
| Non-overlap | `protocol.overlap_baseline` | active episode suppression | `evaluatePartition` | PARTIAL | reusable interval model |
| Cost models | `cost_models` | base and stress identities | `netReturn*` | PARTIAL | bind every promotion field |
| Partitions | `partitions` | development/validation/OOS/holdout | config loader | PASS | retain |
| Sample floors | `sample_and_dependence` | all episode, pair, block, breadth floors | config loader | PARTIAL | bind minimum slices and mutation tests |
| Effective blocks | `effective_block_floor` | non-empty instrument-year blocks | `episodeBreadth` | PASS | retain |
| Instrument concentration | `instrument_contribution_max` | maximum 40% | `episodeBreadth` | PASS | retain |
| Regime/calendar breadth | `minimum_regime_or_calendar_slices` | at least 3 registered slices | `regimeBreadth` | FAIL | bind and use actual regime/calendar definition |
| Bootstrap | `bootstrap` | block resampling, 10,000 replicates | `applyBootstrap` | PASS | retain |
| Matched placebo | `placebo_contract.matched_non_signal` | one-to-one same regime/session distance | `attachPlacebos` | FAIL | implement exact eligibility/matching |
| Timestamp placebo | `placebo_contract.timestamp_placebo` | 10,000 deterministic selections | `timestampPlacebo` | FAIL | implement selections, not pool count |
| Secondary sign permutation | `randomisation.secondary_sign_permutation` | mixed-direction stratified diagnostic | absent | FAIL | implement typed diagnostic |
| Top-5% exclusion | `falsification_suite.tests` | remove max(1, ceil(5%N)), retain IDs | `buildFalsification` | PARTIAL | add typed result and tests |
| Leave-one-instrument-out | `falsification_suite.tests` | one result per frozen instrument | `leaveOneOut` | PARTIAL | retain with explicit aggregation tests |
| SPY regime split | `falsification_suite.tests` | signal-time SPY SMA200 regime | `descriptiveSlices` | FAIL | implement actual slices |
| Theme groups | `falsification_suite.tests` | three exact groups | `descriptiveSlices` | FAIL | implement group aggregation |
| SMA robustness | `robustness_variants` | exact variants only | `variantSummary` | PARTIAL | preserve all unperturbed semantics |
| ATR robustness | `robustness_variants` | exact distance-only variants | `variantSummary` | PARTIAL | preserve all unperturbed semantics |
| Stress costs | `cost_models.stress` | diagnostic only | `simulateLong` | PASS | retain |
| 20-session overlap | `robustness_variants` | trading-session index, per instrument | `overlapSummary` | FAIL | replace episode-array indexing |
| Zero comparator | `secondary_benchmarks` | numeric zero baseline | placeholder output | FAIL | implement arithmetic benchmark |
| SPY buy-and-hold | `secondary_benchmarks` | numeric matching-date diagnostic | placeholder output | FAIL | implement arithmetic benchmark |
| Same-asset buy-and-hold | `secondary_benchmarks` | numeric descriptive diagnostic | absent | FAIL | implement arithmetic benchmark |
| Abstention coverage | `abstention` | typed visible categories | `Abstentions` map | PARTIAL | bind categories and tests |
| Falsification disposition | `falsification_suite` | PASS/FAIL/INSUFFICIENT/INFORMATIONAL/N/A | absent | FAIL | add typed model |
| Promotion classification | `primary_metric`, `sample_and_dependence` | falsification evaluated first | `run` classification order | FAIL | refactor dependency order |
| Holdout protection | `partitions.final_holdout` | fail closed at 2025-01-01 | date guards/guard record | PASS | retain |
| Execution authority | `execution_authority` | NONE; no fills | runner/config | PASS | retain |

## Baseline conclusion

The v1.5 manifest remains authoritative and is not changed by VAL-03R1A.
The baseline contains material implementation gaps, so the harness is not
conformant until the listed bounded corrections and synthetic tests pass.
