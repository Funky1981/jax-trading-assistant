# 00 — Current Gap Review

## Goal

Review the redesign branch and define the missing work needed for token caching, compaction, model routing, and cost control.

## Existing Strengths

The redesign branch already has strong foundations:

- ETF-only policy is being pushed into the product direction.
- Paper trading remains the correct default.
- AI is advisory and cannot directly trade.
- Evidence bundles are being introduced as the structured object before approval.
- AI guardrails validate model output and force deterministic rejection when guardrails fail.
- Postgres remains the source of truth.
- Research and trader runtimes are separated enough to add an LLM infrastructure layer without rebuilding the app.

## Key Existing Files To Respect

- `Docs/plans/Completed/01-etf-news-research-platform/07-research-evidence-bundles.md`
- `Docs/plans/Completed/01-etf-news-research-platform/08-ai-guardrails.md`
- `cmd/research/ai_guardrails.go`
- `cmd/research/backfill.go`
- `internal/modules/approvals`
- `internal/modules/candidates`
- `internal/modules/etfnews`
- `db/postgres/migrations`

## Missing Pieces

Jax does not yet have a dedicated layer for:

1. Building cache-friendly prompts.
2. Separating static instructions from dynamic event data.
3. Estimating tokens before a provider call.
4. Blocking provider calls that exceed budget.
5. Routing different tasks to different models.
6. Compacting raw research into reusable memory artifacts.
7. Recording actual token usage and provider cost.
8. Measuring cache-hit effectiveness.
9. Reducing expensive final-review prompts to compact decision packets.
10. Testing that AI cost controls fail closed.

## Current Risk

Without this layer, Jax can easily become expensive even if the trading logic is safe.

Main failure modes:

- every event includes full rules, full history, and raw news again
- expensive models are called before cheap filters reject weak events
- old context grows endlessly
- summaries cannot be traced back to source evidence
- agent loops retry without spend limits
- final trade review sees raw data instead of a compact decision packet
- token spend is invisible until the bill arrives

## Required Direction

The LLM layer should be boring infrastructure, not agent magic.

It should answer these questions before any model call:

- What task is this?
- Is a model actually required?
- What is the cheapest acceptable model?
- What context is static and cacheable?
- What context is dynamic and fresh?
- What old context should be retrieved?
- What old context should be ignored?
- What is the estimated token cost?
- Is the call allowed under budget?
- How will the result be stored and audited?

## Acceptance Criteria

- Existing AI guardrails remain the final policy boundary.
- Existing evidence bundle shape is reused rather than bypassed.
- Every LLM call is logged with task type, model, provider, token estimate, and cost estimate.
- Expensive final reasoning cannot run directly from raw news intake.
- Compaction creates stable artifacts that can be retrieved later.
- Failed cost checks block the call before provider invocation.
- Tests prove that budget failure does not degrade into unsafe trading behaviour.
