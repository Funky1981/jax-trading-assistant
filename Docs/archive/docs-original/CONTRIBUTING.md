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

To install `golangci-lint`:

- `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

### Dexter (vendored)

From `dexter/`:

- Install: `bun install`
- Tests: `bun test`
- Typecheck: `bun run typecheck`
