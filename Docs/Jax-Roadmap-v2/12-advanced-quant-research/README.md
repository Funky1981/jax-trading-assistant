# Phase 12 — Advanced Quant Research

**Status:** **AUTHORISED / IN PROGRESS — RESEARCH ONLY**

The active bounded stage is `PHASE 12 READINESS — HISTORICAL DATA & RESEARCH
HYPOTHESIS`. It is a documentation/design assessment, not Phase-12
implementation. The prerequisites for `HYP-EVENT-001A` are accepted for bounded
research. See the readiness handover and `PHASE-12-READINESS-DECISION-PACK.md`.
No result has recommendation, paper-execution or live-execution authority.

## Purpose
    Add factor, ML and automated hypothesis capabilities only where conventional analysis has a validated research need.

## Prerequisites
- Phase 07 GO
- Sufficient clean historical data
- Explicit research hypothesis

## Reference systems to inspect at this phase
Fincept AI Quant Lab, Microsoft Qlib/RD-Agent and leakage-safe ML evaluation practices.

## Work packages
- `12.01` — Qlib/RD-Agent suitability spike
- `12.02` — Feature/factor store contract
- `12.03` — Leakage-safe experiment framework
- `12.04` — Factor mining and stability tests
- `12.05` — Model selection/ensembles where justified
- `12.06` — Concept drift and rolling retraining
- `12.07` — Experiment registry and rejection criteria
- `12.08` — Promotion gate into recommendation evidence

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
    No advanced model affects recommendations until it beats defined baselines out-of-sample, is reproducible, and has monitoring/drift/failure behaviour.

See `GATE.md`. Every work package requires independent review before the next one starts.
