# 05 — Security and Secrets

Run n8n privately:

```text
LAN only
Tailscale
WireGuard
```

Use:

```text
N8N_BASIC_AUTH_ACTIVE=true
N8N_ENCRYPTION_KEY=long-random-secret
signed webhooks
dedicated Jax service key
dedicated LiteLLM virtual keys
```

n8n must not have direct broker execution scope.
