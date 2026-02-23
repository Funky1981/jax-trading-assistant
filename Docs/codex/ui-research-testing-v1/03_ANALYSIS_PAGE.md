# Analysis Page (Run inspection)

Route: `/analysis`

## Primary goals
- Inspect a single backtest run (metrics + trade list).
- Compare two runs (optional, v1: side-by-side metrics only).

## UI layout
Top: Run selector
- dropdown populated by `GET /api/v1/backtests/runs?limit=...`
- or accept `runId` query param

Main: 3 panels
1. **Summary Metrics**
   - trades, winRate, avgR, maxDrawdown, pnl (if available), exposure
2. **By Symbol**
   - table: symbol, trades, winRate, pnl
3. **Trades**
   - table: symbol, entryTime, entryPrice, exitTime, exitPrice, pnl, R, reason
   - download CSV button

## Data source
- `GET /api/v1/backtests/runs/{runId}`

## Acceptance criteria
- Selecting a run loads metrics and trades.
- No charts required for v1 (tables + numbers are sufficient).
