# Jax Trading Assistant - Quick Reference: What Works vs What's Missing

> Historical snapshot: this file documents an older runtime topology and archived gap analysis. References to `jax-memory`, `hindsight`, `jax-orchestrator`, and legacy API expectations are not current operating guidance.

## System Status at a Glance

```
┌─────────────────────────────────────────────────────────────────────┐
│                    JAX TRADING ASSISTANT STATUS                      │
│                      February 4, 2026                                │
└─────────────────────────────────────────────────────────────────────┘

✅ = Working    ⚠️ = Partial    ❌ = Missing    🔌 = Disconnected
```

---

## Architecture Status

```
                 ┌──────────────┐
                 │   Frontend   │  ✅ UI Built, 🔌 APIs Missing
                 │  (React/TS)  │  
                 └───────┬──────┘
                         │
                         ↓
            ┌────────────────────────┐
            │      jax-api           │  ✅ Running (Port 8081)
            │   (Main Backend)       │  ⚠️ Missing orchestration/signals APIs
            └────────┬───────────────┘
                     │
         ┌───────────┼───────────┬────────────┐
         │           │           │            │
         ↓           ↓           ↓            ↓
    ┌────────┐  ┌────────┐  ┌────────┐  ┌─────────┐
    │ IB     │  │ Memory │  │ Orch   │  │ Agent0  │
    │ Bridge │  │ Facade │  │ -est   │  │ API     │
    └────────┘  └────────┘  └────────┘  └─────────┘
    ✅ 8092     ✅ 8090     ❌ CLI Only  ❌ Missing
    
    Running     Running     No HTTP     No Service
    in Docker   in Docker   Server      at all
```

---

## Component Status

### ✅ WORKING (Production Ready)

1. **IB Bridge** (Port 8092)
   - FastAPI Python service
   - REST API + WebSocket
   - Market data, orders, positions
   - Docker + health checks
   - Circuit breaker

2. **jax-memory** (Port 8090)
   - Memory facade over Hindsight
   - memory.retain, memory.recall working
   - UTCP tools interface

3. **hindsight** (Port 8888)
   - Vendored vector memory service
   - Running in Docker

4. **jax-api** (Port 8081)
   - Main backend API
   - Auth (JWT), rate limiting, CORS
   - Risk calc, trades, metrics
   - ❌ Missing: orchestration + signals APIs

5. **Frontend** (Port 5173)
   - React + TypeScript + Vite
   - All UI components built
   - 🔌 Disconnected: APIs don't exist

6. **Database** (PostgreSQL)
   - Schema with migrations
   - events, trades, audit_events, market_data

7. **Libraries**
   - libs/auth, libs/strategies, libs/utcp
   - libs/agent0 (client), libs/dexter (client)

---

### ⚠️ PARTIAL (Works but Incomplete)

1. **Dexter** (tools-server.ts)
   - ✅ HTTP server runs
   - ✅ Mock mode works
   - ❌ Real signal generation missing
   - ❌ Event detection not wired

2. **jax-orchestrator**
   - ✅ Core logic exists
   - ✅ Memory integration works
   - ❌ CLI only, no HTTP server
   - ❌ No REST API

3. **Strategy System**
   - ✅ Strategy logic (MACD, RSI, MA)
   - ✅ `/strategies` endpoint (list only)
   - ❌ Signal generation not running
   - ❌ Signal storage missing
   - ❌ Performance tracking missing

---

### ❌ MISSING (Not Implemented)

1. **Agent0 HTTP Service**
   - No service at all
   - Only training code exists
   - Need: FastAPI with /v1/plan, /v1/execute

2. **Orchestration API**
   - Frontend expects:
     - POST /api/v1/orchestrate
     - GET /api/v1/orchestrate/runs/{id}
     - GET /api/v1/orchestrate/runs
   - Backend: None of these exist

3. **Strategy Signals API**
   - Frontend expects:
     - GET /api/v1/strategies/{id}/signals
     - GET /api/v1/strategies/{id}/performance
     - POST /api/v1/strategies/{id}/analyze
   - Backend: None of these exist

4. **Signal Generation Pipeline**
   - No background job
   - No signal storage
   - Strategies not running continuously

5. **Reflection System**
   - memory.reflect not used
   - No belief synthesis
   - No learning loop

