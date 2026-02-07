# 08 — Reflection Jobs (Outcomes -> Beliefs)

**Goal:** Add scheduled reflection to produce insights that improve future decisions.

## 8.1 — What reflection does
- Pull recent `trade_decisions` + `trade_outcomes`
- Synthesize:
  - “what worked”
  - “what failed”
  - “what patterns repeat”
- Store results in `strategy_beliefs`

## 8.2 — Scheduling

Options:
- cron inside a service (simple)
- external scheduler (more production-like)

Start simple:
- daily reflection
- weekly reflection

## 8.3 — TDD
- Unit test: given decisions/outcomes, reflection generates belief items
- Golden test: belief JSON output stable

## 8.4 — Guardrails
- Beliefs must include:
  - time window
  - evidence references (IDs)
  - confidence score
- No single anecdote becomes a “law of nature”

## 8.5 — Definition of Done
- Reflection job runs locally (manual trigger is fine)
- Beliefs are retained and can be recalled by Agent0
