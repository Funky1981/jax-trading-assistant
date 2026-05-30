# AI-ready tickets 06-09 CI follow-up

Date: 2026-05-30

## Findings

- PR #11 was open from `redesign` to `main`; local `redesign` matched `origin/redesign`.
- Latest CI on commit `51b9fda` failed in:
  - `go`: golangci-lint v1.61.0 was built with Go 1.23 while the repo targets Go 1.24.
  - `frontend`: two Playwright strict-mode locator collisions in AI Trading and Notifications specs.
  - `golden-tests`: Docker frontend build failed because npm 10.9.8 required `frontend/package-lock.json` to include `tailwindcss` peer/optional `yaml@2.9.0`.
- After bumping golangci-lint to v1.64.8, lint surfaced three existing findings in chat/trader packages; these were corrected with minimal lint-only edits.
- Follow-up CI after `e4bb9f6` passed frontend/import-boundary/agent0, then exposed:
  - `go`: broad `gofmt -l .` scanned archived/template Go files outside the active validation scope.
  - `golden-tests`: workflow-level Postgres service and Docker Compose Postgres both attempted to bind host port 5433.
- Follow-up CI after `4d3d2a2` passed frontend/import-boundary/agent0, then exposed:
  - `go`: shallow checkout did not include `HEAD^1` for scoped gofmt diffing.
  - `golden-tests`: Compose startup reached `jax-trader`, but `/ready` stayed unhealthy because Compose defaulted `EXECUTION_ENABLED=true` and required broker readiness.
- Follow-up CI after `6550b9d` passed frontend/import-boundary/agent0 and showed:
  - `go`: `libs/guardrails` incident IDs were millisecond-based; two rapid `Open` calls could collide and overwrite the first incident before `Resolve`, making `TestIncidentLog_ListFilter` flaky on fast Linux runners.
  - `golden-tests`: `EXECUTION_ENABLED=false` was propagated, but `docker compose up -d` still included `frontend`; frontend depends on `jax-trader` being Docker-healthy on `/ready`, while the golden workflow only needs backend `/health` endpoints and has its own wait loop.

## Validation evidence

- `npx npm@10.9.8 ci --prefer-offline` passes in `frontend/`.
- `go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run` passes.
- `go test ./...` passes.
- `npm run lint`, `npm run typecheck`, `npm run test`, and `npm run build` pass in `frontend/`.
- `npm run test:e2e` exits successfully; one module guide spec retried once and then passed.
- Follow-up workflow patch scopes gofmt to changed Go files and lets Docker Compose own the golden-test Postgres port.
- Second follow-up patch fetches enough git history for scoped gofmt and starts golden-test Compose with `EXECUTION_ENABLED=false`.
- Third follow-up patch makes `IncidentLog.Open` allocate unique same-millisecond IDs with a suffix and adds a uniqueness assertion to the filter test.
- Third follow-up patch starts only golden-test backend services (`postgres`, `db-migrate`, `ib-bridge`, `agent0-service`, `jax-research`, `jax-trader`) and adds compose log dumping if startup fails before the wait loop.

## Residual risk

- Local Docker verification could not run because Docker Desktop Linux engine was unavailable. The lockfile failure was reproduced and fixed locally with npm 10.9.8, matching the CI/Docker npm version.
