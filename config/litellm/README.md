# Jax LiteLLM Local Gateway

This folder provides a local-first LiteLLM gateway for Jax model calls.

Start:

```powershell
cd config/litellm
$env:JAX_LITELLM_MASTER_KEY="replace-with-local-virtual-key"
docker compose up -d
```

Jax runtime env:

```env
AI_GATEWAY_BASE_URL=http://localhost:4000
AI_GATEWAY_API_KEY=replace-with-local-virtual-key
AI_DEFAULT_MODEL=local-small
AI_STRONG_MODEL_ENABLED=false
```

Provider keys stay in LiteLLM environment variables, not in prompts or source files.
