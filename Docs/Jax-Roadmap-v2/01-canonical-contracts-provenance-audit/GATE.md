# Phase 01 Gate — Canonical Contracts, Provenance & Audit

## Required evidence
- Every Phase 01 work package has an architecture-review decision.
- No unresolved NO-GO remains.
- CONDITIONAL GO items are explicitly tracked and are non-blocking.
- Phase-level tests/benchmarks are reproducible from documented commands.
- Repository state and migrations are known and clean enough to proceed.
- Safety and paper/live boundaries are explicitly verified.
- The phase exit condition below is demonstrated, not merely asserted.

## Exit condition
A reviewer can trace a representative output to immutable/identifiable inputs, source/provider, version and timestamp without depending on transient logs.

## Decision
Reviewer returns one of:
- **GO PHASE 01**
- **CONDITIONAL GO PHASE 01**
- **NO-GO PHASE 01**
- **ROADMAP CHANGE**

The next phase cannot start before this gate is resolved.
