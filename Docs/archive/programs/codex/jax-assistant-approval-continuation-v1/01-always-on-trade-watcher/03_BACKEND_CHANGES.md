# Backend Changes

## Files to add
- `cmd/trader/trade_watcher.go`
- `cmd/trader/instance_scheduler.go`
- `cmd/trader/candidate_trade_service.go`
- `cmd/trader/live_event_stream.go`
- `internal/modules/candidates/service.go`
- `internal/modules/candidates/store.go`

## Files to modify
- `cmd/trader/main.go`
  - start watcher loop on boot
- `cmd/trader/frontend_api.go`
  - add candidate trade endpoints
  - add SSE endpoint for watcher updates
- `cmd/trader/strategy_instances_loader.go`
  - preserve mode/status fields
- `internal/modules/execution/engine.go`
  - accept only approved candidate trades, not raw chat/model proposals

## New endpoints
- `GET /api/v1/candidates`
- `GET /api/v1/candidates/{id}`
- `POST /api/v1/candidates/{id}/refresh`
- `GET /api/v1/events/stream` (SSE)
