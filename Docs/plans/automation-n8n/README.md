# n8n Automation Integration

## Purpose

Add n8n after the current redesign as an automation layer for repeated workflows.

## Target Split

```text
n8n = scheduler / notification runner / retry layer
Jax = source of truth / trading logic / risk / approval validation
LiteLLM = model/cost gateway
Postgres = durable truth
```

## n8n May

```text
trigger Jax workflows
schedule backfills
send notifications
call Jax APIs
call LiteLLM for low-risk summaries
retry failed jobs
```

## n8n Must Not

```text
decide trades
approve trades by itself
create broker orders
change stops
calculate position size as truth
bypass Jax guardrails
enable live trading
```

## Docs

- `01-boundaries.md`
- `02-safe-workflows.md`
- `03-forbidden-workflows.md`
- `04-integration-contracts.md`
- `05-security-and-secrets.md`
- `06-cost-saving-patterns.md`
- `07-rollout-plan.md`
- `08-uat-and-acceptance.md`
- `09-codex-prompts.md`
