# jax-trading assistant

Specs live in `Docs/` (see `Docs/docs/`).

Repo scaffold and service skeletons (spec `Docs/docs/02_Repository_Scaffold_and_Service_Skeletons.md`):

- `config/providers.json`
- `libs/utcp/*` (UTCP client + local tools + Postgres storage adapter)
- Vendored upstream services:
  - `services/hindsight/` (Hindsight memory backend; pinned commit in `services/hindsight/UPSTREAM.md`)

Jax API service lives under `services/jax-api/`:

- Entrypoint: `go run ./services/jax-api/cmd/jax-api`
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
- Or: `make test`
- Lint (recommended): `golangci-lint run ./...`

### Optional (Make)

- `make test`, `make lint`, `make up`, `make down`

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