6. **Market Data Ingestion**
   - IB Bridge streams data ✅
   - No service consuming it ❌
   - No storage pipeline ❌

---

## Critical Disconnects

### 1. Frontend → Orchestration ❌

```
Frontend: useOrchestrationRun()
  ↓ calls
POST /api/v1/orchestrate
  ↓ expects
jax-orchestrator HTTP API
  ↓ reality
404 Not Found ❌
```

**Fix:** Create HTTP server wrapping jax-orchestrator

---

### 2. Frontend → Signals ❌

```
Frontend: StrategyMonitorPanel
  ↓ calls
GET /api/v1/strategies/macd/signals
  ↓ expects
Signal database + API
  ↓ reality
404 Not Found ❌
```

**Fix:** Add signal endpoints to jax-api

---

### 3. Orchestrator → Agent0 ❌

```
jax-orchestrator: agent.Plan(ctx, req)
  ↓ calls
libs/agent0.Client
  ↓ sends HTTP to
Agent0 service (expected: port ????)
  ↓ reality
Connection refused ❌
```

**Fix:** Create Agent0 HTTP service

---

### 4. IB Data → System ⚠️

```
IB Bridge: WebSocket /ws/quotes/AAPL
  ↓ streams to
??? (nothing listening)
  ↓ should go to
jax-ingest service
  ↓ reality
Service doesn't exist ❌
```

**Fix:** Create ingestion pipeline

---

## What User Can Do Today

### ✅ Working Now

- Start IB Bridge: `docker compose up ib-bridge`
- Get IB quote: `curl http://localhost:8092/quotes/AAPL`
- Check health: `curl http://localhost:8081/health`
- Login: `POST http://localhost:8081/auth/login`
- Calculate risk: `POST http://localhost:8081/risk/calc`
- List strategies: `GET http://localhost:8081/strategies`
- Store/recall memory: via jax-memory UTCP tools
- View frontend UI: `http://localhost:5173`

### ❌ Cannot Do Yet

- Trigger AI orchestration from UI
- See AI trading suggestions
- View real-time strategy signals
- Get signal history
- See AI reasoning/confidence
- Trigger on-demand analysis
- View orchestration history
- See reflection/beliefs

---

## Priority Fix List

### Week 1: Make AI Visible

**Day 1-2: Agent0 Service**
- Create `services/agent0-api/main.py`
- Endpoints: POST /v1/plan, POST /v1/execute
- Docker + health check
- Port: 8094

