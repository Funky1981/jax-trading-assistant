# Autonomous Trading System - Implementation Roadmap

**Project:** Jax Trading Assistant - Autonomous Mode  
**Goal:** Transform from passive chatbot to active market monitoring & trade recommendation system  
**Timeline:** 12-16 weeks  
**Status:** Planning Phase

---

## 🎯 Mission Statement

Enable Jax to:

1. **Monitor** markets and news 24/7 autonomously
2. **Detect** trading opportunities using technical + fundamental signals
3. **Generate** trade recommendations with AI analysis
4. **Present** trades to user for approval via UI
5. **Execute** approved trades automatically via Interactive Brokers
6. **Learn** from outcomes to improve future recommendations

---

## 📊 Current State Summary

| Component | Status | Completeness |
| --------- | ------ | ------------ |
| Market Data Ingestion | Code exists, not deployed | 60% |
| Strategy Signals | Implemented, not automated | 70% |
| Agent0 AI Service | Running, reactive only | 80% |
| IB Trade Execution | Fully working | 100% |
| Memory & Learning | Working, no reflection loop | 75% |
| Orchestrator | CLI only, no API | 50% |
| Frontend | UI ready, APIs missing | 65% |
| **OVERALL** | **Passive System** | **70%** |

**Gap to Autonomous:** Missing automation layer, signal pipeline, and recommendation queue.

---

## 🗺️ Implementation Phases

### **Phase 1: Foundation & Data Pipeline (Week 1-2)**

**Goal:** Get market data flowing continuously into the database

#### 1.1 Deploy Market Data Service

- [x] Review `services/jax-market/` implementation
- [ ] Add `jax-market` to docker-compose.yml default profile
- [ ] Configure ingestion intervals (start: 60s, optimize later)
- [ ] Create watchlist configuration file
- [ ] Test continuous data ingestion for 5-10 symbols

**Files to modify:**

- `docker-compose.yml`
- `config/market-watchlist.json` (new)
- `services/jax-market/config.yaml`

**Acceptance Criteria:**

- ✅ jax-market service starts with `docker compose up`
- ✅ Market data appears in `market_data` table every 60s
- ✅ Health check passes: `curl http://localhost:8095/health`
- ✅ Logs show successful ingestion for all watchlist symbols

#### 1.2 Create Market Data Monitoring Dashboard

- [ ] Add metrics endpoint to jax-market
- [ ] Track: quotes ingested, errors, latency, provider health
- [ ] Create Grafana dashboard (optional but recommended)
- [ ] Add data freshness alerts

**Deliverables:**

- `GET /metrics` endpoint on jax-market
- Dashboard showing real-time data flow

#### 1.3 Database Schema Updates

- [ ] Create migration `000004_signals_and_runs.up.sql`
- [ ] Add `strategy_signals` table
- [ ] Add `orchestration_runs` table
- [ ] Add `trade_approvals` table
- [ ] Add indexes for signal queries

**Schema Design:**

```sql
-- Strategy signals (pending recommendations)
CREATE TABLE strategy_signals (
    id UUID PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    strategy_id VARCHAR(50) NOT NULL,
    signal_type VARCHAR(10) NOT NULL, -- BUY, SELL, HOLD
    confidence DECIMAL(3,2) NOT NULL, -- 0.00-1.00
    entry_price DECIMAL(12,2),
    stop_loss DECIMAL(12,2),
    take_profit DECIMAL(12,2),
    reasoning TEXT,
    generated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'pending', -- pending, approved, rejected, expired
    orchestration_run_id UUID,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Orchestration runs (AI analysis tracking)
CREATE TABLE orchestration_runs (
    id UUID PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    trigger_type VARCHAR(50), -- signal, scheduled, manual
    trigger_id UUID,
    agent_suggestion TEXT,
    confidence DECIMAL(3,2),
    reasoning TEXT,
    memories_recalled INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'running',
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    error TEXT
);

-- Trade approvals (user decisions)
CREATE TABLE trade_approvals (
    id UUID PRIMARY KEY,
    signal_id UUID REFERENCES strategy_signals(id),
    orchestration_run_id UUID REFERENCES orchestration_runs(id),
    approved BOOLEAN NOT NULL,
    approved_at TIMESTAMP NOT NULL,
    approved_by VARCHAR(100),
    modification_notes TEXT,
    order_id VARCHAR(100) -- IB order ID if approved
);
```

**Run migration:**

```bash
psql $JAX_POSTGRES_DSN -f db/postgres/migrations/000004_signals_and_runs.up.sql
```

**Acceptance Criteria:**

- ✅ All tables created successfully
- ✅ Foreign key constraints work
- ✅ Indexes created for common queries
- ✅ Can insert/query test records

---

### **Phase 2: Signal Generation Pipeline (Week 2-3)**

**Goal:** Automatically run strategies on market data and store signals

#### 2.1 Background Signal Generator Service

- [ ] Create `services/jax-signal-generator/`
- [ ] Implement scheduler (every 5 minutes)
- [ ] For each watchlist symbol:
  - Fetch latest market data
  - Run all active strategies (MACD, RSI, MA)
  - Generate signals with confidence scores
  - Store in `strategy_signals` table
