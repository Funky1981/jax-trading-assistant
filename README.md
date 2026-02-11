# jax-trading assistant

**Canonical docs live in `Docs/`** (see `Docs/README.md` for the index).

## Quick Links

- Project overview: `Docs/PROJECT_OVERVIEW.md`
- Status snapshot: `Docs/STATUS.md`
- Roadmap: `Docs/ROADMAP.md`
- Quick start: `Docs/QUICKSTART.md`
- IB setup & bridge: `Docs/IB_GUIDE.md`

## Services

- **Jax API**: `go run ./services/jax-api/cmd/jax-api`
  - Endpoints: `GET /health`, `POST /risk/calc`, `GET /strategies`, `POST /symbols/{symbol}/process`, `GET /trades`, `GET /trades/{id}`
- **Jax Memory**: `go run ./services/jax-memory/cmd/jax-memory`
  - UTCP endpoint: `POST /tools` (`memory.retain`, `memory.recall`, `memory.reflect`)

Vendored repos:
- `services/hindsight/` (pinned commit in `services/hindsight/UPSTREAM.md`)
- `dexter/`
- `Agent0/`

## Environment

Use local `.env` files (or shell env vars) for secrets and keep them untracked.

- `dexter/.env` for Dexter API keys
- `services/hindsight/.env` for Hindsight API keys

## Testing (Go)

- `scripts/test.ps1`
- `go test ./...`
- `make test`
- Lint: `golangci-lint run ./...`

## Storage

Postgres schema: `db/postgres/schema.sql`.

Quick local Postgres:

- `docker compose -f db/postgres/docker-compose.yml up -d`
- Apply schema from `db/postgres/schema.sql`
