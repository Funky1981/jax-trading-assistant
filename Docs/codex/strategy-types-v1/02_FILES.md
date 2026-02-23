# 02_FILES: What to add/modify

## Add (new)
- `libs/strategytypes/metadata.go`
- `libs/strategytypes/types.go`
- `libs/strategytypes/registry.go`
- `libs/strategytypes/validate.go`

- `libs/strategytypes/sameday/earnings_drift_v1.go`
- `libs/strategytypes/sameday/news_repricing_v1.go`
- `libs/strategytypes/sameday/opening_range_to_close_v1.go`
- `libs/strategytypes/sameday/panic_reversion_v1.go`
- `libs/strategytypes/sameday/index_flow_v1.go`

- `services/jax-api/internal/app/strategy_types_registry.go` (or similar)
- `services/jax-api/internal/infra/http/strategy_types_routes.go`

## Modify
- `services/jax-api/cmd/jax-api/main.go` (wire registry + route)

## Optional (later)
- Add instance validation that checks `strategyId` exists and `params` pass validation.