- [ ] Add deduplication (don't create duplicate signals)

**File Structure:**

```text
services/jax-signal-generator/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── generator.go      # Core signal generation logic
│   │   └── scheduler.go      # Periodic execution
│   ├── domain/
│   │   └── signal.go         # Signal entity
│   └── infra/
│       ├── postgres_signals.go  # Signal repository
│       └── market_client.go     # Client for jax-market
├── config.yaml
└── Dockerfile
```

**Implementation:**

```go
// services/jax-signal-generator/internal/app/generator.go
type SignalGenerator struct {
    marketClient  MarketDataClient
    signalRepo    SignalRepository
    strategyReg   *strategies.Registry
}

func (g *SignalGenerator) GenerateSignals(ctx context.Context) error {
    symbols := g.getWatchlist()
    
    for _, symbol := range symbols {
        // Get recent market data
        data, err := g.marketClient.GetRecentCandles(ctx, symbol, 100)
        if err != nil {
            log.Warn("Failed to fetch data", "symbol", symbol, "error", err)
            continue
        }
        
        // Run all strategies
        for _, strategy := range g.strategyReg.All() {
            result := strategy.Analyze(ctx, data)
            
            // Only store actionable signals (not HOLD with low confidence)
            if result.Signal != "HOLD" || result.Confidence > 0.7 {
                signal := domain.Signal{
                    ID:         uuid.New(),
                    Symbol:     symbol,
                    StrategyID: strategy.ID(),
                    SignalType: result.Signal,
                    Confidence: result.Confidence,
                    EntryPrice: result.Entry,
                    StopLoss:   result.Stop,
                    TakeProfit: result.Targets[0],
                    Reasoning:  result.Reasoning,
                    GeneratedAt: time.Now(),
                    ExpiresAt:  time.Now().Add(24 * time.Hour),
                }
                
                // Check for duplicate (same symbol+strategy+type in last hour)
                if !g.signalRepo.HasRecentSignal(ctx, signal) {
                    g.signalRepo.Store(ctx, signal)
                }
            }
        }
    }
    
    return nil
}
```

#### 2.2 Signal API Endpoints

- [ ] Add to `services/jax-api/`:
  - `GET /api/v1/signals` - List pending signals
  - `GET /api/v1/signals/{id}` - Get signal details
  - `POST /api/v1/signals/{id}/approve` - Approve signal
  - `POST /api/v1/signals/{id}/reject` - Reject signal
  - `DELETE /api/v1/signals/{id}` - Cancel signal

**Add to jax-api handlers:**

```go
// services/jax-api/internal/infra/http/handlers_signals.go
func (h *SignalHandlers) ListSignals(w http.ResponseWriter, r *http.Request) {
    status := r.URL.Query().Get("status") // pending, approved, rejected, all
    limit := parseInt(r.URL.Query().Get("limit"), 50)
    
    signals, err := h.signalRepo.List(r.Context(), status, limit)
    if err != nil {
        respondError(w, err)
        return
    }
    
    respondJSON(w, signals)
}

func (h *SignalHandlers) ApproveSignal(w http.ResponseWriter, r *http.Request) {
    signalID := chi.URLParam(r, "id")
    
    var req struct {
        Notes string `json:"notes"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Mark signal approved
    signal, err := h.signalRepo.Get(r.Context(), signalID)
    if err != nil {
        respondError(w, err)
        return
    }
    
    // Store approval decision
    approval := domain.TradeApproval{
        SignalID:    signal.ID,
        Approved:    true,
        ApprovedAt:  time.Now(),
        ApprovedBy:  getUserFromContext(r.Context()),
        Notes:       req.Notes,
    }
    h.approvalRepo.Store(r.Context(), approval)
    
    // Update signal status
    signal.Status = "approved"
    h.signalRepo.Update(r.Context(), signal)
    
    // Trigger trade execution (Phase 4)
    h.executeApprovedSignal(r.Context(), signal, approval)
    
    respondJSON(w, approval)
}
```

#### 2.3 Signal Expiration Job

- [ ] Create cleanup job (runs every hour)
- [ ] Mark expired signals (older than 24h) as `expired`
- [ ] Archive old signals to separate table

**Acceptance Criteria:**

- ✅ Signal generator runs every 5 minutes
- ✅ Signals appear in database for all watchlist symbols
- ✅ API returns pending signals via `GET /api/v1/signals`
- ✅ No duplicate signals created for same setup
- ✅ Expired signals cleaned up automatically

---

### **Phase 3: Orchestrator HTTP API (Week 3-4)**

**Goal:** Enable orchestration to be triggered via HTTP (not just CLI)

#### 3.1 Create Orchestrator Server

- [ ] Create `services/jax-orchestrator/cmd/server/main.go`
- [ ] Implement HTTP endpoints:
  - `POST /api/v1/orchestrate` - Trigger orchestration
  - `GET /api/v1/orchestrate/runs/{id}` - Get run status
  - `GET /api/v1/orchestrate/runs` - List recent runs
  - `GET /health` - Health check

**Implementation:**

```go
// services/jax-orchestrator/cmd/server/main.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "github.com/go-chi/chi/v5"
    "jax-trading-assistant/libs/agent0"
    "jax-trading-assistant/libs/utcp"
    "jax-trading-assistant/services/jax-orchestrator/internal/app"
)

