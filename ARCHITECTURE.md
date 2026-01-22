# Architecture (Clean Architecture)

This repo follows a Clean Architecture / Hexagonal style.

In the multi-service layout:

- Each service lives under `services/<name>/` and keeps its own `internal/` packages.
- Shared packages live under `libs/` and should be stable and well-tested.

## Directory structure

```text
jax-trading assistant/
  Docs/                         # Plan docs (see `Docs/backend/` and `Docs/frontend/`)
  services/
    jax-api/                    # HTTP API service (current focus)
    jax-orchestrator/           # Agent0 pipeline service (skeleton)
    jax-memory/                 # Hindsight facade service (skeleton)
    jax-ingest/                 # Dexter ingestion service (skeleton)
    hindsight/                  # Vendored Hindsight upstream (pinned; see UPSTREAM.md)
  libs/
    utcp/                       # UTCP client + tool implementations (local/http) + Postgres storage adapter
    contracts/                  # Shared DTOs/interfaces (memory schemas etc; WIP)
    observability/              # Shared logging/tracing helpers (WIP)
    testing/                    # Shared fakes/fixtures (WIP)
  cmd/
    jax-utcp-smoke/             # Smoke entrypoint to exercise UTCP tools end-to-end
  config/
    providers.json              # UTCP provider definitions (http/local)
  db/
    postgres/
      schema.sql                # Postgres schema for storage provider
      docker-compose.yml        # Local Postgres for development
  scripts/
    test.ps1                    # Local quality gate (gofmt, golangci-lint, go test)
  dexter/                       # Vendored Dexter repo (research agent)
  Agent0/                       # Vendored Agent0 repo (reference / inspiration)
  .github/workflows/            # CI (gofmt, golangci-lint, go test)
```

## Dependency rules (per service)

- `internal/domain` must not import from any other layer.
- `internal/app` may import `internal/domain`, but must not import `internal/infra`.
- `internal/infra` may import `internal/domain` + `internal/app`, but keep adapters isolated.
- Prefer defining interfaces in the consuming layer (usually `internal/app`) and implement them in `internal/infra`.

## Testing rules

- Unit tests live next to the code (`*_test.go`).
- Infrastructure code should be testable via dependency injection:
  - HTTP clients injected (so tests can use `httptest`).
  - File paths provided as parameters.
  - Avoid global state.
