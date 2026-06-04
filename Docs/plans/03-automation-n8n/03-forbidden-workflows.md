# 03 — Forbidden Workflows

Forbidden:

```text
direct broker orders
risk logic in n8n
AI trade decisions in n8n
live trading enablement
tool access from untrusted news input
workflow state as trading truth
```

Required pattern:

```text
n8n -> Jax API -> Jax validates -> Jax persists -> n8n notifies
```
