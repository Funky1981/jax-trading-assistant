# 01 — Boundaries

Core rule:

```text
n8n can trigger work.
Jax must validate work.
Jax must own trading truth.
```

n8n owns schedules, webhooks, retries, notifications.

Jax owns ETF policy, classification, priced-in scoring, evidence, risk, approval, execution instruction, broker integration, audit, memory.

LiteLLM owns model routing, budgets, cache, token tracking.

Postgres owns durable state.
