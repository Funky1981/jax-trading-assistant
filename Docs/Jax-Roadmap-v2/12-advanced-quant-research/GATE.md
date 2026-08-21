# Phase 12 Gate — Advanced Quant Research

## Required evidence
- Every Phase 12 work package has an architecture-review decision.
- No unresolved NO-GO remains.
- CONDITIONAL GO items are explicitly tracked and are non-blocking.
- Phase-level tests/benchmarks are reproducible from documented commands.
- Repository state and migrations are known and clean enough to proceed.
- Safety and paper/live boundaries are explicitly verified.
- The phase exit condition below is demonstrated, not merely asserted.

## Exit condition
No advanced model affects recommendations until it beats defined baselines out-of-sample, is reproducible, and has monitoring/drift/failure behaviour.

## Decision
Reviewer returns one of:
- **GO PHASE 12**
- **CONDITIONAL GO PHASE 12**
- **NO-GO PHASE 12**
- **ROADMAP CHANGE**

The next phase cannot start before this gate is resolved.
