# ROADMAP CHANGE — LLM Cost and Context Efficiency

**Date:** 2026-08-10  
**Status:** Proposed roadmap requirement; no implementation authority  
**Scope:** Cross-cutting, primarily Phases 06 and 08, with audit/governance hooks where naturally required

## Trigger

Phase 00 hosted-model evaluation exposed a roadmap gap. Jax is beginning to record provider token/cost evidence, but the roadmap did not explicitly require the production AI/research layer to minimise avoidable model calls, constrain context growth, use provider caching deliberately, reuse exact results safely, route work by capability/cost, and expose AI spend as an auditable system property.

Without an explicit requirement, token cost could become an emergent property of prompts and agent behaviour rather than a controlled resource.

## Decision

Add **LLM Cost and Context Efficiency** as a cross-cutting architectural requirement.

> The cheapest correct model call is a reused, already-proven result; the next cheapest is a minimal-context call to the least expensive model proven adequate for the task. Premium inference is an explicit escalation, not a default.

Cost optimisation must never weaken provenance, correctness, replayability, freshness, safety, uncertainty handling, or human approval controls.

## Roadmap placement

### Phase 00

No production optimisation work is added. The frozen benchmark remains a capability measurement. Do not add application-level response caching, semantic reuse, context compression, prompt rewrites for cacheability, dynamic routing or batch substitution. Natural provider-side prompt caching may be observed and accounted for.

### Phase 01

Where AI-call audit contracts are introduced or touched, ensure they can later record provider/model identity, prompt/output/policy versions, input/evidence fingerprints, token categories, retries, pricing snapshot, calculated cost, cache/reuse decision, routing/escalation decision and whether an LLM call was avoided. Do not build the full cache here merely to satisfy metadata requirements.

### Phase 06

Add bounded-context requirements for evidence selection, task-specific context budgets, version-aware evidence packets, incremental research, duplicate suppression, provenance-preserving summaries and explicit oversize-context behaviour.

See `06-research-recommendations/LLM-CONTEXT-BUDGET-REQUIREMENTS.md`.

### Phase 08

Implement the production cost/context control plane: safe exact-result reuse, provider prompt-cache-aware request construction, model routing/escalation, token/spend budgets, batch/offline paths where justified, usage telemetry and durable-agent context compaction/checkpoint rules.

See `08-ai-tooling-agents/LLM-COST-CONTEXT-EFFICIENCY-REQUIREMENTS.md`.

### Governance

Any work package introducing a paid hosted-model path must satisfy `governance/LLM-COST-BUDGET-GATE.md`.

## Non-goals

This change does not choose a permanent provider, set permanent prices, require a vector database, permit stale financial evidence reuse, let cached AI output bypass safety/risk/approval gates, optimise frozen benchmarks after seeing results, or authorize live trading.

## Acceptance for roadmap integration

This roadmap change is integrated when the referenced requirement documents are present; future Phase 06/08 planning traces to them; provider prices remain runtime/configuration evidence rather than hard-coded truth; AI usage is treated as a measurable budgeted resource; and benchmark-integrity exceptions are preserved.
