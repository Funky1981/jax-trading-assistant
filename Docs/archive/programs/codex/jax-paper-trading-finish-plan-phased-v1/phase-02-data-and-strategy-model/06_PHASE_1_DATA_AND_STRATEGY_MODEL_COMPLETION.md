# Phase 1 — Data and Strategy Model Completion

## Objective
Finish the core model so Jax can manage strategy types and instances cleanly.

## Current state
- strategy instance JSON bootstrap exists
- loader upserts into `strategy_instances`
- strategies list/detail exists in API
- missing: proper management layer and richer data contracts

## Required deliverables

### 1.1 Strategy types metadata endpoint completion
Expose richer metadata:
- strategy type ID
- required params
- optional params
- timeframe/session requirements
- supported instruments
- flatten rules
- description/help text

### 1.2 Strategy instances CRUD
Add endpoints:
- `GET /api/v1/strategy-instances`
- `GET /api/v1/strategy-instances/{id}`
- `POST /api/v1/strategy-instances`
- `PATCH /api/v1/strategy-instances/{id}`
- `POST /api/v1/strategy-instances/{id}/enable`
- `POST /api/v1/strategy-instances/{id}/disable`
- `POST /api/v1/strategy-instances/{id}/clone`

Files:
- `cmd/trader/strategy_instance_handlers.go`
- `internal/modules/strategyinstances/service.go`
- `internal/modules/strategyinstances/store.go`

### 1.3 Event and dataset linkage
Finish data contracts so research runs and paper candidates can link to:
- event inputs
- dataset provenance
- strategy instance attribution

### 1.4 Backtest result persistence review
Ensure current `cmd/research/backtest.go` writes enough information to support:
- analysis page
- replay
- gate verification
