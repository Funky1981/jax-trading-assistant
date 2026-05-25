# LLM Cost, Cache, and Context Management Build Pack

## Purpose

This build pack adds a production-grade LLM cost and context layer for Jax.

The goal is to let Jax use powerful models for research and final reasoning without repeatedly sending huge prompts, bloated memory, full chat history, or raw news bundles into every call.

## Target Outcome

Jax should:

- keep static instructions cache-friendly
- compact old context into reusable memory artifacts
- retrieve only relevant memory for each event
- route tasks to the cheapest acceptable model
- block model calls that exceed budget limits
- log token usage and estimated cost for every run
- keep AI advisory-only and never allow model output to execute trades

## Fits Existing Redesign Branch

This pack extends the current redesign branch. It does not replace the ETF research platform or AI-ready plan.

The existing redesign branch already has the correct safety posture:

- ETF-only phase-one policy
- paper trading first
- human approval before execution
- AI explains and ranks but does not directly trade
- Postgres remains the source of truth
- evidence bundles are required before candidate approval
- deterministic guardrails override AI output

This pack adds the missing infrastructure around token caching, prompt construction, context compaction, model routing, usage logging, and budget enforcement.

## Recommended File Order

| File | Purpose |
|---|---|
| `00-current-gap-review.md` | Review what exists and what is missing |
| `01-target-architecture.md` | Define the LLM cost/context architecture |
| `02-cacheable-prompt-layout.md` | Define stable prefix and dynamic prompt sections |
| `03-context-compaction.md` | Define how raw context becomes compact memory |
| `04-model-router-and-cost-governor.md` | Define routing, budgets, and blocking behaviour |
| `05-database-schema.md` | Define Postgres tables for usage, cache, and artifacts |
| `06-implementation-plan.md` | Codex-ready build order and acceptance checks |

## Non-Negotiables

- No model can place a broker order.
- No model can approve a trade.
- No model can bypass ETF allowlist, priced-in rejection, stale quote checks, spread checks, paper-mode checks, or approval checks.
- Expensive final reasoning is only allowed after cheap classification and deterministic pre-checks.
- Every LLM call must have a traceable task type, model route, token estimate, actual token usage if available, and cost estimate.
- Prompt caching is treated as an optimisation, not a correctness dependency.
- Compacted memory must be auditable back to source evidence where possible.

## First Implementation Slice

Build this first:

1. Prompt package model
2. Cacheable prefix builder
3. Token estimator
4. Usage logger
5. Cost governor
6. Compaction service interface
7. Model router interface
8. No-op provider implementations for tests

Do not start with autonomous agents. Start with cost control and deterministic boundaries.