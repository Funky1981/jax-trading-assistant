# Implementation Roadmap: Bridging the Gaps

**Project:** Jax Trading Assistant  
**Goal:** Make AI trading suggestions visible in the UI  
**Timeline:** 2-3 weeks for Phase 1

---

## 🎯 Mission Statement

Transform the Jax Trading Assistant from a collection of isolated services into a fully integrated AI-powered trading system where:

1. Users can trigger AI analysis from the frontend
2. AI generates trading suggestions based on memory + signals
3. Real-time strategy signals appear in the UI
4. The system learns from past decisions via reflection

---

## 📋 Phase 1: AI Orchestration (Week 1) - CRITICAL PATH

**Goal:** User clicks "Analyze AAPL" and sees AI suggestion with confidence score

### Task 1.1: Create Agent0 HTTP Service (2 days)

**Why:** jax-orchestrator needs to call Agent0 for planning

**Implementation:**

#### Step 1: Create service directory structure

```bash
mkdir -p services/agent0-api
cd services/agent0-api
```

#### Step 2: Create files

`services/agent0-api/main.py`:

```python
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn
from typing import List, Dict, Any, Optional

app = FastAPI(title="Agent0 Planning Service")

class Memory(BaseModel):
    summary: str
    type: str
    symbol: Optional[str] = None
    tags: List[str] = []
    data: Dict[str, Any] = {}

class PlanRequest(BaseModel):
    task: str
    context: str
    symbol: Optional[str] = None
    constraints: Dict[str, Any] = {}
    memories: List[Memory] = []

class PlanResponse(BaseModel):
    summary: str
    steps: List[str]
    action: str  # "buy", "sell", "hold", "skipped"
    confidence: float  # 0.0 to 1.0
    reasoning_notes: str

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "agent0-api", "version": "1.0.0"}

@app.post("/v1/plan")
async def plan(request: PlanRequest) -> PlanResponse:
    """
    Simple rule-based planner (v1)
    Can be enhanced with ML later
    """
    # Extract key context
    has_positive_signals = any(
        mem.type == "signal" and "buy" in mem.summary.lower()
        for mem in request.memories
    )
    has_negative_signals = any(
        mem.type == "signal" and "sell" in mem.summary.lower()
        for mem in request.memories
    )
    
    # Simple decision logic
    if has_positive_signals and not has_negative_signals:
        action = "buy"
        confidence = 0.70
        reasoning = f"Positive signals for {request.symbol}. Memories suggest bullish setup."
    elif has_negative_signals and not has_positive_signals:
        action = "sell"
        confidence = 0.65
        reasoning = f"Negative signals for {request.symbol}. Memories suggest bearish setup."
    elif request.constraints.get("force_action"):
        action = request.constraints["force_action"]
        confidence = 0.50
        reasoning = "Forced action by constraint."
    else:
        action = "hold"
        confidence = 0.60
        reasoning = "No clear signals. Holding current position."
    
    return PlanResponse(
        summary=f"Analyzed {request.symbol}: {action.upper()} recommendation",
        steps=[
            "Reviewed market context",
            f"Recalled {len(request.memories)} relevant memories",
            "Evaluated signal strength",
            f"Decided to {action}"
        ],
        action=action,
        confidence=confidence,
        reasoning_notes=reasoning
    )

@app.post("/v1/execute")
async def execute(request: Dict[str, Any]):
    """
    Placeholder for execution logic
    Returns success for now
    """
    return {"success": True, "message": "Execution acknowledged"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8094)
```

`services/agent0-api/requirements.txt`:

```text
fastapi==0.109.0
uvicorn[standard]==0.27.0
pydantic==2.5.0
```

`services/agent0-api/Dockerfile`:

```dockerfile
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

EXPOSE 8094

CMD ["python", "main.py"]
```

#### Step 3: Test locally

```bash
cd services/agent0-api
pip install -r requirements.txt
python main.py

# In another terminal:
curl http://localhost:8094/health
curl -X POST http://localhost:8094/v1/plan \
  -H "Content-Type: application/json" \
  -d '{
    "task": "analyze AAPL",
    "context": "Market is bullish",
    "symbol": "AAPL",
    "memories": [{"summary": "Previous buy signal", "type": "signal"}]
  }'
```

#### Step 4: Add to docker-compose.yml

```yaml
  agent0-api:
    build:
      context: ./services/agent0-api
      dockerfile: Dockerfile
    environment:
      HOST: 0.0.0.0
      PORT: 8094
      LOG_LEVEL: ${AGENT0_LOG_LEVEL:-INFO}
    ports:
      - "8094:8094"
    healthcheck:
      test: ["CMD-SHELL", "python -c \"import requests; requests.get('http://localhost:8094/health')\""]
      interval: 30s
      timeout: 10s
      start_period: 10s
      retries: 3
```

