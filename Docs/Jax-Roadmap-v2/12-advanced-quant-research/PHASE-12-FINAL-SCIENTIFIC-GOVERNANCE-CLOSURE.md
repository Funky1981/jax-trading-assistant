# Phase 12 Final Scientific Governance Closure

## Status

Phase 12 remains `COMPLETE / CONDITIONAL GO` pending external technical-lead
closure. Phase 13 is `NOT STARTED`.

The API-cost disposition is accepted without additional inference:

- `ORIGINAL INTERNAL COST ESTIMATE = $2.677034`.
- `CORRECTED RETAINED-RESPONSE COST = $2.678539`.
- `EXACT HISTORICAL PROVIDER-BILLED HYP COST = NOT RECONCILABLE FROM RETAINED EVIDENCE`.

The reusable request-level calculator and retry-usage durability are corrected
and regression-tested. This is an accepted accounting disposition; it does
not imply that the retained-response total is the complete historical bill.

## HYP-EVENT-001A corrected status

`HYP-EVENT-001A — NOT VALIDATED`.

The observed 2024 values remain retained as:

- `EXPLORATORY 2024 RESULT — NOT VALIDATED`.
- Direction-only cost-adjusted mean: `0.00419212`, 92 observations, 55.43% hit rate.
- Evidence-conditioned cost-adjusted mean: `0.00853903`, 59 observations, 59.32% hit rate.

They are not formal/pristine OOS evidence, profitability evidence, edge
evidence or promotion evidence. The historical run did not record the required
development → validation → falsification → progression decision → candidate
freeze admission sequence.

## OOS admission and contamination controls

`internal/modules/advancedquant/oos_admission.go` adds content-bound immutable
admission records requiring all prerequisite result identities, ordered UTC
completion times, a deterministic progression decision and candidate-freeze
identity. `FrozenOOSRun.ScoreOOS` fails closed without that record.

The one-way lifecycle distinguishes:

`SEALED` → `ADMISSIBLE_FOR_OOS` → `OPENED_FORMAL_OOS` → `CONTAMINATED` /
`EXPLORATORY_ONLY`.

An opened partition cannot be restored to pristine OOS status.

## Classifier qualification

`internal/modules/hypevidence/qualification.go` adds an immutable,
identity-bound qualification artifact requiring:

- exact sample and selection policy;
- event and attempt identities;
- durable usage for every attempt;
- raw-response identities;
- structured/evidence/future-information/injection/abstention checks;
- semantic-review method;
- deterministic decision identity.

Bulk classification now requires an exact matching `QUALIFICATION_PASS` for
classifier, prompt, provider and model. Historical HYP qualification remains
explicitly incomplete; no missing artifact was fabricated.

## Retry usage durability

The classifier persists provider usage before structured-output acceptance,
including responses that will be rejected and retried. The cumulative budget
accounts for each retained attempt, and restart cost loading prefers the
per-attempt usage audit to avoid duplicate accounting.

## Analysis population and exclusion contract

`internal/modules/advancedquant/population.go` adds typed populations:

- `CLASSIFIED_EVENT_POPULATION`;
- `DIRECTIONAL_EVENT_POPULATION`;
- `EVIDENCE_CONDITIONED_POPULATION`;
- `RETURN_ELIGIBLE_POPULATION`;
- `ANALYSIS_POPULATION`.

Exclusions use typed reasons such as `NON_DIRECTIONAL_NEUTRAL`,
`CLASSIFIER_ABSTENTION`, `EVIDENCE_THRESHOLD_NOT_MET`,
`MARKET_DATA_UNAVAILABLE`, `SESSION_WINDOW_INCOMPLETE` and
`BENCHMARK_UNAVAILABLE`. Population summaries report classifier distribution,
typed exclusions, partition/direction/evidence-state rates and a mandatory
selection-bias flag when the conditioned subset differs from the directional
population.

The prior generic `skipped_missing_market_or_window` interpretation is
superseded. The 294 historical exclusions are primarily NEUTRAL and
INSUFFICIENT_EVIDENCE classifier states.

## Promotion and scientific boundaries

- Promotion: `PROMOTION_CLOSED`.
- Trading edge: `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`.
- 2025 holdout: `SEALED`.
- Forward paper evidence: `0 DAYS / 0 ORDERS`.
- Recommendation logic: unchanged.
- No paper/live execution authority was added.

## Verification

Focused tests pass for:

- OOS admission ordering and fail-closed scoring;
- OOS contamination lifecycle;
- classifier qualification compatibility and invalidation;
- provider retry/accounting durability;
- typed population and selection-bias reporting.

The full repository suite, vet, and diff check pass at the final closure
working tree. No test makes a paid API call. The prior Docker race-detector
result for workflow and paper-trading packages remains accepted; native
Windows cgo is still unavailable.

## Adversarial review

Reviewed attacks include invalid OOS opening, retrospective admission,
candidate-freeze replacement, contamination reset, classifier identity reuse,
missing retry liability, neutral-as-market omission, hidden population
selection, comparison-population overclaiming, 2025 access and promotion
bypass.

Required closure status:

`Blocking scientific-governance findings remaining: 0`

`Adversarial scientific-governance review: PASS`

The scientific result itself remains `NOT VALIDATED`; this is not a positive
performance finding.