type Server struct {
    orchestrator *app.Orchestrator
    runRepo      RunRepository
}

type OrchestrationRequest struct {
    Symbol      string   `json:"symbol"`
    TriggerType string   `json:"trigger_type"` // signal, manual, scheduled
    TriggerID   string   `json:"trigger_id,omitempty"`
    Context     string   `json:"context,omitempty"`
}

func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
    var req OrchestrationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    
    // Create orchestration run record
    run := &OrchestrationRun{
        ID:          uuid.New(),
        Symbol:      req.Symbol,
        TriggerType: req.TriggerType,
        TriggerID:   req.TriggerID,
        Status:      "running",
        StartedAt:   time.Now(),
    }
    s.runRepo.Create(r.Context(), run)
    
    // Execute orchestration asynchronously
    go func() {
        ctx := context.Background()
        result, err := s.orchestrator.Execute(ctx, app.OrchestrationInput{
            Symbol:      req.Symbol,
            TriggerType: req.TriggerType,
        })
        
        if err != nil {
            run.Status = "failed"
            run.Error = err.Error()
        } else {
            run.Status = "completed"
            run.AgentSuggestion = result.Plan.Summary
            run.Confidence = result.Plan.Confidence
            run.Reasoning = result.Plan.Reasoning
            run.MemoriesRecalled = len(result.RecalledMemories)
        }
        run.CompletedAt = time.Now()
        s.runRepo.Update(ctx, run)
    }()
    
    // Return run ID immediately
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "run_id": run.ID,
        "status": "running",
        "symbol": req.Symbol,
    })
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
    runID := chi.URLParam(r, "id")
    
    run, err := s.runRepo.Get(r.Context(), runID)
    if err != nil {
        http.Error(w, "Run not found", 404)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(run)
}

func main() {
    // Initialize dependencies (similar to CLI but as HTTP service)
    memoryClient := utcp.NewMemoryService(/* config */)
    agent0Client := agent0.NewClient(/* config */)
    
    orchestrator := app.NewOrchestrator(
        memoryClient,
        agent0Client,
        /* other deps */
    )
    
    server := &Server{
        orchestrator: orchestrator,
        runRepo:      NewPostgresRunRepository(),
    }
    
    r := chi.NewRouter()
    r.Post("/api/v1/orchestrate", server.handleOrchestrate)
    r.Get("/api/v1/orchestrate/runs/{id}", server.handleGetRun)
    r.Get("/api/v1/orchestrate/runs", server.handleListRuns)
    r.Get("/health", server.handleHealth)
    
    log.Println("Orchestrator server starting on :8093")
    http.ListenAndServe(":8093", r)
}
```

#### 3.2 Update Docker Compose

- [ ] Remove orchestrator from `profiles: [jobs]`
- [ ] Add as default service with HTTP server
- [ ] Configure environment variables
- [ ] Add health check

**Update docker-compose.yml:**

```yaml
  jax-orchestrator:
    build:
      context: .
      dockerfile: services/jax-orchestrator/Dockerfile.server
    ports:
      - "8093:8093"
    environment:
      - AGENT0_URL=http://agent0-api:8094
      - MEMORY_URL=http://jax-memory:8090
      - POSTGRES_DSN=${JAX_POSTGRES_DSN}
    depends_on:
      - postgres
      - agent0-api
      - jax-memory
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8093/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

**Acceptance Criteria:**

- ✅ Orchestrator starts as HTTP service
- ✅ `POST /api/v1/orchestrate` triggers analysis
- ✅ Returns run ID immediately (async execution)
- ✅ `GET /api/v1/orchestrate/runs/{id}` shows progress
- ✅ Orchestration run stored in database
- ✅ Agent0 called successfully
- ✅ Memories recalled and retained

---

### **Phase 4: Autonomous Signal-to-Orchestration Pipeline (Week 4-5)**

**Goal:** Connect signal generation → auto-trigger orchestration → create recommendations

#### 4.1 Signal-Triggered Orchestration

- [ ] When high-confidence signal created (confidence > 0.75):
  - Auto-trigger orchestration
  - Pass signal context to Agent0
  - Store orchestration run linked to signal
- [ ] Configure thresholds (which signals trigger AI analysis)

**Implementation:**

```go
// services/jax-signal-generator/internal/app/orchestrator_trigger.go
func (g *SignalGenerator) handleHighConfidenceSignal(ctx context.Context, signal domain.Signal) {
    // Only trigger orchestration for strong signals
    if signal.Confidence < 0.75 {
        return
    }
    
    // Call orchestrator
    resp, err := g.orchestratorClient.Trigger(ctx, OrchestrateRequest{
        Symbol:      signal.Symbol,
        TriggerType: "signal",
        TriggerID:   signal.ID.String(),
        Context:     fmt.Sprintf("Strategy %s detected %s signal with %.0f%% confidence",
            signal.StrategyID, signal.SignalType, signal.Confidence*100),
    })
    
    if err != nil {
        log.Error("Failed to trigger orchestration", "error", err)
        return
    }
    
    // Link orchestration run to signal
    signal.OrchestrationRunID = &resp.RunID
    g.signalRepo.Update(ctx, signal)
}
```

