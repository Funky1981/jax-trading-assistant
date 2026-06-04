# 07 — Jax Integration Contracts

Suggested config:

```env
AI_GATEWAY_BASE_URL=http://home-server:4000
AI_GATEWAY_API_KEY=sk-jax-virtual-key
AI_DEFAULT_MODEL=local-small
AI_REASONING_MODEL=paid-cheap
AI_ESCALATION_MODEL=paid-strong
AI_STRONG_MODEL_ENABLED=false
```

All calls go through:

```go
type LLMProviderClient interface {
    Complete(ctx context.Context, pkg PromptPackage) (LLMResult, error)
}
```

No direct paid provider calls outside gateway/client boundary.
