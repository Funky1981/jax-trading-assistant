# 06_TASKS: Implementation checklist (Codex)

1) Add `libs/strategytypes` core types + registry.
2) Implement five strategy type files under `libs/strategytypes/sameday/`.
3) Add `jax-api` route: `/api/v1/strategy-types` and `/api/v1/strategy-types/{id}`.
4) Wire registry init in `services/jax-api/cmd/jax-api/main.go`.
5) Add unit tests:
   - `libs/strategytypes/sameday/*_test.go` with toy candles.
6) Add docs in `Docs/codex/strategy-types-v1/` (this pack).

Definition of done:
- `go test ./...` passes.
- Curling `/api/v1/strategy-types` returns 5 entries.