**Success Criteria:**

- ✅ Service starts: `docker compose up agent0-api`
- ✅ Health check passes: `curl http://localhost:8094/health`
- ✅ Plan endpoint works: Returns PlanResponse
- ✅ Logs show "Uvicorn running on <http://0.0.0.0:8094>"

---

### Task 1.2: Create Orchestrator HTTP Server (2 days)

**Why:** Frontend needs REST API to trigger orchestration

**Implementation:**

#### Step 1: Create HTTP server

`services/jax-orchestrator/cmd/server/main.go`:

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "jax-trading-assistant/libs/agent0"
    "jax-trading-assistant/libs/contracts"
    "jax-trading-assistant/libs/observability"
    "jax-trading-assistant/libs/strategies"
    "jax-trading-assistant/libs/utcp"
    "jax-trading-assistant/services/jax-orchestrator/internal/app"
    "jax-trading-assistant/services/jax-orchestrator/internal/config"
)

type Server struct {
    orchestrator *app.Orchestrator
    port         int
}

func main() {
    var port int
    var providersPath string
    flag.IntVar(&port, "port", 8093, "HTTP port")
    flag.StringVar(&providersPath, "providers", "config/providers.json", "Providers path")
    flag.Parse()

    // Initialize UTCP client
    client, err := utcp.NewUTCPClientFromFile(providersPath)
    if err != nil {
        log.Fatal(err)
    }

    // Initialize memory adapter
    memorySvc := utcp.NewMemoryService(client)
    memory := &memoryAdapter{svc: memorySvc}

    // Initialize Agent0 client
    agent0URL := os.Getenv("AGENT0_URL")
    if agent0URL == "" {
        agent0URL = "http://localhost:8094"
    }
    agentClient, err := agent0.New(agent0URL)
    if err != nil {
        log.Fatalf("failed to create agent0 client: %v", err)
    }
    agent := &agent0Adapter{client: agentClient}

    // Initialize strategy registry (optional)
    var strategyRegistry *strategies.Registry
    // TODO: Load from config if needed

    // Create orchestrator
    tools := &stubToolRunner{}
    orch := app.NewOrchestrator(memory, agent, tools, strategyRegistry)

    // Start server
    srv := &Server{
        orchestrator: orch,
        port:         port,
    }
    srv.Start()
}

func (s *Server) Start() {
    Symbol          string         `json:"symbol"`
    Strategy        string         `json:"strategy,omitempty"`
    Bank            string         `json:"bank,omitempty"`
    UserContext     string         `json:"context,omitempty"`
    Constraints     map[string]any `json:"constraints,omitempty"`
    ResearchQueries []string       `json:"research_queries,omitempty"`
}

func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req orchestrateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if req.Symbol == "" {
        http.Error(w, "symbol is required", http.StatusBadRequest)
        return
    }
    if req.Bank == "" {
        req.Bank = "decisions" // default
    }

    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()

    runID := observability.NewRunID()
    ctx = observability.WithRunInfo(ctx, observability.RunInfo{
        RunID:  runID,
        TaskID: "orchestrate-" + req.Symbol,
        Symbol: req.Symbol,
    })

    result, err := s.orchestrator.Run(ctx, app.OrchestrationRequest{
        Bank:            req.Bank,
        Symbol:          req.Symbol,
        Strategy:        req.Strategy,
        UserContext:     req.UserContext,
        Constraints:     req.Constraints,
        ResearchQueries: req.ResearchQueries,
    })
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]any{
        "runId": runID,
        "plan": map[string]any{
            "summary":         result.Plan.Summary,
            "steps":           result.Plan.Steps,
            "action":          result.Plan.Action,
            "confidence":      result.Plan.Confidence,
            "reasoning_notes": result.Plan.ReasoningNotes,
        },
        "tools":     result.Tools,
        "symbol":    req.Symbol,
        "timestamp": time.Now().Format(time.RFC3339),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Stub implementations for other handlers
func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement run status tracking
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "status": "completed",
        "message": "Run status tracking not yet implemented",
    })
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
    // TODO: Implement run history
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode([]map[string]any{})
}

// Adapters
type memoryAdapter struct {
    svc *utcp.MemoryService
}

func (m *memoryAdapter) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
    out, err := m.svc.Recall(ctx, contracts.MemoryRecallRequest{Bank: bank, Query: query})
    if err != nil {
        return nil, err
    }
    return out.Items, nil
}

