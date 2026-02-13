---
name: jax-adr0012-migration
description: Migration guardrails for ADR-0012 (Trader + Research runtime split) in jax-trading-assistant. Use when implementing modular-monolith refactors, introducing `cmd/trader` and `cmd/research`, enforcing import boundaries, or building artifact promotion and approval pathways.
---

# Jax Adr0012 Migration

Execute migration work in small, reversible phases aligned to ADR-0012.

## Workflow

1. Identify the active phase from ADR-0012 and constrain edits to that phase scope.
2. Preserve current behavior first:
   - add or maintain golden/replay coverage before invasive refactors
3. Apply phase-specific changes:
   - composition roots, orchestration seam collapse, artifact gate, import checks
4. Validate against phase gate criteria before claiming completion.

## Mandatory Constraints

- Keep Trader runtime deterministic and free of research-only dependencies.
- Preserve compatibility shims until the phase explicitly decommissions them.
- Treat artifact promotion as immutable payload plus approval state transition.

## Required Evidence

- files changed mapped to explicit phase objective
- tests proving no unapproved behavior drift
- boundary verification where Trader import deny rules apply

Use `references/phase-checklist.md` as the source of truth.
