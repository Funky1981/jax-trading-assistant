# Phase 10 — Workflow, HITL & Operational Safety

## Purpose
    Formalise the controlled journey from research to human-approved paper intent with auditability and recovery.

## Prerequisites
- Phases 08 and 09 GO

## Reference systems to inspect at this phase
Fincept workflow executor, audit logger, confirmation service, risk controls and Alpha Arena kill/crash-recovery concepts.

## Work packages
- `10.01` — State machine for recommendation→risk→approval→paper intent
- `10.02` — Explicit confirmation/HITL contract
- `10.03` — Permission boundaries for destructive tools
- `10.04` — Append-only workflow audit
- `10.05` — Circuit breakers/kill switches
- `10.06` — Crash recovery and reconciliation states
- `10.07` — Operator health/observability

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
Every state transition is explicit, authorised, auditable and recoverable; failures cannot silently advance to a more dangerous state.

See `GATE.md`. Every work package requires independent review before the next one starts.
