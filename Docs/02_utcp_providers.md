
# 02 – UTCP Providers and Tools

Goal: define the complete **go‑utcp tool layer** for Jax so that all higher‑level
code (Jax Core, Dexter integration, Lab, UI) talk only to clear, typed
interfaces instead of ad‑hoc HTTP calls.

This file is the blueprint for:

- `internal/infra/utcp/client.go`
- `internal/infra/utcp/*_service.go` wrappers
- Provider configuration file(s) under `config/`

---

## 1. UTCP Client Abstraction in Go

### 1.1 Core Client Responsibilities

Create a package `internal/infra/utcp` with:

- A `Client` interface, e.g.:

```go
type Client interface {
    CallTool(ctx context.Context, providerID, toolName string, input any, output any) error
}
```

- A concrete implementation `UTCPClient` that:
  - Loads provider definitions from config (e.g. JSON or YAML).
  - Knows how to route calls to:
    - Local tools.
    - HTTP‑based tools.
    - CLI‑based tools (optional, if needed later).

### 1.2 Configuration File

Create e.g. `config/providers.json` describing where each provider lives:

```json
{
  "providers": [
    {
      "id": "market-data",
      "transport": "http",
      "endpoint": "http://market-service:8080/tools"
    },
    {
      "id": "risk",
      "transport": "local"
    },
    {
      "id": "backtest",
      "transport": "local"
    },
    {
      "id": "broker",
      "transport": "http",
      "endpoint": "http://broker-sim:8080/tools"
    },
    {
      "id": "storage",
      "transport": "local"
    },
    {
      "id": "dexter",
      "transport": "http",
      "endpoint": "http://dexter:3000/tools"
    }
  ]
}
```

Codex should implement a small config loader that maps this into Go structs.

---

## 2. Provider: Market Data

**Provider ID:** `market-data`

Wrap in `internal/infra/utcp/market_service.go`:

```go
type MarketDataService struct {
    client Client
}

func NewMarketDataService(c Client) *MarketDataService { ... }
```

### 2.1 Tool: market.get_quote

Input:

```json
{ "symbol": "AAPL" }
```

Output:

```json
{
  "symbol": "AAPL",
  "price": 182.10,
  "currency": "USD",
  "timestamp": "2025-02-01T14:30:00Z"
}
```

### 2.2 Tool: market.get_candles

Input:

```json
{
  "symbol": "AAPL",
  "timeframe": "1D",
  "limit": 200
}
```

Output:

```json
{
  "symbol": "AAPL",
  "timeframe": "1D",
  "candles": [
    { "ts": "...", "open": 1, "high": 2, "low": 0.5, "close": 1.5, "volume": 12345 }
  ]
}
```

### 2.3 Tool: market.get_earnings

Input:

```json
{ "symbol": "AAPL", "limit": 8 }
```

Output (simplified):

```json
{
  "symbol": "AAPL",
  "earnings": [
    {
      "date": "2025-01-25",
      "eps_actual": 1.2,
      "eps_estimate": 1.0,
      "surprise_pct": 20.0
    }
  ]
}
```

---

## 3. Provider: Risk

**Provider ID:** `risk`

Implement as **local tools** initially (pure Go functions wrapped as UTCP tools).
Expose a typed service `RiskService` in `internal/infra/utcp/risk_service.go`.

### 3.1 Tool: risk.position_size

Input:

```json
{
  "accountSize": 10000,
  "riskPercent": 3,
  "entry": 100,
  "stop": 95
}
```

Output:

```json
{
  "positionSize": 60,
  "riskPerUnit": 5,
  "totalRisk": 300
}
```

### 3.2 Tool: risk.r_multiple

Input:

```json
{
  "entry": 100,
  "stop": 95,
  "target": 110
}
```

Output:

```json
{
  "rMultiple": 2.0
}
```

---

## 4. Provider: Backtest

**Provider ID:** `backtest`

Implementation can be pure Go, exposed as UTCP tools via the local transport.

### 4.1 Tool: backtest.run_strategy

Input:

```json
{
  "strategyConfigId": "earnings_gap_v1",
  "symbols": ["AAPL", "MSFT"],
  "from": "2020-01-01",
  "to": "2025-01-01"
}
```

Output (simplified):

```json
{
  "runId": "bt_20250101_001",
  "stats": {
    "trades": 124,
    "winRate": 0.57,
    "avgR": 1.4,
    "maxDrawdown": -0.12,
    "sharpe": 1.8
  }
}
```

### 4.2 Tool: backtest.get_run

Input:

```json
{ "runId": "bt_20250101_001" }
```

Output:

```json
{
  "runId": "bt_20250101_001",
  "stats": { ... },
  "bySymbol": [
    { "symbol": "AAPL", "trades": 40, "winRate": 0.6 },
    { "symbol": "MSFT", "trades": 84, "winRate": 0.55 }
  ]
}
```

---

## 5. Provider: Broker (sim / paper)

**Provider ID:** `broker`

Initially create a **simulated broker service**. Real broker integration can
come later.

### 5.1 Tool: broker.place_order

Input:

```json
{
  "symbol": "AAPL",
  "side": "buy",
  "quantity": 50,
  "orderType": "market"
}
```

Output:

```json
{
  "orderId": "sim_123",
  "status": "accepted"
}
```

### 5.2 Tool: broker.get_positions

Input: `{}`  
Output:

```json
{
  "positions": [
    { "symbol": "AAPL", "quantity": 50, "avgPrice": 101.2 }
  ]
}
```

---

## 6. Provider: Storage

**Provider ID:** `storage`

Can wrap SQLite/Postgres or JSON files. Expose as UTCP tools so lab + core
can both use it.

Useful tools:

- `storage.save_event`
- `storage.save_trade`
- `storage.get_trade`
- `storage.list_trades`

Define JSON contracts in a way that maps directly to Go domain types.

Example input for `storage.save_trade`:

```json
{
  "trade": {
    "id": "ts_001",
    "symbol": "AAPL",
    "direction": "long",
    "entry": 100,
    "stop": 95,
    "targets": [110, 115],
    "strategyId": "earnings_gap_v1"
  }
}
```

---

## 7. Provider: Dexter

The Dexter provider is fully defined in `03_dexter_integration.md`, but is
declared here for completeness.

**Provider ID:** `dexter`  
**Core tools:**

- `dexter.research_company`
- `dexter.compare_companies`

---

## 8. Tasks for Codex / AI

1. Implement `internal/infra/utcp/config.go` to parse `config/providers.json`.
2. Implement `internal/infra/utcp/client.go` with a `Client` interface and
   concrete implementation.
3. Implement typed service wrappers:
   - `MarketDataService`
   - `RiskService`
   - `BacktestService`
   - `BrokerService`
   - `StorageService`
   - `DexterService` (stub for now; fully defined in `03_dexter_integration.md`).
4. Write small integration tests using stub or mock providers where possible.
5. Ensure clear error paths:
   - Timeouts.
   - Invalid JSON from tools.
   - Missing provider/tool config.