#### 4.2 Enhanced Agent0 Prompts

- [ ] Update Agent0 prompts to include signal context
- [ ] Add signal metadata to Agent0 requests
- [ ] Ensure Agent0 considers:
  - Technical signal (from strategy)
  - Historical performance of similar setups
  - Current market conditions
  - Risk parameters

**Update orchestrator:**

```go
// services/jax-orchestrator/internal/app/orchestrator.go
func (o *Orchestrator) buildAgent0Context(signal *domain.Signal, memories []contracts.MemoryItem) string {
    ctx := fmt.Sprintf(`You are analyzing a trading opportunity:

Symbol: %s
Signal Type: %s from %s strategy
Confidence: %.0f%%
Entry: $%.2f
Stop Loss: $%.2f
Take Profit: $%.2f
Strategy Reasoning: %s

Recent similar outcomes:
`, signal.Symbol, signal.SignalType, signal.StrategyID, 
   signal.Confidence*100, signal.EntryPrice, signal.StopLoss, signal.TakeProfit,
   signal.Reasoning)
    
    for _, mem := range memories {
        ctx += fmt.Sprintf("- %s\n", mem.Summary)
    }
    
    ctx += `
Based on this signal and historical context, should we take this trade?
Provide:
1. Your recommendation (BUY/SELL/PASS)
2. Confidence score (0.0-1.0)
3. Reasoning (what makes this setup strong or weak?)
4. Any adjustments to entry/stop/target
`
    return ctx
}
```

#### 4.3 Recommendation Queue

- [ ] Create UI component for pending recommendations
- [ ] Show:
  - Technical signal details
  - AI analysis & reasoning
  - Risk metrics (R:R ratio, account % risk)
  - One-click approve/reject buttons

**Acceptance Criteria:**

- ✅ High-confidence signals auto-trigger orchestration
- ✅ Agent0 receives full signal context
- ✅ Orchestration run linked to originating signal
- ✅ Recommendation appears in queue (database + API)
- ✅ User can see which signals triggered AI analysis

---

### **Phase 5: Trade Execution Automation (Week 5-6)**

**Goal:** Approved signals automatically execute via IB Bridge

#### 5.1 Trade Execution Service

- [ ] Create execution handler in jax-api
- [ ] On signal approval:
  - Calculate position size (based on risk % and stop loss)
  - Create IB order (limit/market based on config)
  - Submit to IB Bridge
  - Store order ID with approval record
  - Monitor order status

**Implementation:**

```go
// services/jax-api/internal/app/trade_executor.go
type TradeExecutor struct {
    ibBridge     IBBridgeClient
    riskCalc     RiskCalculator
    tradeRepo    TradeRepository
    signalRepo   SignalRepository
}

func (e *TradeExecutor) ExecuteApprovedSignal(ctx context.Context, signal domain.Signal, approval domain.TradeApproval) error {
    // Get account info
    account, err := e.ibBridge.GetAccount(ctx)
    if err != nil {
        return fmt.Errorf("failed to get account: %w", err)
    }
    
    // Calculate position size (e.g., risk 1% of account on this trade)
    accountValue := account.NetLiquidation
    riskAmount := accountValue * 0.01 // 1% risk
    stopDistance := math.Abs(signal.EntryPrice - signal.StopLoss)
    shares := int(riskAmount / stopDistance)
    
    // Create IB order
    order := IBOrder{
        Symbol:      signal.Symbol,
        Action:      signal.SignalType, // BUY or SELL
        Quantity:    shares,
        OrderType:   "LMT", // Limit order
        LimitPrice:  signal.EntryPrice,
        Transmit:    true,
    }
    
    // Submit to IB
    orderResp, err := e.ibBridge.PlaceOrder(ctx, order)
    if err != nil {
        return fmt.Errorf("failed to place order: %w", err)
    }
    
    // Store trade record
    trade := domain.Trade{
        ID:             uuid.New(),
        Symbol:         signal.Symbol,
        Direction:      signal.SignalType,
        EntryPrice:     signal.EntryPrice,
        StopLoss:       signal.StopLoss,
        Targets:        []float64{signal.TakeProfit},
        Quantity:       shares,
        StrategyID:     signal.StrategyID,
        SignalID:       &signal.ID,
        IBOrderID:      orderResp.OrderID,
        Status:         "pending",
        SubmittedAt:    time.Now(),
    }
    e.tradeRepo.Create(ctx, trade)
    
    // Update approval with order ID
    approval.OrderID = orderResp.OrderID
    e.approvalRepo.Update(ctx, approval)
    
    return nil
}
```

#### 5.2 Order Status Monitoring

- [ ] Create background job to poll IB for order status
- [ ] Update trade records when:
  - Order filled → update entry price, status = "open"
  - Order cancelled → status = "cancelled"
  - Order rejected → status = "rejected"
- [ ] Send notifications on status changes

#### 5.3 Position Management

