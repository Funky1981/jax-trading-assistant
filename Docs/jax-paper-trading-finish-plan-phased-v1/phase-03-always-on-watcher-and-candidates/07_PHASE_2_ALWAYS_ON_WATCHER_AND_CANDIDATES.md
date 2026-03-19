# Phase 2 — Always-On Watcher and Candidate Trades

## Objective
Make `cmd/trader` continuously scan enabled strategy instances and emit candidate trades.

## Why this matters
Without this, Jax is still manual-analysis software, not an always-on assistant.

## Required deliverables

### 2.1 Watcher loop
Add long-running watcher service in trader runtime:
- load enabled instances
- schedule scans by timeframe
- evaluate setup conditions
- emit candidate trades

Files to add:
- `cmd/trader/trade_watcher.go`
- `cmd/trader/instance_scheduler.go`
- `internal/modules/candidates/service.go`
- `internal/modules/candidates/store.go`

### 2.2 Candidate trade model
New table:
- `candidate_trades`

Minimum fields:
- `candidate_id`
- `instance_id`
- `strategy_type_id`
- `symbol`
- `direction`
- `status`
- `entry_plan`
- `stop_plan`
- `target_plan`
- `risk_blockers`
- `expires_at`
- `created_at`

### 2.3 Candidate event log
New table:
- `candidate_trade_events`

Track:
- detected
- qualified
- blocked
- queued
- expired
- approved
- rejected
- reanalyzed

### 2.4 Candidate endpoints
Add:
- `GET /api/v1/candidates`
- `GET /api/v1/candidates/{id}`
- `POST /api/v1/candidates/{id}/refresh`

### 2.5 Live update transport
Add SSE first:
- `GET /api/v1/events/stream`