**Day 3-4: Orchestrator HTTP**
- Create `services/jax-orchestrator/cmd/server/main.go`
- Endpoints: POST /api/v1/orchestrate, GET /api/v1/orchestrate/runs/*
- Wire to Agent0
- Port: 8093

**Day 5: Test Integration**
- Frontend → orchestrate API → Agent0 → Memory
- Verify UI shows AI suggestions

### Week 2: Add Signals

**Day 1: Signal Storage**
- Migration: 000004_strategy_signals.up.sql
- Table: strategy_signals

**Day 2-3: Signal API**
- handlers_signals.go in jax-api
- GET /api/v1/strategies/{id}/signals
- GET /api/v1/strategies/{id}/performance

**Day 4-5: Signal Generator**
- Background job running strategies
- Store signals in DB
- Test frontend display

---

## Testing Checklist

### Phase 1: AI Integration

- [ ] Start Agent0: `docker compose up agent0-api`
- [ ] Agent0 health: `curl http://localhost:8094/health`
- [ ] Start orchestrator: `docker compose up jax-orchestrator`
- [ ] Trigger orchestration: `curl -X POST http://localhost:8093/api/v1/orchestrate -d '{"symbol":"AAPL"}'`
- [ ] Check memory: Query jax-memory for retained decision
- [ ] Frontend: Click "Analyze AAPL", see AI suggestion

### Phase 2: Signals

- [ ] Migration applied: `000004_strategy_signals`
- [ ] Signals in DB: `SELECT * FROM strategy_signals;`
- [ ] API works: `curl http://localhost:8081/api/v1/strategies/macd/signals`
- [ ] Frontend: StrategyMonitorPanel shows signals
- [ ] Auto-refresh: Signals update every 10s

---

## File Creation Summary

### Must Create (High Priority)

```
services/
  agent0-api/              ← NEW SERVICE
    main.py
    planner.py
    executor.py
    Dockerfile
  
  jax-orchestrator/
    cmd/server/
      main.go              ← NEW HTTP SERVER
    internal/handlers/
      orchestrate.go       ← NEW
  
  jax-api/internal/infra/http/
    handlers_signals.go    ← NEW
  
  jax-api/internal/app/
    signal_generator.go    ← NEW

db/postgres/migrations/
  000004_strategy_signals.up.sql    ← NEW
  000004_strategy_signals.down.sql  ← NEW
```

### Must Modify

```
docker-compose.yml
  ← Add agent0-api service
  ← Add jax-orchestrator HTTP service

services/jax-api/cmd/jax-api/main.go
  ← Register signal endpoints
  ← Add orchestration proxy (optional)

dexter/src/tools-server.ts
  ← Remove mock mode
  ← Add real signal generation
```

---

## Expected vs Actual Endpoints

| Endpoint | Expected By | Status | Notes |
|----------|-------------|--------|-------|
| POST /api/v1/orchestrate | Frontend | ❌ 404 | Need HTTP wrapper |
| GET /api/v1/orchestrate/runs/{id} | Frontend | ❌ 404 | Need HTTP wrapper |
| GET /api/v1/orchestrate/runs | Frontend | ❌ 404 | Need HTTP wrapper |
| GET /api/v1/strategies/{id}/signals | Frontend | ❌ 404 | Need endpoint + DB |
| GET /api/v1/strategies/{id}/performance | Frontend | ❌ 404 | Need endpoint |
| POST /api/v1/strategies/{id}/analyze | Frontend | ❌ 404 | Need endpoint |
| GET /strategies | Frontend | ✅ 200 | Works (lists configs) |
| POST /risk/calc | Frontend | ✅ 200 | Works |
| GET /api/v1/metrics | Frontend | ✅ 200 | Works |
| POST /tools (memory) | Orchestrator | ✅ 200 | Works via jax-memory |
| GET /quotes/{symbol} | System | ✅ 200 | Works via IB Bridge |

---

## Data Flow Status

### Intended Flow (Docs)

```
IB Gateway
  ↓ TCP socket
IB Bridge (8092) ✅
  ↓ WebSocket/HTTP
jax-ingest ❌
  ↓ Events
Dexter ⚠️ (mock)
  ↓ Signals
Database ❌
  ↓ API
Frontend 🔌
```

### Memory Flow

```
Decision made
  ↓
jax-orchestrator (CLI ⚠️)
  ↓ memory.retain
jax-memory (8090) ✅
  ↓ HTTP
hindsight (8888) ✅
  ↓ Storage
Vector DB ✅
```

### AI Flow

```
User request
  ↓ POST /api/v1/orchestrate
jax-orchestrator HTTP ❌
  ↓ memory.recall
jax-memory ✅
  ↓ Memories
Agent0 API ❌
  ↓ Plan
go-UTCP tools ✅
  ↓ Result
Frontend 🔌
```

---

## Quick Start (After Fixes)

### Expected Usage (Post-Phase 1)

```bash
# 1. Start all services
docker compose up -d

# Services running:
# - hindsight:8888 ✅
# - jax-memory:8090 ✅
# - ib-bridge:8092 ✅
# - jax-api:8081 ✅
# - agent0-api:8094 ← NEW
# - jax-orchestrator:8093 ← NEW
# - frontend:5173 ✅

# 2. Trigger AI analysis
curl -X POST http://localhost:8093/api/v1/orchestrate \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "strategy": "macd"}'

# Response:
{
  "plan": {
    "action": "buy",
    "confidence": 0.75,
    "reasoning": "MACD crossed above signal line..."
  },
  "tools": [...],
  "runId": "orch-123"
}

# 3. Check memory
curl -X POST http://localhost:8090/tools \
  -d '{"tool": "memory.recall", "input": {"bank": "decisions", "query": {"symbol": "AAPL"}}}'

# 4. View in UI
# Open http://localhost:5173
# Click "Analyze AAPL"
# See AI suggestion appear
```

---

**See full report: GAP_ANALYSIS_REPORT.md**