- [ ] Track open positions
- [ ] Monitor for stop loss / take profit hits
- [ ] Auto-close positions when targets reached
- [ ] Store exit details in trades table

**Acceptance Criteria:**

- ✅ Approved signal calculates position size correctly
- ✅ Order submitted to IB Bridge
- ✅ Order ID stored with trade record
- ✅ Order status updates automatically
- ✅ Filled orders show correct entry price
- ✅ Positions tracked in database

---

### **Phase 6: Frontend Integration (Week 6-7)**

**Goal:** User sees autonomous recommendations in UI and can approve/reject

#### 6.1 Signals Dashboard Component

- [ ] Create `SignalsQueue` component
- [ ] Show pending signals with:
  - Symbol, strategy, confidence
  - Entry/stop/target levels
  - AI reasoning (if orchestration completed)
  - Time remaining before expiration
  - Approve/Reject buttons

**Implementation:**

```typescript
// frontend/src/components/dashboard/SignalsQueuePanel.tsx
import { useSignals } from '@/hooks/useSignals';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { TrendingUp, TrendingDown, Clock, Brain } from 'lucide-react';

export function SignalsQueuePanel() {
  const { signals, approve, reject, isLoading } = useSignals({ status: 'pending' });
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>Trading Opportunities</CardTitle>
        <CardDescription>
          {signals?.length || 0} pending recommendations
        </CardDescription>
      </CardHeader>
      <CardContent>
        {signals?.map(signal => (
          <SignalCard
            key={signal.id}
            signal={signal}
            onApprove={() => approve(signal.id)}
            onReject={() => reject(signal.id)}
          />
        ))}
      </CardContent>
    </Card>
  );
}

function SignalCard({ signal, onApprove, onReject }) {
  const timeRemaining = formatTimeRemaining(signal.expiresAt);
  const riskReward = calculateRR(signal.entryPrice, signal.stopLoss, signal.takeProfit);
  
  return (
    <div className="border rounded-lg p-4 mb-4">
      <div className="flex justify-between items-start mb-2">
        <div>
          <h3 className="font-bold text-lg">{signal.symbol}</h3>
          <p className="text-sm text-muted-foreground">
            {signal.strategyId} • {signal.signalType}
          </p>
        </div>
        <Badge variant={signal.confidence > 0.8 ? 'default' : 'secondary'}>
          {(signal.confidence * 100).toFixed(0)}% confidence
        </Badge>
      </div>
      
      {signal.orchestrationRun && (
        <div className="bg-muted p-3 rounded mb-3">
          <div className="flex items-start gap-2">
            <Brain className="w-4 h-4 mt-1" />
            <div>
              <p className="text-sm font-medium">AI Analysis</p>
              <p className="text-sm">{signal.orchestrationRun.reasoning}</p>
            </div>
          </div>
        </div>
      )}
      
      <div className="grid grid-cols-3 gap-4 mb-3">
        <div>
          <p className="text-xs text-muted-foreground">Entry</p>
          <p className="font-medium">${signal.entryPrice.toFixed(2)}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Stop</p>
          <p className="font-medium text-red-600">${signal.stopLoss.toFixed(2)}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Target</p>
          <p className="font-medium text-green-600">${signal.takeProfit.toFixed(2)}</p>
        </div>
      </div>
      
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm">Risk:Reward {riskReward}</span>
        <span className="text-sm text-muted-foreground flex items-center gap-1">
          <Clock className="w-3 h-3" />
          {timeRemaining}
        </span>
      </div>
      
      <div className="flex gap-2">
        <Button 
          onClick={onApprove}
          className="flex-1"
          variant="default"
        >
          Approve Trade
        </Button>
        <Button 
          onClick={onReject}
          className="flex-1"
          variant="outline"
        >
          Reject
        </Button>
      </div>
    </div>
  );
}
```

#### 6.2 Create Signals Hook

- [ ] Implement `useSignals` hook
- [ ] Auto-refresh every 10 seconds
- [ ] Optimistic updates on approve/reject
- [ ] WebSocket support (future enhancement)

```typescript
// frontend/src/hooks/useSignals.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { signalsService } from '@/data/signals-service';

export function useSignals({ status = 'pending' } = {}) {
  const queryClient = useQueryClient();
  
  const query = useQuery({
    queryKey: ['signals', status],
    queryFn: () => signalsService.list({ status, limit: 50 }),
    refetchInterval: 10000, // Refresh every 10s
  });
  
  const approveMutation = useMutation({
    mutationFn: (signalId: string) => signalsService.approve(signalId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signals'] });
    },
  });
  
  const rejectMutation = useMutation({
    mutationFn: (signalId: string) => signalsService.reject(signalId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signals'] });
    },
  });
  
  return {
    signals: query.data,
    isLoading: query.isLoading,
    error: query.error,
    approve: approveMutation.mutate,
    reject: rejectMutation.mutate,
  };
}
```

#### 6.3 Add to Dashboard

- [ ] Add `SignalsQueuePanel` to main dashboard
- [ ] Add notification badge when new signals arrive
- [ ] Add browser notifications (optional)
- [ ] Add sound alerts for high-confidence signals

