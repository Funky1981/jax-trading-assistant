# Phase 11 Gate — High-Fidelity Paper Trading

## Required evidence
- Every Phase 11 work package has an architecture-review decision.
- No unresolved NO-GO remains.
- CONDITIONAL GO items are explicitly tracked and are non-blocking.
- Phase-level tests/benchmarks are reproducible from documented commands.
- Repository state and migrations are known and clean enough to proceed.
- Safety and paper/live boundaries are explicitly verified.
- The phase exit condition below is demonstrated, not merely asserted.

## Exit condition
Approved paper intents produce realistic auditable orders/fills/positions, reconcile correctly, and can run for an agreed sustained evaluation period without live execution being enabled.

## Decision
Reviewer returns one of:
- **GO PHASE 11**
- **CONDITIONAL GO PHASE 11**
- **NO-GO PHASE 11**
- **ROADMAP CHANGE**

The next phase cannot start before this gate is resolved.
