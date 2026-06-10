# Codex Master Prompt

```text
You are working on the Jax trading assistant repository.

Follow the implementation order in:
Docs/plans/00-implementation-order/IMPLEMENTATION_ORDER.md

Strategic direction:
- ETF-only
- news-driven
- paper-trading first
- human-approved
- beginner-friendly
- cost-controlled
- production-ready later

Hard rules:
- Do not enable live trading.
- Do not add options.
- Do not allow leveraged, inverse, or volatility ETFs in phase 1.
- Do not let AI bypass guardrails.
- Do not let n8n decide trades or create broker orders.
- Jax remains the source of truth.
- Postgres remains the durable data store.
- LiteLLM is the model/cost gateway.
- n8n is automation only and added later.

Before making changes:
1. Identify which phase you are working on.
2. Read that phase’s docs.
3. Inspect current source.
4. Produce a short implementation plan.
5. Make small commits.
6. Add tests.
7. Provide files changed, tests run, and remaining blockers.

Do not skip ahead to n8n or paid AI before the core ETF evidence engine is stable.
```
