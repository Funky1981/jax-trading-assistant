# jax-trading assistant

Specs live in `Docs/`.

Initial backend scaffolding (spec `Docs/02_utcp_providers.md`):

- `config/providers.json`
- `internal/infra/utcp/*`
  - Includes `UTCPClient` (HTTP/local transports) and strict config loading
  - Includes local `risk` tools (`risk.position_size`, `risk.r_multiple`)
  - Includes local `backtest` stub tools (`backtest.run_strategy`, `backtest.get_run`)

Jax Core (spec `Docs/04_jax_core.md`):

- Entrypoint: `go run ./cmd/jax-core`
- HTTP endpoints: `GET /health`, `POST /risk/calc`, `GET /strategies`, `POST /symbols/{symbol}/process`, `GET /trades`, `GET /trades/{id}`

Vendored repos:

- `dexter/`
- `Agent0/`

## Architecture

See `ARCHITECTURE.md`.

## Testing

### Automated (Go)

- `scripts/test.ps1`
- Or: `go test ./...`
- Lint (recommended): `golangci-lint run ./...`

### Manual / component tests

- Dexter: `cd dexter; bun install; bun test`
- Dexter tools server (UTCP provider endpoint):
  - Mock mode: `cd dexter; $env:DEXTER_TOOLS_MOCK=\"1\"; bun run tools:server`

## Postgres (Storage)

Storage is designed to run on Postgres (schema in `db/postgres/schema.sql`). You can stand up Postgres any way you like; tests do not require a live DB (they use `sqlmock`).

Quick local Postgres:

- `docker compose -f db/postgres/docker-compose.yml up -d`
- Apply schema from `db/postgres/schema.sql`

Smoke test (risk/backtest/storage):

- Set `JAX_POSTGRES_DSN=postgres://jax:jax@localhost:5432/jax?sslmode=disable`
- Run: `go run ./cmd/jax-utcp-smoke`
