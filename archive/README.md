# Deprecated Services

These services were part of the microservices architecture (pre-February 2026).
They have been superseded by the modular monolith architecture.

| Service | Replaced By | Port |
|---------|------------|------|
| `jax-orchestrator` | `cmd/research` | 8091 |
| `jax-signal-generator` | `cmd/trader` | 8100 |
| `jax-trade-executor` | `cmd/trader` | 8100 |

See `Docs/ADR-0012-two-runtime-modular-monolith.md` for full migration details.

**DO NOT USE IN PRODUCTION.** Kept for historical reference only.