**Acceptance Criteria:**

- ✅ Frontend polls for new signals every 10s
- ✅ Signals display with all relevant info
- ✅ AI reasoning visible when available
- ✅ Approve button triggers trade execution
- ✅ Reject button marks signal as rejected
- ✅ Optimistic UI updates (no page refresh needed)

---

### **Phase 7: News & Event Integration (Week 7-9)**

**Goal:** Add fundamental analysis - earnings, news, economic events

#### 7.1 News API Integration

- [ ] Choose news provider (Alpha Vantage, Benzinga, NewsAPI)
- [ ] Create `services/jax-news/`
- [ ] Implement news fetching:
  - Symbol-specific news
  - Market-wide news
  - Earnings announcements
  - Economic calendar
- [ ] Store in database with sentiment analysis

**Schema:**

```sql
CREATE TABLE news_events (
    id UUID PRIMARY KEY,
    source VARCHAR(50),
    symbol VARCHAR(10),
    headline TEXT NOT NULL,
    summary TEXT,
    url TEXT,
    published_at TIMESTAMP NOT NULL,
    sentiment VARCHAR(20), -- positive, negative, neutral
    sentiment_score DECIMAL(3,2),
    relevance_score DECIMAL(3,2),
    fetched_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_news_symbol_published ON news_events(symbol, published_at DESC);
```

#### 7.2 News-Based Signal Detection

- [ ] Create `NewsSignalDetector` strategy
- [ ] Trigger on:
  - Earnings beat/miss
  - Major news events (M&A, FDA approval, etc.)
  - Unusual volume + positive news
- [ ] Generate signals with news context

**Implementation:**

```go
// libs/strategies/news_catalyst.go
type NewsCatalystStrategy struct {
    newsClient NewsClient
}

func (s *NewsCatalystStrategy) Analyze(ctx context.Context, symbol string) (*StrategyResult, error) {
    // Get recent news (last 24 hours)
    news, err := s.newsClient.GetRecent(ctx, symbol, 24*time.Hour)
    if err != nil {
        return nil, err
    }
    
    // Check for significant positive catalyst
    hasPositiveCatalyst := false
    catalystScore := 0.0
    
    for _, article := range news {
        if article.SentimentScore > 0.7 && article.RelevanceScore > 0.8 {
            hasPositiveCatalyst = true
            catalystScore = math.Max(catalystScore, article.SentimentScore)
        }
    }
    
    if !hasPositiveCatalyst {
        return &StrategyResult{Signal: "HOLD", Confidence: 0.0}, nil
    }
    
    // Check if price is reacting (volume spike)
    volumeData := s.getVolumeData(ctx, symbol)
    avgVolume := calculateAvgVolume(volumeData)
    currentVolume := volumeData[len(volumeData)-1]
    
    volumeSpike := float64(currentVolume) / float64(avgVolume)
    
    if volumeSpike > 2.0 { // Volume > 2x average
        return &StrategyResult{
            Signal:     "BUY",
            Confidence: catalystScore * 0.9, // High confidence on news + volume
            Reasoning:  fmt.Sprintf("Positive news catalyst with %.1fx volume spike", volumeSpike),
        }, nil
    }
    
    return &StrategyResult{Signal: "HOLD", Confidence: 0.5}, nil
}
```

#### 7.3 Integrate News into Agent0

- [ ] Update Agent0 prompts to include recent news
- [ ] Provide news context in orchestration
- [ ] AI considers news sentiment in recommendations

**Acceptance Criteria:**

- ✅ News fetched for watchlist symbols
- ✅ News stored with sentiment scores
- ✅ News-based signals generated
- ✅ Agent0 receives news context
- ✅ Frontend shows news in signal cards

---

### **Phase 8: Learning & Reflection System (Week 9-10)**

**Goal:** Implement automated reflection to improve over time

#### 8.1 Trade Outcome Tracking

- [ ] Update trades table when positions close
- [ ] Calculate:
  - P&L ($ and %)
  - Hold time
  - Max favorable/adverse excursion
  - Win/loss outcome
- [ ] Link outcome to originating signal + strategy

**Schema update:**

```sql
ALTER TABLE trades ADD COLUMN exit_price DECIMAL(12,2);
ALTER TABLE trades ADD COLUMN exit_time TIMESTAMP;
ALTER TABLE trades ADD COLUMN pnl DECIMAL(12,2);
ALTER TABLE trades ADD COLUMN pnl_percent DECIMAL(5,2);
ALTER TABLE trades ADD COLUMN outcome VARCHAR(10); -- win, loss, breakeven
ALTER TABLE trades ADD COLUMN hold_duration INTERVAL;
```

#### 8.2 Reflection Job

- [ ] Create scheduled job (daily or weekly)
- [ ] Query recent completed trades
- [ ] Group by strategy
- [ ] Call `memory.reflect` with:
  - Strategy performance stats
  - Notable patterns (e.g., "RSI works better on tech stocks")
  - Risk management insights
- [ ] Store beliefs in memory

**Implementation:**

