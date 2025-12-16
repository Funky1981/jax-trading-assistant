
# 04 – Jax Core (Deterministic Trading Engine)

Goal: build the **main Go service** that runs the trading logic:

- Detects events.
- Generates rule‑based trade setups.
- Calculates risk and position size.
- Optionally enriches with Dexter research.
- Exposes an HTTP API for the React dashboard and for automation.

---

## 1. Project Layout (Backend)

Under the backend repo (root of this spec):

```text
jax-trading-assistant/
  cmd/
    jax-core/
      main.go
  internal/
    domain/
      event.go
      trade_setup.go
      risk_result.go
      strategy_config.go
    app/
      event_detector.go
      trade_generator.go
      risk_engine.go
      orchestrator.go
    infra/
      utcp/
        client.go
        market_service.go
        risk_service.go
        storage_service.go
        backtest_service.go
        dexter_service.go
      http/
        server.go
        handlers_events.go
        handlers_trades.go
        handlers_backtest.go
    config/
      providers.json
      strategies/
        earnings_gap_v1.json
        volatility_spike_v1.json
```

Codex should create these files where relevant while working through this
spec, using previous specs for UTCP and Dexter.

---

## 2. Domain Layer

### 2.1 Event

```go
package domain

type EventType string

const (
    EventEarningsSurprise EventType = "earnings_surprise"
    EventGapOpen          EventType = "gap_open"
    EventVolumeSpike      EventType = "volume_spike"
)

type Event struct {
    ID       string                 `json:"id"`
    Symbol   string                 `json:"symbol"`
    Type     EventType              `json:"type"`
    Time     time.Time              `json:"time"`
    Payload  map[string]any         `json:"payload"` // e.g. surprise %, gap %, etc.
}
```

### 2.2 TradeSetup

```go
type TradeDirection string

const (
    Long  TradeDirection = "long"
    Short TradeDirection = "short"
)

type TradeSetup struct {
    ID         string         `json:"id"`
    Symbol     string         `json:"symbol"`
    Direction  TradeDirection `json:"direction"`
    Entry      float64        `json:"entry"`
    Stop       float64        `json:"stop"`
    Targets    []float64      `json:"targets"`
    EventID    string         `json:"eventId"`
    StrategyID string         `json:"strategyId"`
    Notes      string         `json:"notes"`

    Research   *ResearchBundle `json:"research,omitempty"`
}
```

`ResearchBundle` is defined in the Dexter integration spec.

### 2.3 RiskResult

```go
type RiskResult struct {
    PositionSize int     `json:"positionSize"`
    RiskPerUnit  float64 `json:"riskPerUnit"`
    TotalRisk    float64 `json:"totalRisk"`
    RMultiple    float64 `json:"rMultiple"`
}
```

### 2.4 StrategyConfig

Store as JSON under `config/strategies/`:

```go
type StrategyConfig struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    EventTypes     []string `json:"eventTypes"`
    MinRR          float64  `json:"minRR"`
    MaxRiskPercent float64  `json:"maxRiskPercent"`

    EntryRule      string   `json:"entryRule"`   // descriptive rule string
    StopRule       string   `json:"stopRule"`    // parse/interpret in Go
    TargetRule     string   `json:"targetRule"`  // e.g. "2R,3R"
}
```

Codex should also add a loader that reads all `*.json` strategy files into
memory at startup.

---

## 3. Application Layer

### 3.1 EventDetector

File: `internal/app/event_detector.go`

Responsibilities:

- Use `MarketDataService` and (optionally) earnings tools to produce **Event**
  instances per symbol.
- Implement simple rules such as:
  - Earnings surprise > threshold.
  - Gap % between previous close and open > threshold.
  - Volume spike relative to average.

Pseudo‑API:

```go
type EventDetector struct {
    market *utcp.MarketDataService
    // thresholds or strategy config references
}

func (d *EventDetector) DetectEarningsSurprises(ctx context.Context, symbol string) ([]domain.Event, error) { ... }
func (d *EventDetector) DetectGaps(ctx context.Context, symbol string) ([]domain.Event, error) { ... }
func (d *EventDetector) DetectVolumeSpikes(ctx context.Context, symbol string) ([]domain.Event, error) { ... }
```

### 3.2 TradeGenerator

File: `internal/app/trade_generator.go`

Responsibilities:

- For a given Event + StrategyConfig:
  - Decide direction (long/short).
  - Compute entry/stop/targets based on recent candles.
- Uses `MarketDataService` to fetch recent price history if needed.

