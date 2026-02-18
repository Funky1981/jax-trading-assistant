# jax-trading assistant

**Canonical docs live in `Docs/`** (see `Docs/README.md` for the index).

## Quick Links

- Project overview: `Docs/PROJECT_OVERVIEW.md`
- Status snapshot: `Docs/STATUS.md`
- Roadmap: `Docs/ROADMAP.md`
- Quick start: `Docs/QUICKSTART.md`
- IB setup & bridge: `Docs/IB_GUIDE.md`

## Architecture

Jax uses a **modular monolith** with two runtime entrypoints (ADR-0012):

| Runtime | Port | Role |
|---------|------|------|
| `cmd/trader` | 8100 | Deterministic trade execution — loads approved strategy artifacts only |
| `cmd/research` | 8091 | Orchestration pipeline — Agent0, memory, Dexter integration |

External boundaries kept as separate processes:
- **jax-api** (8081) — REST API for the frontend dashboard
- **ib-bridge** (8092) — Interactive Brokers Gateway adapter
- **agent0-service** (8093) — LLM/AI agent
- **jax-memory** (8090) — Memory layer (Hindsight)
- **jax-market** (8095) — Market data feed

## Quick Start

```powershell
docker compose up -d
```

Services start automatically. Trader and research runtimes load approved strategy artifacts from Postgres on startup.

See `Docs/QUICKSTART.md` for full setup including IB Gateway connection.

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
