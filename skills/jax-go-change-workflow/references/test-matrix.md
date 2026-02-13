# Go Test Matrix

Use this matrix to expand scope only when needed.

## Typical Scopes

- Single service edit:
  - `go test ./services/<service>/...`
- Single library edit:
  - `go test ./libs/<library>/...`
  - then test direct service consumers if known
- Root wiring or shared contracts edit:
  - `go test ./...`

## High-Risk Areas

- `libs/contracts/`: run broad dependent tests.
- `libs/utcp/`: run service tests that integrate UTCP tools.
- `libs/trading/executor/` and `services/jax-trade-executor/`: include golden/replay where behavior matters.
- `services/jax-api/` orchestration handlers: include frontend data/hook tests when response shape changes.

## Fast Iteration Pattern

1. `go test ./path/to/edited/package/...`
2. `go test ./path/to/closest/service/...`
3. broaden only when change crosses module boundaries
