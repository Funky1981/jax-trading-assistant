# Phase 08 Gate — Controlled AI Tools & Durable Research Agents

## Required evidence
- Every Phase 08 work package has an architecture-review decision.
- No unresolved NO-GO remains.
- CONDITIONAL GO items are explicitly tracked and are non-blocking.
- Phase-level tests/benchmarks are reproducible from documented commands.
- Repository state and migrations are known and clean enough to proceed.
- Safety and paper/live boundaries are explicitly verified.
- The phase exit condition below is demonstrated, not merely asserted.
- Paid-model paths have auditable token/cache/cost evidence and deterministic task/run budget enforcement.
- Exact result reuse has complete validity/invalidation keys; semantic result reuse is disabled unless separately approved and evaluated.
- Durable agents reconstruct bounded working context from checkpointed state rather than replaying unbounded transcripts.
- Model routing/escalation is versioned, explainable and backed by task-specific evaluation evidence.

## Exit condition
A long-running research task can stop/resume, use only permitted tools, preserve step evidence and budgets, survive failure, and improve a report without bypassing recommendation/risk gates.

## Decision
Reviewer returns one of:
- **GO PHASE 08**
- **CONDITIONAL GO PHASE 08**
- **NO-GO PHASE 08**
- **ROADMAP CHANGE**

The next phase cannot start before this gate is resolved.