func (m *memoryAdapter) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
    out, err := m.svc.Retain(ctx, contracts.MemoryRetainRequest{Bank: bank, Item: item})
    if err != nil {
        return "", err
    }
    return out.ID, nil
}

type agent0Adapter struct {
    client *agent0.Client
}

func (a *agent0Adapter) Plan(ctx context.Context, req agent0.PlanRequest) (agent0.PlanResponse, error) {
    return a.client.Plan(ctx, req)
}

func (a *agent0Adapter) Execute(ctx context.Context, req agent0.ExecuteRequest) (agent0.ExecuteResponse, error) {
    return a.client.Execute(ctx, req)
}

type stubToolRunner struct{}

func (stubToolRunner) Execute(_ context.Context, _ app.PlanResult) ([]app.ToolRun, error) {
    return []app.ToolRun{}, nil
}
```

#### Step 2: Build and test

```bash
cd services/jax-orchestrator
go build -o bin/server ./cmd/server

# Test
./bin/server -port 8093

# In another terminal:
curl http://localhost:8093/health
curl -X POST http://localhost:8093/api/v1/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL"}'
```

#### Step 3: Add to docker-compose.yml

```yaml
  jax-orchestrator:
    build:
      context: .
      dockerfile: services/jax-orchestrator/Dockerfile.server
    environment:
      PORT: 8093
      AGENT0_URL: http://agent0-api:8094
      PROVIDERS_PATH: /workspace/config/providers.json
    ports:
      - "8093:8093"
    depends_on:
      - jax-memory
      - agent0-api
    volumes:
      - ./config:/workspace/config:ro
```

#### Step 4: Create Dockerfile

`services/jax-orchestrator/Dockerfile.server`:

```dockerfile
FROM golang:1.22 AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/orchestrator-server ./services/jax-orchestrator/cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/orchestrator-server /bin/orchestrator-server
EXPOSE 8093
CMD ["/bin/orchestrator-server"]
```

**Success Criteria:**

- ✅ Server starts on port 8093
- ✅ `/health` returns 200
- ✅ `POST /api/v1/orchestrate` calls Agent0
- ✅ Memory recall/retain works
- ✅ Returns plan with confidence

---

### Task 1.3: Update Frontend Configuration (1 hour)

**Why:** Frontend needs to know where orchestrator API is

**Implementation:**

#### Step 1: Update environment config

`frontend/.env`:

```env
VITE_API_URL=http://localhost:8081
VITE_MEMORY_API_URL=http://localhost:8090
VITE_ORCHESTRATOR_API_URL=http://localhost:8093
```

#### Step 2: Update http client (if needed)

`frontend/src/data/orchestration-service.ts`:

```typescript
import { apiClient } from './http-client';
// OR create dedicated client:
import { createHttpClient } from './http-client';

const orchestratorClient = createHttpClient({
  baseUrl: import.meta.env.VITE_ORCHESTRATOR_API_URL || 'http://localhost:8093',
  timeoutMs: 30_000,
});

export const orchestrationService = {
  async run(request: OrchestrationRequest): Promise<OrchestrationResult> {
    return orchestratorClient.post<OrchestrationResult>('/api/v1/orchestrate', request);
  },
  // ... rest unchanged
};
```

**Success Criteria:**

- ✅ Frontend calls correct orchestrator URL
- ✅ No CORS errors
- ✅ Requests reach orchestrator

---

### Task 1.4: End-to-End Integration Test (1 day)

**Implementation:**

#### Step 1: Start all services

```bash
docker compose up -d hindsight jax-memory ib-bridge agent0-api jax-orchestrator

# Check all healthy
curl http://localhost:8888/health  # hindsight
curl http://localhost:8090/health  # jax-memory (might need /tools)
curl http://localhost:8092/health  # ib-bridge
curl http://localhost:8094/health  # agent0-api
curl http://localhost:8093/health  # jax-orchestrator
```

#### Step 2: Test orchestration flow

```bash
# Trigger orchestration
curl -X POST http://localhost:8093/api/v1/orchestrate \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "AAPL",
    "strategy": "macd",
    "context": "Analyzing for gap trade",
    "bank": "decisions"
  }' | jq .

# Expected response:
{
  "runId": "orch-abc123",
  "plan": {
    "summary": "Analyzed AAPL: BUY recommendation",
    "steps": [...],
    "action": "buy",
    "confidence": 0.70,
    "reasoning_notes": "..."
  },
  "tools": [],
  "symbol": "AAPL",
  "timestamp": "2026-02-04T..."
}
```

#### Step 3: Verify memory retention

```bash
# Check that decision was retained
curl -X POST http://localhost:8090/tools \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "memory.recall",
    "input": {
      "bank": "decisions",
      "query": {
        "symbol": "AAPL",
        "limit": 5
      }
    }
  }' | jq .
