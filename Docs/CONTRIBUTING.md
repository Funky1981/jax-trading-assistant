# Contributing / Quality Gates

## Development principles

- Separation of concerns: keep `domain` pure, keep IO in `infra`.
- Test-first where practical (TDD): write a failing test for new behavior, then implement.
- DRY: factor shared logic into helpers, but don’t over-abstract early.

## Running tests

### Go (Jax backend)

- Automated: `go test ./...`
- Format: `gofmt -w .` (or `gofmt -l .` to see diffs)
- Lint: `golangci-lint run ./...` (recommended)
- Workflow wrapper: `.\scripts\go-verify.ps1 -Mode quick|standard|full`

To install `golangci-lint`:

- `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

### Dexter (vendored)

From `dexter/`:

- Install: `bun install`
- Tests: `bun test`
- Typecheck: `bun run typecheck`

### Platform Validation (Backend + Frontend + API smoke)

- Quick gate: `.\scripts\test-platform.ps1 -Mode quick`
- Full gate with visual e2e output: `.\scripts\test-platform.ps1 -Mode full -OpenVisualReport`