Pseudo‑API:

```go
type TradeGenerator struct {
    market     *utcp.MarketDataService
    strategies map[string]domain.StrategyConfig
}

func (g *TradeGenerator) GenerateFromEvent(ctx context.Context, e domain.Event) ([]domain.TradeSetup, error) {
    // 1. find strategies that include e.Type
    // 2. for each strategy, compute a TradeSetup
}
```

### 3.3 RiskEngine

File: `internal/app/risk_engine.go`

Responsibilities:

- Take TradeSetup + risk settings (account size, risk %).  
- Use `RiskService` via UTCP to calculate `RiskResult`.

```go
type RiskEngine struct {
    risk *utcp.RiskService
}

func (r *RiskEngine) Evaluate(ctx context.Context, setup domain.TradeSetup, accountSize float64) (domain.RiskResult, error) {
    // call risk.position_size and risk.r_multiple as needed
}
```

### 3.4 Orchestrator

File: `internal/app/orchestrator.go`

Responsibilities:

- Glue together EventDetector, TradeGenerator, RiskEngine, Storage, Dexter.

Pseudo‑flow for one symbol:

```go
func (o *Orchestrator) ProcessSymbol(ctx context.Context, symbol string, accountSize float64) error {
    events, err := o.detector.DetectAll(ctx, symbol)
    if err != nil { return err }

    for _, e := range events {
        setups, err := o.generator.GenerateFromEvent(ctx, e)
        if err != nil { continue }

        for _, s := range setups {
            risk, err := o.riskEngine.Evaluate(ctx, s, accountSize)
            if err != nil { continue }

            if o.dexter != nil {
                bundle, err := o.dexter.ResearchCompany(ctx, s.Symbol, []string{
                    "Summarise the last 4 quarters of earnings.",
                    "Highlight key risks and catalysts for this trade idea.",
                })
                if err == nil {
                    s.Research = bundle
                }
            }

            if err := o.storage.SaveTrade(ctx, s, risk, &e); err != nil {
                return err
            }
        }
    }
    return nil
}
```

---

## 4. Infrastructure: HTTP API

File: `internal/infra/http/server.go` and handlers.

### 4.1 Endpoints

Suggested core endpoints:

- `GET /health`  
  Simple health check.

- `POST /symbols/{symbol}/process`  
  - Triggers detection + setup generation for a symbol.
  - Body can include `accountSize` and optional strategy filters.

- `GET /events`  
  - Query params: `symbol`, `type`, `limit`.

- `GET /trades`  
  - Query params: `symbol`, `strategyId`, `limit`.

- `GET /trades/{id}`  
  - Returns full trade including risk + research if available.

- `POST /risk/calc`  
  - Quick risk calculator endpoint for the UI.

- `POST /backtest/run`  
  - Thin wrapper around `backtest.run_strategy` UTCP tool.

### 4.2 Handler Layout

Example file structure:

```text
internal/infra/http/
  server.go
  handlers_events.go
  handlers_trades.go
  handlers_backtest.go
  handlers_risk.go
```

Each handler:

- Accepts/validates JSON.
- Calls application services (or UTCP via services).
- Returns JSON with clear error messages when needed.

---

## 5. Configuration

Add a simple config struct:

```go
type Config struct {
    HTTPPort       int      `json:"httpPort"`
    DefaultSymbols []string `json:"defaultSymbols"`
    AccountSize    float64  `json:"accountSize"`
    UseDexter      bool     `json:"useDexter"`
}
```

Load from `config/jax-core.json` or environment variables.

---

## 6. Tasks for Codex / AI

1. Create the `cmd/jax-core/main.go` entrypoint:
   - Load config.
   - Initialise UTCP client and service wrappers.
   - Initialise EventDetector, TradeGenerator, RiskEngine, Orchestrator.
   - Start HTTP server.

2. Implement all **domain models** and ensure they serialise cleanly to/from JSON.

3. Implement **EventDetector** with at least one working rule
   (e.g. gap open or earnings surprise).

4. Implement **TradeGenerator** for one starter strategy
   (`earnings_gap_v1.json`).

5. Implement **RiskEngine** using the UTCP `risk` provider.

6. Implement **StorageService** using SQLite or Postgres with a minimal schema.

7. Implement core HTTP handlers for:
   - `/symbols/{symbol}/process`
   - `/trades`
   - `/trades/{id}`
   - `/risk/calc`

8. Add unit tests around:
   - Event detection.
   - Trade generation.
   - Risk evaluation.
   - HTTP handler happy path for one endpoint.
