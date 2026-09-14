# VAL-03R1A Manifest Conformance Matrix

This outcome-free matrix records the R1A baseline and later corrective closure
states. It does not load market bars or inspect VAL-03 results.

| Frozen requirement | Manifest/source | Implementation evidence | Test evidence | Baseline | Final R1B | Notes |
|---|---|---|---|---:|---:|---|
| Candidate identity/version | `selected_candidate` | `loadFrozenExperimentConfig` | config binding | PASS | PASS | unchanged |
| Confidence gate | `confidence_contract` | `actionableByConfig` | exact 0.60 tests | PASS | PASS | unchanged |
| Indicator calculations | `indicator_contract` | `buildSignal` | indicator tests | PARTIAL | PASS | all inputs bound |
| Entry/exit semantics | `protocol`, `exit_contract` | `simulateLong` | boundary tests | PARTIAL | PASS | frozen geometry |
| Non-overlap | `protocol.overlap_baseline` | active interval model | overlap tests | PARTIAL | PASS | trading-session semantics |
| Cost models | `cost_models` | `netReturn*` | cost tests | PARTIAL | PASS | base/stress bound |
| Partitions | `partitions` | config loader | drift tests | PASS | PASS | unchanged |
| Sample floors | `sample_and_dependence` | config and classifier | floor tests | PARTIAL | PASS | all floors bound |
| Effective blocks | `effective_block_floor` | `episodeBreadth` | breadth tests | PASS | PASS | unchanged |
| Instrument concentration | `instrument_contribution_max` | `episodeBreadth` | classifier tests | PASS | PASS | unchanged |
| Calendar/regime breadth | `minimum_regime_or_calendar_slices` | year×regime cells | cell tests | FAIL | PASS | exact R1B policy |
| Bootstrap | `bootstrap` | `applyBootstrap` | deterministic tests | PASS | PASS | unchanged |
| Matched placebo | `placebo_contract` | `attachPlacebos` | eligibility tests | FAIL | PASS | exact matching |
| Timestamp placebo | `placebo_contract` | deterministic selections | replicate tests | FAIL | PASS | 10,000 sets |
| Secondary sign permutation | `randomisation` | explicit typed wiring | mechanics tests | FAIL | PASS | independent signs |
| Top-5 exclusion | `falsification_suite` | ceil rule and IDs | boundary/disposition tests | PARTIAL | PASS | blocking disposition |
| Leave-one-instrument-out | `falsification_suite` | `leaveOneOut` | aggregation tests | PARTIAL | PASS | retained |
| SPY regime split | `falsification_suite` | signal-time regime | slice tests | FAIL | PASS | descriptive |
| Theme groups | `falsification_suite` | exact theme groups | aggregation tests | FAIL | PASS | descriptive |
| SMA robustness | `robustness_variants` | `variantSummary` | variant tests | PARTIAL | PASS | no replacement |
| ATR robustness | `robustness_variants` | `variantSummary` | variant tests | PARTIAL | PASS | no replacement |
| Stress costs | `cost_models.stress` | stress arithmetic | cost tests | PASS | PASS | diagnostic only |
| 20-session overlap | `robustness_variants` | session-index filter | weekend/holiday tests | FAIL | PASS | exact boundary |
| Zero comparator | `secondary_benchmarks` | numeric benchmark | arithmetic tests | FAIL | PASS | descriptive |
| SPY buy-and-hold | `secondary_benchmarks` | numeric benchmark | arithmetic tests | FAIL | PASS | descriptive |
| Same-asset buy-and-hold | `secondary_benchmarks` | numeric benchmark | arithmetic tests | FAIL | PASS | descriptive |
| Abstention coverage | `abstention` | typed visible categories | accounting tests | PARTIAL | PASS | no-trade retained |
| Falsification disposition | `falsification_suite` | closed typed statuses | disposition tests | FAIL | PASS | blocking explicit |
| Promotion classification | `primary_metric` | blocking-aware classifier | dependency tests | FAIL | PASS | falsification first |
| Holdout protection | `final_holdout` | date and run guards | 2025 tests | PASS | PASS | unchanged |
| Execution authority | `execution_authority` | `NONE`, no fills | safety tests | PASS | PASS | unchanged |

## Closure conclusion

- Baseline: PASS 9, PARTIAL 10, FAIL 12.
- Final R1B: PASS 31, PARTIAL 0, FAIL 0.
- Unresolved material ambiguities: 0.
- Manifest v1.5 remains unchanged and historical VAL-03 evidence remains
  immutable.

## R1C/R1D production-path conformance

R1C established production wiring, but R1D found that the first sign-null
implementation reused the observed directional effect when production effects
were populated. The R1D column below distinguishes implementation existence,
production-path proof, and null-statistic correctness.

| Requirement | Implementation exists | Production path proven | R1C | Final R1D | Evidence |
|---|---:|---:|---:|---:|---|
| Secondary BUY/SELL diagnostic creation | PASS | PASS | PASS | PASS | evaluator integration test |
| Typed persisted stratum identity | PASS | PASS | PASS | PASS | canonical stratum and production observation test |
| Mixed-only stratum filtering | PASS | PASS | PASS | PASS | mixed/excluded ID tests |
| Observed directional statistic | PASS | PASS | PASS | PASS | explicit observed-statistic test |
| Sign × absolute-magnitude null | PASS | PASS | PASS | PASS | R1D reference and regression tests |
| Deterministic null signs | PASS | PASS | PASS | PASS | direct SHA-256 bit-convention test |
| Unknown disposition fail-closed | PASS | PASS | PASS | PASS | classifier regression test |
| Calendar-regime cell persistence | PASS | PASS | PASS | PASS | persisted-cell test |

R1D unresolved ambiguity: 0. No outcome-bearing validation was executed.
