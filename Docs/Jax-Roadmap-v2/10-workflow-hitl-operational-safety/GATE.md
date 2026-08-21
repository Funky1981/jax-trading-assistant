# Phase 10 Gate — Workflow, HITL & Operational Safety

## Required evidence
- Every Phase 10 work package has an architecture-review decision.
- No unresolved NO-GO remains.
- CONDITIONAL GO items are explicitly tracked and are non-blocking.
- Phase-level tests/benchmarks are reproducible from documented commands.
- Repository state and migrations are known and clean enough to proceed.
- Safety and paper/live boundaries are explicitly verified.
- The phase exit condition below is demonstrated, not merely asserted.

## Exit condition
Every state transition is explicit, authorised, auditable and recoverable; failures cannot silently advance to a more dangerous state.

## Decision
Reviewer returns one of:
- **GO PHASE 10**
- **CONDITIONAL GO PHASE 10**
- **NO-GO PHASE 10**
- **ROADMAP CHANGE**

The next phase cannot start before this gate is resolved.
