# Phase 06 — Research & Recommendation Engine

## Purpose
    Combine evidence, quantitative context and controlled AI reasoning into explainable recommendations without execution authority.

## Prerequisites
- Phases 04 and 05 GO

## Reference systems to inspect at this phase
Fincept research/agent patterns only where they support structured evidence; Jax-specific recommendation grammar remains independent. Also read `LLM-CONTEXT-BUDGET-REQUIREMENTS.md` and `../references/LLM-COST-CONTEXT-EFFICIENCY.md`.

## Work packages
- `06.01` — Define evidence packet contract
- `06.02` — Research planner/context builder (including bounded context, evidence selection and incremental-research requirements)
- `06.03` — Bull case / bear case / contradiction / unknown extraction
- `06.04` — Recommendation grammar and eligibility rules
- `06.05` — Freshness/evidence sufficiency gates
- `06.06` — Confidence calibration contract
- `06.07` — UI/API read model for useful recommendations

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Cross-cutting LLM context requirement
Research context must be bounded, provenance-preserving and built from task-relevant evidence rather than unbounded history/corpus dumps. Oversize handling must be explicit; material contradictions or exact evidence must not be silently truncated.

## Exit gate
Jax can produce reproducible evidence-linked WATCH/NO_TRADE/CANDIDATE-style outputs with thesis, counter-evidence, unknowns, invalidation and data freshness; no order or approval state is created.

See `GATE.md`. Every work package requires independent review before the next one starts.
