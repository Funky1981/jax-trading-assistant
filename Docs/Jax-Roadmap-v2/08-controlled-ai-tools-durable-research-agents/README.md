# Phase 08 — Controlled AI Tools & Durable Research Agents

## Purpose
    Allow models to perform deeper research through constrained, auditable tools and checkpointed tasks.

## Prerequisites
- Phase 07 GO

## Reference systems to inspect at this phase
Fincept MCP authorization/schema/async patterns and agentic research checkpoint/reflection/budget concepts. Also read `LLM-COST-CONTEXT-EFFICIENCY-REQUIREMENTS.md`, `../references/LLM-COST-CONTEXT-EFFICIENCY.md`, and `../governance/LLM-COST-BUDGET-GATE.md`.

## Work packages
- `08.01` — Tool registry and JSON-schema validation
- `08.02` — Read-only tool permission tiers
- `08.03` — Timeout/cancellation/budget controls (including paid-model token/spend accounting)
- `08.04` — Durable research task/checkpoint state (including bounded resume-context reconstruction)
- `08.05` — Adaptive gap-finding/replanning
- `08.06` — Critic/reflection stage
- `08.07` — Research memory with provenance (including exact safe reuse/invalidation contracts)
- `08.08` — Agent evaluation harness (including routing/cache/cost-quality evaluation)

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Cross-cutting LLM efficiency requirement
The controlled AI layer must treat inference spend/context as auditable resources: provider usage normalization, hard budgets, safe exact reuse, provider prompt-cache awareness, validated model routing/escalation, bounded durable-agent context and cost telemetry. Semantic-result caching remains disabled unless separately proven safe.

## Exit gate
A long-running research task can stop/resume, use only permitted tools, preserve step evidence and budgets, survive failure, and improve a report without bypassing recommendation/risk gates.

See `GATE.md`. Every work package requires independent review before the next one starts.
