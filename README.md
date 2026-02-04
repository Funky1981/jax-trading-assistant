# jax-trading assistant

Specs live in `Docs/` (`Docs/backend` for build steps, `Docs/frontend` for UI docs).

Repo scaffold and service skeletons (spec `Docs/backend/02_Repository_Scaffold_and_Service_Skeletons.md`):

- `config/providers.json`
- `libs/utcp/*` (UTCP client + local tools + Postgres storage adapter)
- Vendored upstream services:
  - `services/hindsight/` (Hindsight memory backend; pinned commit in `services/hindsight/UPSTREAM.md`)

Jax API service lives under `services/jax-api/`:

- Entrypoint: `go run ./services/jax-api/cmd/jax-api`
- HTTP endpoints: `GET /health`, `POST /risk/calc`, `GET /strategies`, `POST /symbols/{symbol}/process`, `GET /trades`, `GET /trades/{id}`

Jax Memory service lives under `services/jax-memory/`:

- Entrypoint: `go run ./services/jax-memory/cmd/jax-memory`
- UTCP endpoint: `POST /tools` supporting `memory.retain`, `memory.recall`, `memory.reflect`
- Uses `HINDSIGHT_URL` if set; otherwise falls back to an in-memory store

Vendored repos:

- `dexter/`
- `Agent0/`

## Architecture

See `ARCHITECTURE.md`.

## Environment

Use local `.env` files (or shell env vars) for secrets and keep them untracked. Common locations:

- `dexter/.env` for Dexter API keys
- `services/hindsight/.env` for Hindsight API keys

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

## Knowledge Base (Strategy Registry)

The knowledge base stores strategy documentation (playbooks, anti-patterns, risk docs, etc.) in Postgres and makes them available at runtime for agent retrieval.

### Quick Start

```bash
make knowledge-up      # Start Postgres container (jax_knowledge db)

make knowledge-schema  # Apply schema (creates strategy_documents table)

make knowledge-ingest  # Ingest markdown files from knowledge/md/

```

### Environment

- `JAX_KNOWLEDGE_DSN` — Connection string for jax_knowledge database
- Default: `postgres://postgres:postgres@localhost:5432/jax_knowledge?sslmode=disable`

### Dry Run

Test ingestion without writing to the database:

```bash
make knowledge-ingest-dry

```

### Sanity Query

After ingestion, verify documents were loaded:

```sql
SELECT doc_type, status, count(*)
FROM strategy_documents
GROUP BY 1, 2
ORDER BY 1, 2;

```

### Package Usage

The `internal/strategyregistry` package provides runtime access to approved documents:

```go
import "jax-trading-assistant/internal/strategyregistry"

registry, err := strategyregistry.NewFromDSN(ctx, dsn)
if err != nil {
    log.Fatal(err)
}
defer registry.Close()

// Get all approved strategies
strategies, _ := registry.GetApprovedStrategies(ctx)

// Get anti-patterns for risk checks
antiPatterns, _ := registry.GetAntiPatterns(ctx)

// Get specific document by path
doc, _ := registry.GetByRelPath(ctx, "strategies/earnings_gap_v1.md")

```

All query methods enforce a `WHERE status='approved'` gate by default.
