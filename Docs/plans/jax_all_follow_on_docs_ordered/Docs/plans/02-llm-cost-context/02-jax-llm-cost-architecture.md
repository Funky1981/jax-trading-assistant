# 02 — Jax LLM Cost Architecture

Flow:

```text
Jax task -> PromptPackage -> Context compaction -> Cost estimate -> Cost governor -> Model router -> LiteLLM -> Output validation -> Usage log
```

Add:

```text
internal/modules/llmcontext/
  prompt_package.go
  prompt_builder.go
  cacheable_prefix.go
  dynamic_context.go
  compaction.go
  memory_retrieval.go
  model_router.go
  cost_governor.go
  usage_logger.go
  provider_client.go
  noop_provider.go
```

LLM layer may produce summaries/recommendations.

LLM layer must not produce broker orders, approvals, live-mode enablement, position-size increases, or stop-loss changes.
