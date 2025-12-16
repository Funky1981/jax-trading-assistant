# Architecture (Clean Architecture)

This repo follows a Clean Architecture / Hexagonal style:

- `internal/domain`: Pure business types (no IO, no HTTP, no DB, no UTCP).
- `internal/app`: Use-cases / orchestration. Depends on `domain` and *interfaces*.
- `internal/infra`: Adapters (UTCP client, HTTP server, storage, external services).
- `cmd/*`: Entrypoints (wire config + dependencies, start servers/processes).

## Directory structure

```text
jax-trading assistant/
  Docs/                         # Specs / build plan (01..06)
  cmd/
    jax-utcp-smoke/             # Smoke entrypoint to exercise UTCP tools end-to-end
  config/
    providers.json              # UTCP provider definitions (http/local)
  db/
    postgres/
      schema.sql                # Postgres schema for storage provider
      docker-compose.yml        # Local Postgres for development
  internal/
    infra/
      utcp/                     # UTCP client + provider adapters + tool registrations
  scripts/
    test.ps1                    # Local quality gate (gofmt, golangci-lint, go test)
  dexter/                       # Vendored Dexter repo (research agent)
  Agent0/                       # Vendored Agent0 repo (reference / inspiration)
  .github/workflows/            # CI (gofmt, golangci-lint, go test)
```

Notes:

- `internal/domain` and `internal/app` are not created yet; they’ll appear when `Docs/04_jax_core.md` begins.
- Vendored repos (`dexter/`, `Agent0/`) are treated as external projects; Jax code should not depend on their internals directly.

## Dependency rules

- `domain` must not import from any other internal layer.
- `app` may import `domain`, but must not import `infra` packages.
- `infra` may import `domain` + `app` (for wiring), but keep adapters isolated.
- Prefer defining interfaces in the consuming layer (usually `app`) and implement
  them in `infra`.

## Testing rules

- Unit tests live next to the code (`*_test.go`).
- Infrastructure code should be testable via dependency injection:
  - HTTP clients injected (so tests can use `httptest`).
  - File paths provided as parameters.
  - Avoid global state.
