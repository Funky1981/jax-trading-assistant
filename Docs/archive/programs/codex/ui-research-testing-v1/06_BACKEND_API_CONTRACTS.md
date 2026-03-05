# Backend contracts (minimal)

These are API shapes the UI assumes. Implement in `jax-api` initially.

## Strategy Instances
### GET /api/v1/instances
Returns: `StrategyInstance[]`

### POST /api/v1/instances
Body: `{ instanceId, strategyId, enabled, configJson }`
Returns: `StrategyInstance`

### PUT /api/v1/instances/{id}
Body: `{ enabled?, configJson? }`

### POST /api/v1/instances/{id}/enable
### POST /api/v1/instances/{id}/disable

Persistence rule:
- Store in DB (table `strategy_instances`)
- Also allow file export/import from UI (frontend handles files; backend doesn't need filesystem).

## Backtests
### POST /api/v1/backtests/run
Body: `{ instanceId, from, to, symbolsOverride? }`
Returns: `{ runId }`

### GET /api/v1/backtests/runs?instanceId=&limit=
Returns: `BacktestRunSummary[]`

### GET /api/v1/backtests/runs/{runId}
Returns: `BacktestRunDetail`

Important:
- Replace fake backtest implementation before trusting numbers.

## Testing
### GET /api/v1/testing/status
Returns array of gate statuses for Gate0..Gate7

### POST /api/v1/testing/recon/data
Starts data reconciliation job; returns job id and summary.

### POST /api/v1/testing/recon/pnl
Starts pnl reconciliation; returns summary.

All testing endpoints must be paper-only gated.