```go
// services/jax-reflection/internal/app/reflector.go
type Reflector struct {
    tradeRepo  TradeRepository
    memoryClient MemoryClient
}

func (r *Reflector) ReflectOnPeriod(ctx context.Context, since time.Time) error {
    trades := r.tradeRepo.GetCompleted(ctx, since)
    
    // Group by strategy
    byStrategy := groupByStrategy(trades)
    
    for strategyID, trades := range byStrategy {
        stats := calculateStats(trades)
        
        // Generate reflection prompt
        prompt := fmt.Sprintf(`Analyze these %d trades from strategy %s:

Win rate: %.1f%%
Avg win: $%.2f (%.1f%%)
Avg loss: $%.2f (%.1f%%)
Profit factor: %.2f
Best trade: %s
Worst trade: %s

What patterns do you notice? What should we learn?`,
            len(trades), strategyID,
            stats.WinRate*100,
            stats.AvgWin, stats.AvgWinPercent,
            stats.AvgLoss, stats.AvgLossPercent,
            stats.ProfitFactor,
            stats.BestTrade,
            stats.WorstTrade)
        
        // Call memory.reflect
        reflection, err := r.memoryClient.Reflect(ctx, contracts.MemoryReflectRequest{
            Bank: "trade_outcomes",
            Prompt: prompt,
        })
        
        if err != nil {
            log.Error("Reflection failed", "strategy", strategyID, "error", err)
            continue
        }
        
        // Store belief
        r.memoryClient.Retain(ctx, contracts.MemoryRetainRequest{
            Bank: "beliefs",
            Item: contracts.MemoryItem{
                Summary: fmt.Sprintf("Strategy %s performance insight", strategyID),
                Content: reflection.Insight,
                Metadata: map[string]interface{}{
                    "strategy_id": strategyID,
                    "trade_count": len(trades),
                    "win_rate": stats.WinRate,
                    "generated_at": time.Now(),
                },
            },
        })
    }
    
    return nil
}
```

#### 8.3 Use Beliefs in Future Decisions

- [ ] When orchestrating, recall beliefs
- [ ] Include beliefs in Agent0 context
- [ ] AI considers past learnings in recommendations

**Update orchestrator:**

```go
func (o *Orchestrator) recallRelevantContext(ctx context.Context, symbol string, strategyID string) string {
    // Recall past outcomes for this symbol
    symbolMemories, _ := o.memory.Recall(ctx, "trade_outcomes", contracts.MemoryQuery{
        Query: fmt.Sprintf("trades on %s", symbol),
        Limit: 5,
    })
    
    // Recall beliefs about this strategy
    strategyBeliefs, _ := o.memory.Recall(ctx, "beliefs", contracts.MemoryQuery{
        Query: fmt.Sprintf("strategy %s performance", strategyID),
        Limit: 3,
    })
    
    context := "Relevant past experiences:\n"
    for _, mem := range symbolMemories {
        context += fmt.Sprintf("- %s\n", mem.Summary)
    }
    
    context += "\nLearned insights:\n"
    for _, belief := range strategyBeliefs {
        context += fmt.Sprintf("- %s\n", belief.Content)
    }
    
    return context
}
```

**Acceptance Criteria:**

- ✅ Trade outcomes calculated and stored
- ✅ Reflection job runs on schedule
- ✅ Beliefs generated from trade patterns
- ✅ Beliefs stored in memory
- ✅ Future orchestrations include beliefs
- ✅ AI recommendations improve over time (measurable)

---

### **Phase 9: Advanced Signal Detection (Week 10-12)**

**Goal:** Add more sophisticated signal types beyond basic technical indicators

#### 9.1 Multi-Timeframe Analysis

- [ ] Analyze multiple timeframes (1min, 5min, 1hr, daily)
- [ ] Require alignment across timeframes for higher confidence
- [ ] Detect divergences between timeframes

#### 9.2 Market Regime Detection

- [ ] Classify market as: trending, ranging, volatile, quiet
- [ ] Adjust strategy selection based on regime
- [ ] Different strategies for different conditions

#### 9.3 Correlation & Sector Analysis

- [ ] Track sector performance
- [ ] Detect relative strength
- [ ] Avoid correlated positions (risk management)

#### 9.4 Pattern Recognition

- [ ] Chart patterns (triangles, head & shoulders, etc.)
- [ ] Candlestick patterns (engulfing, doji, etc.)
- [ ] Support/resistance levels

**Acceptance Criteria:**

- ✅ At least 3 new signal types implemented
- ✅ Multi-timeframe signals have higher avg confidence
- ✅ Market regime affects strategy selection
- ✅ Advanced signals integrated into orchestration

---

### **Phase 10: Production Hardening (Week 12-13)**

**Goal:** Make system reliable, monitored, and fault-tolerant

#### 10.1 Monitoring & Alerting

- [ ] Add Prometheus metrics to all services
- [ ] Create Grafana dashboards:
  - Signal generation rate
  - Orchestration success rate
  - Trade execution latency
  - Memory system performance
  - API error rates
- [ ] Set up alerts for:
  - Service down
  - High error rates
  - IB connection lost
  - Data ingestion stopped

#### 10.2 Error Handling & Recovery

