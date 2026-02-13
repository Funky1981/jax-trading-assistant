# Orchestration Pipeline Map

Use this map to trace calls and configs.

## Primary Entry Points

- `services/jax-api/internal/infra/http/handlers_orchestration_v1.go`
- `services/jax-signal-generator/internal/orchestrator/client.go`

## Orchestrator Wiring

- `services/jax-orchestrator/cmd/jax-orchestrator-http/clients.go`
- `services/jax-orchestrator/internal/...` application logic

## Related Config

- `config/providers.json`
- `config/jax-core.json`
- `config/jax-signal-generator.json`

## Common Failure Modes

- Contract shape drift between API handler and orchestrator client.
- Provider URL/env mismatch in runtime config.
- Partial availability of Memory, Agent0, Dexter backends.

## Fast Diagnostic Commands

- `docker compose logs -f jax-api`
- `docker compose logs -f jax-orchestrator`
- `docker compose logs -f jax-signal-generator`
