# Programme Definition of Done

Code existing is not sufficient evidence that a capability is done.

## Work-package level

During Autonomous Development Mode, a work package is internally complete when:
- acceptance criteria are proven;
- meaningful success and relevant failure paths are tested;
- source/provenance/versioning requirements are satisfied;
- operational failures are observable;
- safety invariants are preserved;
- documentation/contracts are updated;
- adversarial Codex self-review finds no material unresolved blocker;
- one bounded local commit/evidence record exists;
- later-phase scope has not been pulled forward.

Internal state: `IMPLEMENTED / INTERNALLY VERIFIED`.

## Phase level

A phase is complete only when:
- all required work packages are internally verified;
- no unresolved package blocker remains;
- for Phase 05 and later, the dedicated adversarial phase review has completed;
- for Phase 05 and later, the final adversarial review has zero blocking findings;
- phase-level verification is reproducible;
- the exact phase exit condition is demonstrated, not merely asserted;
- safety and cost/dependency boundaries are known;
- the external technical lead returns `GO PHASE NN` or accepted `CONDITIONAL GO PHASE NN`.

## Programme usefulness before live trading

Jax is useful before live trading when it can:
1. ingest trustworthy market/corporate/macro/event evidence;
2. resolve affected instruments correctly;
3. calculate deterministic quantitative context;
4. produce explainable, evidence-linked recommendations;
5. replay/evaluate recommendations historically and prospectively;
6. account for portfolio state and deterministic risk;
7. operate a high-fidelity paper workflow under human approval.

Live execution is not part of this definition of useful completion and requires a separate later decision.