- [ ] Implement retry logic with exponential backoff
- [ ] Add circuit breakers to external calls
- [ ] Handle IB disconnections gracefully
- [ ] Recover from database failures

#### 10.3 Rate Limiting & Cost Control

- [ ] Rate limit external API calls (news, market data)
- [ ] Track Agent0 LLM token usage
- [ ] Implement cost budgets
- [ ] Add kill switches for runaway costs

#### 10.4 Data Backups

- [ ] Automated database backups
- [ ] Backup memory system (Hindsight data)
- [ ] Export trade history

#### 10.5 Testing

- [ ] Integration tests for full pipeline
- [ ] Load testing (can it handle 100+ symbols?)
- [ ] Chaos engineering (kill services, test recovery)

**Acceptance Criteria:**

- ✅ All services have metrics
- ✅ Dashboards show real-time system health
- ✅ Alerts configured and tested
- ✅ Services recover from failures
- ✅ Data backed up automatically
- ✅ Integration tests pass

---

### **Phase 11: User Experience Enhancements (Week 13-14)**

**Goal:** Polish the UI and add quality-of-life features

#### 11.1 Real-Time Updates

- [ ] WebSocket for live signal updates
- [ ] Toast notifications for new signals
- [ ] Browser notifications (opt-in)
- [ ] Sound alerts for high-confidence signals

#### 11.2 Signal Filtering & Search

- [ ] Filter by: strategy, symbol, confidence, date
- [ ] Search signal history
- [ ] Save custom filters

#### 11.3 Performance Analytics

- [ ] Strategy performance dashboard
- [ ] Win rate by symbol, strategy, timeframe
- [ ] Equity curve visualization
- [ ] Drawdown analysis

#### 11.4 Backtesting UI

- [ ] Test strategies on historical data
- [ ] Visualize results
- [ ] Compare strategy performance

#### 11.5 Configuration UI

- [ ] Manage watchlist from UI
- [ ] Configure strategy parameters
- [ ] Set risk limits
- [ ] Adjust orchestration triggers

**Acceptance Criteria:**

- ✅ Real-time signal updates (no manual refresh)
- ✅ Filtering and search work smoothly
- ✅ Performance analytics accurate
- ✅ User can configure system from UI

---

### **Phase 12: Advanced Features (Week 14-16+)**

**Goal:** Next-level capabilities

#### 12.1 Multi-Account Support

- [ ] Support multiple IB accounts
- [ ] Account-specific risk limits
- [ ] Portfolio-level position management

#### 12.2 Options Trading

- [ ] Options chain data
- [ ] Options strategies (covered calls, spreads)
- [ ] Greeks calculation
- [ ] IV rank analysis

#### 12.3 Paper Trading Mode

- [ ] Simulated trading environment
- [ ] Test new strategies risk-free
- [ ] Compare paper vs live performance

#### 12.4 Social Features

- [ ] Share signals with team
- [ ] Collaborative watchlists
- [ ] Trade journals with notes

#### 12.5 Mobile App

- [ ] React Native mobile app
- [ ] Push notifications
- [ ] Approve trades on-the-go

---

## 📈 Success Metrics

### Week 4 (End of Phase 3)

- [ ] 10+ signals generated per day
- [ ] Orchestration success rate > 95%
- [ ] Average orchestration latency < 5s

### Week 8 (End of Phase 6)

- [ ] User can see and approve signals in UI
- [ ] At least 1 live trade executed successfully
- [ ] Zero critical errors in production

### Week 12 (End of Phase 9)

- [ ] 5+ signal types operational
- [ ] Reflection system running
- [ ] Measurable improvement in win rate over time

### Week 16 (End of Phase 12)

- [ ] System running autonomously 24/7
- [ ] Average daily signals: 20+
- [ ] Trading multiple symbols profitably
- [ ] Full audit trail for all decisions

---

## 🚨 Risk Management

### Phase Gates

Before advancing to next phase:

- ✅ All acceptance criteria met
- ✅ No critical bugs
- ✅ Code reviewed
- ✅ Tests passing
- ✅ Deployed to staging environment

### Rollback Plan

- Keep previous phase deployments available
- Database migrations must have rollback scripts
- Feature flags for new capabilities

### Testing Strategy

- Unit tests for all business logic
- Integration tests for service interactions
- End-to-end tests for critical user flows
- Manual testing for UI/UX
- Paper trading validation before live

---

## 📝 Next Steps

1. **Review this roadmap** and adjust phases/timeline
2. **Set up project tracking** (GitHub Projects, Jira, etc.)
3. **Start Phase 1** - Deploy market data pipeline
4. **Weekly progress reviews** - Stay on track
5. **Iterate based on learnings** - Adjust plan as needed

---

## 🎯 Vision: 6 Months from Now

The user wakes up to:

- 3-5 high-quality trade recommendations waiting for approval
- AI has analyzed overnight news and market conditions
- Each signal has clear reasoning, risk metrics, and historical context
- One tap approves the trade
- Position managed automatically (stops, targets)
- Weekly reflection report shows improving performance
- System has learned from 100+ past trades
- Equity curve trending upward

**Jax becomes your tireless trading partner, watching the markets while you sleep.**