```

#### Step 4: Test from frontend

```typescript
// In browser console or component
const result = await orchestrationService.run({
  symbol: 'AAPL',
  strategy: 'macd',
  context: 'Test from UI'
});
console.log(result);
```

**Success Criteria:**

- ✅ Orchestration completes in < 5s
- ✅ Agent0 returns plan
- ✅ Memory shows retained decision
- ✅ Frontend receives response
- ✅ No errors in any service logs

---

### Task 1.5: Add UI Component for Orchestration Results (1 day)

**Why:** User needs to see AI suggestions

**Implementation:**

#### Step 1: Create AI Assistant Panel

`frontend/src/components/dashboard/AIAssistantPanel.tsx`:

```typescript
import { Brain, TrendingUp, TrendingDown } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { useOrchestrationRun } from '@/hooks/useOrchestration';
import { cn } from '@/lib/utils';

export function AIAssistantPanel() {
  const [symbol, setSymbol] = useState('');
  const { mutate: runOrchestration, data, isLoading } = useOrchestrationRun();

  const handleAnalyze = () => {
    if (!symbol.trim()) return;
    
    runOrchestration({
      symbol: symbol.toUpperCase(),
      strategy: 'macd',
      context: `User requested analysis of ${symbol.toUpperCase()}`,
      bank: 'decisions'
    });
  };

  return (
    <Card className="p-6">
      <div className="flex items-center gap-2 mb-4">
        <Brain className="h-5 w-5 text-primary" />
        <h3 className="text-lg font-semibold">AI Trading Assistant</h3>
      </div>

      <div className="flex gap-2 mb-4">
        <Input
          placeholder="Enter symbol (e.g., AAPL)"
          value={symbol}
          onChange={(e) => setSymbol(e.target.value.toUpperCase())}
          onKeyPress={(e) => e.key === 'Enter' && handleAnalyze()}
        />
        <Button onClick={handleAnalyze} disabled={isLoading}>
          {isLoading ? 'Analyzing...' : 'Analyze'}
        </Button>
      </div>

      {data && (
        <div className="space-y-3">
          {/* Action Badge */}
          <div className="flex items-center gap-2">
            {data.plan.action === 'buy' && (
              <TrendingUp className="h-5 w-5 text-success" />
            )}
            {data.plan.action === 'sell' && (
              <TrendingDown className="h-5 w-5 text-destructive" />
            )}
            <Badge
              variant={
                data.plan.action === 'buy'
                  ? 'success'
                  : data.plan.action === 'sell'
                  ? 'destructive'
                  : 'secondary'
              }
              className="text-sm"
            >
              {data.plan.action.toUpperCase()}
            </Badge>
            <span className="text-sm text-muted-foreground">
              Confidence: {(data.plan.confidence * 100).toFixed(0)}%
            </span>
          </div>

          {/* Summary */}
          <p className="text-sm">{data.plan.summary}</p>

          {/* Steps */}
          <div className="text-xs text-muted-foreground">
            <p className="font-semibold mb-1">Analysis Steps:</p>
            <ol className="list-decimal list-inside space-y-1">
              {data.plan.steps.map((step, i) => (
                <li key={i}>{step}</li>
              ))}
            </ol>
          </div>

          {/* Reasoning */}
          {data.plan.reasoning_notes && (
            <div className="text-xs text-muted-foreground">
              <p className="font-semibold mb-1">AI Reasoning:</p>
              <p className="italic">{data.plan.reasoning_notes}</p>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}
```

#### Step 2: Add to Dashboard

`frontend/src/pages/DashboardPage.tsx`:

```typescript
import { AIAssistantPanel } from '@/components/dashboard/AIAssistantPanel';

// In the grid:
<DashboardPanel span={{ base: 12, lg: 6 }}>
  <AIAssistantPanel />
</DashboardPanel>
```

**Success Criteria:**

- ✅ User enters symbol, clicks Analyze
- ✅ Loading state shows during orchestration
- ✅ AI suggestion appears with confidence
- ✅ Action badge shows buy/sell/hold
- ✅ Reasoning visible to user

---

## ✅ Phase 1 Complete Checklist

- [ ] Agent0 HTTP service running (port 8094)
- [ ] Agent0 `/v1/plan` endpoint works
- [ ] jax-orchestrator HTTP server running (port 8093)
- [ ] Orchestrator `/api/v1/orchestrate` endpoint works
- [ ] Memory recall integration works
- [ ] Memory retention after orchestration works
- [ ] Frontend can trigger orchestration
- [ ] AI suggestion appears in UI
- [ ] Confidence score displayed
- [ ] User sees AI reasoning
- [ ] No errors in docker compose logs
- [ ] All health checks green

---

## 📋 Phase 2: Strategy Signals (Week 2)

[See GAP_ANALYSIS_REPORT.md for detailed Phase 2 plan]

**Summary:**

1. Add signal storage (migration 000004)
2. Create signal generation endpoints
3. Background signal generator
4. Wire to StrategyMonitorPanel

---

## 📋 Phase 3: Real Market Data (Week 3)

**Summary:**

1. Enable Dexter production mode
2. Create ingestion pipeline (IB → DB)
3. Wire Dexter to signal generation
4. End-to-end: IB → Dexter → Signals → UI

---

## 🔍 Validation Points

### After Phase 1

**Test Script:**

```bash
#!/bin/bash
echo "=== Phase 1 Validation ==="

echo "1. Starting services..."
docker compose up -d hindsight jax-memory ib-bridge agent0-api jax-orchestrator

echo "2. Waiting for services to be healthy..."
sleep 10

echo "3. Testing Agent0..."
curl http://localhost:8094/health || echo "FAIL: Agent0 not healthy"

echo "4. Testing Orchestrator..."
curl http://localhost:8093/health || echo "FAIL: Orchestrator not healthy"

echo "5. Triggering orchestration..."
RESULT=$(curl -s -X POST http://localhost:8093/api/v1/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"symbol":"AAPL","bank":"decisions"}')
echo $RESULT | jq .

echo "6. Checking memory..."
curl -s -X POST http://localhost:8090/tools \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "memory.recall",
    "input": {"bank": "decisions", "query": {"symbol": "AAPL", "limit": 1}}
  }' | jq .

echo "=== Validation Complete ==="
```

Run: `chmod +x validate-phase1.sh && ./validate-phase1.sh`

**Expected Output:**

```text
1. Starting services... ✅
2. Waiting... ✅
3. Testing Agent0... {"status":"healthy",...} ✅
4. Testing Orchestrator... {"status":"healthy",...} ✅
5. Triggering orchestration... {"runId":"orch-...","plan":{...}} ✅
6. Checking memory... {"items":[...]} ✅
```

---

## 🚨 Common Issues & Solutions

### Issue 1: Agent0 service won't start

**Symptoms:**

```text
agent0-api_1 exited with code 1
ModuleNotFoundError: No module named 'fastapi'
```

**Solution:**

```bash
cd services/agent0-api
pip install -r requirements.txt
# OR rebuild docker image:
docker compose build agent0-api
```

### Issue 2: Orchestrator can't connect to Agent0

**Symptoms:**

```text
failed to call agent0: connection refused
```

**Solution:**

- Check AGENT0_URL environment variable
- Ensure agent0-api is running: `docker compose ps agent0-api`
- Check network: `docker compose exec jax-orchestrator ping agent0-api`

### Issue 3: Frontend CORS error

**Symptoms:**

```text
Access to fetch at 'http://localhost:8093/api/v1/orchestrate' has been blocked by CORS
```

**Solution:**

Add CORS middleware to orchestrator:

```go
// In main.go
mux := http.NewServeMux()
handler := enableCORS(mux)
http.ListenAndServe(addr, handler)

func enableCORS(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        if r.Method == "OPTIONS" {
            return
        }
        h.ServeHTTP(w, r)
    })
}
```

---

## 📊 Success Metrics

### Phase 1 Metrics

- **Latency:** Orchestration completes in < 5s
- **Reliability:** 99% success rate (no crashes)
- **Memory:** < 200MB per service
- **User Experience:** AI suggestion visible in < 5s from click

### Monitoring

```bash
# Check orchestration performance
docker compose logs jax-orchestrator | grep "orchestration completed"

# Check memory usage
docker stats --no-stream

# Check Agent0 response times
docker compose logs agent0-api | grep "POST /v1/plan"
```

---

## 🎓 Next Steps After Phase 1

1. **Enhance Agent0:** Add ML model for better planning
2. **Add Signal Storage:** Phase 2 implementation
3. **Enable Reflection:** Weekly belief synthesis
4. **Performance Tuning:** Optimize response times
5. **Add Monitoring:** Prometheus + Grafana
6. **Write Tests:** Integration tests for full flow

---

## Ready to start? Begin with Task 1.1: Create Agent0 HTTP Service
