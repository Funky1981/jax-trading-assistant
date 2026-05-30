# LLM API Cost and Context Management

## Purpose

Reduce paid API usage while keeping AI traceable and safe.

## Goal

```text
deterministic code first
local models second
cheap paid models third
strong paid models only when justified
track tokens/cost
cache safe outputs
never weaken guardrails
```

## Stack

```text
Jax
  ↓
PromptPackage
  ↓
CostGovernor
  ↓
ModelRouter
  ↓
LiteLLM
  ↓
Ollama / paid APIs
```

## Docs

- `01-home-server-litellm-setup.md`
- `02-jax-llm-cost-architecture.md`
- `03-model-routing-policy.md`
- `04-token-tracking-and-budgets.md`
- `05-cache-policy.md`
- `06-context-compaction.md`
- `07-jax-integration-contracts.md`
- `08-security-and-secrets.md`
- `09-uat-and-acceptance.md`
- `10-codex-prompts.md`
