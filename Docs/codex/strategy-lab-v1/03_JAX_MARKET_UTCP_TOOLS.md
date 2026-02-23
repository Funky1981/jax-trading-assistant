# Implement UTCP Tools in jax-market

## Objective
Make `jax-market` a UTCP provider at `/tools` so `jax-api` and the backtest engine can request candles/quotes consistently.

## Required changes

### 1) Add `/tools` endpoint to `jax-market`
Location: `services/jax-market/cmd/jax-market/main.go`

Implement UTCP-compatible handler:
- Request JSON:
  - `{ "tool": "<toolName>", "input": { ... } }`
- Response JSON:
  - `{ "ok": true, "output": { ... } }`
  - `{ "ok": false, "error": "..." }`

### 2) Tool IDs
Align tool names with `libs/utcp` expectations used by `MarketDataService`:
- Provider ID: `market-data` (already in `config/providers.json`)
- Tools:
  - `market.get_quote`
  - `market.get_candles`
  - `market.get_earnings`

If these constants are scattered/missing, create a single file:
- `libs/utcp/tool_ids.go` (provider IDs + tool names)

### 3) Candle requirements (MUST)
Support:
- Timeframes: `1m`, `5m`, `15m`, `1h`, `1d` (minimum: `1m` and `1d`)
Input:
- symbol
- timeframe
- either:
  - `limit` (simple)
  - or `from` + `to` (preferred for backtest)
Output:
- candles array with:
  - ts (UTC)
  - open/high/low/close
  - volume

### 4) Update provider config
Update `config/providers.json`:

```json
{
  "id": "market-data",
  "transport": "http",
  "endpoint": "http://jax-market:8095/tools"
}
```

### 5) Acceptance criteria
- From `jax-api`, `MarketDataService.GetCandles()` succeeds against `jax-market`.
- Intraday candles return expected bar counts and UTC timestamps.
- Tool responses are stable and versioned (add `version` field if you want).

## Notes
- For backtesting, prefer pulling candles from Postgres first, and falling back to live provider if missing.
- Keep the UTCP tool handler thin; put logic in packages under `services/jax-market/internal/...`.
