# Orchestration Pipeline Map

Use this map to trace calls and configs.

## Primary Entry Points

- `cmd/trader/frontend_api.go`
- `cmd/research/main.go`

## Orchestrator Wiring

- `internal/modules/orchestration/service.go`
- `internal/modules/orchestration/adapters.go`
- `libs/utcp/`

## Related Config

- `config/providers.json`
- `config/jax-core.json`

## Common Failure Modes

- Contract shape drift between API handler and orchestrator client.
- Provider URL/env mismatch in runtime config.
- Partial availability of Memory, Agent0, Dexter backends.

## Fast Diagnostic Commands

- `docker compose logs -f jax-trader`
- `docker compose logs -f jax-research`
- `go test ./internal/modules/orchestration/...`
