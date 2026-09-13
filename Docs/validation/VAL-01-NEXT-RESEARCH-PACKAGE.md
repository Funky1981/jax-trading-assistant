# VAL-01 Next Research Package

## Status

`DEFINED ONLY — NOT EXECUTED`. This package is not Phase 13, not real
forward-paper trading and not authorization to select a new strategy.

## Evidence gap addressed

Existing Jax strategy logic lacks a candidate-specific frozen evidence bundle
that can survive the promotion contract. HYP-EVENT-001A additionally lacks a
durable historical development→validation→admission→freeze progression and
retains an accepted survivorship limitation. The next package must close those
scientific identity and evidence gaps before any forward-paper admission.

## Required research question

Can one already-existing Jax candidate be reconstructed from frozen,
point-in-time inputs and demonstrate reproducible, cost-adjusted,
benchmark-relative performance across development, validation and genuinely
unseen OOS/walk-forward windows, while surviving null/placebo, concentration,
regime, survivorship and parameter-perturbation checks?

This is a falsifiable research question, not a claim that an edge exists.

## Scope and data rules

- Select from the existing inventory only after a separate registered design
  decision; do not shop across strategies after inspecting outcomes.
- Bind candidate ID/version, code SHA, configuration hash, dataset identities,
  provider contracts, calendar, corporate-action mode and cost model.
- Use only information knowable at decision time; preserve publication,
  acceptance, session and retrieval semantics.
- Define the universe and inclusion/exclusion rules before scoring.
- Resolve or explicitly bound symbol changes, delistings, survivorship and
  issuer/sector membership.
- Keep a final untouched holdout where the chosen research line requires one.

## Partitions

Freeze the partition map before implementation:

- development: candidate construction only;
- validation: threshold/configuration selection;
- OOS/walk-forward: frozen candidate, unseen windows;
- final holdout: sealed and read only after an external decision.

No repeated OOS repair is permitted. Any material change after OOS inspection
creates a contaminated research line and needs a new independent evaluation.

## Minimum evidence bundle

- hypothesis and economic rationale;
- exact entry, exit, sizing, risk, abstention and invalidation rules;
- dataset ID/hash and provenance manifest;
- benchmark and trading calendar;
- realistic fees, spread, slippage, latency and fill assumptions;
- sample-size and dependence rationale;
- development/validation/OOS results;
- walk-forward or equivalent future-only windows;
- registered baseline/null;
- complete falsification/trial ledger;
- issuer/sector/regime/outlier concentration analysis;
- calibration and resolution where confidence/probabilities exist;
- reproducible artifact and failure/rejection outcome.

## Success and rejection

The package succeeds scientifically if the frozen artifact is reproducible and
the candidate either survives the preregistered tests or is cleanly rejected.
It must be rejected or remain unpromoted if the baseline wins, the effect
vanishes after costs, placebo/null is comparable, evidence is concentrated or
unstable, sample/data quality is inadequate, leakage is found, or the OOS
result cannot be reconstructed.

## Spend and execution boundary

- Maximum new paid spend: `$0` unless separately authorized.
- No hosted inference, paid data, GPU, broker call, paper order or live order is
  part of this package definition.
- No recommendation logic changes and no Phase-13 work.

## Required terminal decision

The package must produce one of: `FORWARD_PAPER_ELIGIBLE` for a specific frozen
version, `RESEARCH_CANDIDATE`, `INSUFFICIENT_EVIDENCE`, `FAILED_VALIDATION` or
`CONTAMINATED_FOR_FORMAL_OOS`. A positive result is not required.
