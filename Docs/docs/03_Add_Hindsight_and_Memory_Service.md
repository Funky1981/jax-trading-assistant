# 03 — Add Hindsight + Memory Service (TDD First)

**Goal:** Introduce Hindsight as a container + create `jax-memory` as a facade.

## 3.1 — Add Hindsight to docker-compose (write tests first)
We want:
- A running memory backend
- `jax-memory` service that can:
  - `Ping()`
  - `Retain()`
  - `Recall()`
  - `Reflect()` (optional to start, but interface should exist)

### TDD approach
1) Create an interface in `libs/contracts`:

```go
type MemoryStore interface {
  Ping(ctx context.Context) error
  Retain(ctx context.Context, bank string, item MemoryItem) (MemoryID, error)
  Recall(ctx context.Context, bank string, query MemoryQuery) ([]MemoryItem, error)
  Reflect(ctx context.Context, bank string, params ReflectionParams) ([]MemoryItem, error)
}
```

2) Write unit tests for:
- `InMemoryMemoryStore` (fake) in `libs/testing`
- `HindsightClient` (adapter) using a mocked HTTP server (httptest)

**Do not** start by calling a real Hindsight container in unit tests.
That becomes an **integration test** later.

## 3.2 — Integration test (optional but recommended)
Add one integration test suite:
- spins docker compose
- calls `jax-memory` -> `Ping`
- retains one item -> recalls it

Keep it separate:
- `go test -tags=integration ./...`

## 3.3 — What should be stored (rules)
- Store *summaries*, not raw price tick data
- Store decision context and outcome
- Never store broker tokens, passwords, API keys

## 3.4 — Definition of Done
- `jax-memory` compiles
- Unit tests cover:
  - request building
  - error handling
  - JSON parsing
- Integration test (tagged) proves end-to-end connectivity
