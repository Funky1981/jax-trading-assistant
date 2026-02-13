# ADR-0012: Modular Monolith Implementation Plan

**Status**: Ready for Review  
**Created**: February 13, 2026  
**Timeline**: 8-10 weeks (with AI assistance)  
**Approach**: Incremental strangler pattern with parallel validation

---

## Overview

**Goal**: Migrate from 9 HTTP-connected microservices to 2 runtime entrypoints (trader + research) with artifact-based promotion gate.

**Key Principles**:
1. ✅ **Never break production** - Old system runs until new system proven
2. ✅ **Validate at every step** - Golden tests, replay tests, parallel runs
3. ✅ **Incremental delivery** - Each phase is independently valuable
4. ✅ **Clear rollback points** - Can abort at any checkpoint

**Risk Mitigation**:
- Phase 0 builds safety net (testing infrastructure) before touching production code
- Phases 1-2 collapse non-critical paths first (learn the pattern)
- Phase 3 tackles critical path (trade execution) with 2-week parallel validation
- Phase 4 completes the vision (research runtime + artifact builder)

---

## Phase 0: Foundation & Safety Net

**Duration**: Week 1  
**Risk**: Very Low  
**Goal**: Build testing infrastructure to validate all future changes

### Tasks

#### Task 0.1: Golden Test Infrastructure (Day 1)
**Files to Create**:
```
tests/
  golden/
    README.md
    capture.go
    signals/
      baseline-2026-02-13.json
    executions/
      baseline-2026-02-13.json
    orchestration/
      baseline-2026-02-13.json
```

**Implementation**:
```go
// tests/golden/capture.go
package golden

import (
    "encoding/json"
    "net/http"
    "os"
    "time"
)

type Snapshot struct {
    Captured  time.Time
    Service   string
    Endpoint  string
    Request   interface{}
    Response  interface{}
    Metadata  map[string]string
}

func CaptureSignals() error {
    // Call current jax-signal-generator
    resp, err := http.Get("http://localhost:8096/api/v1/signals")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    var signals interface{}
    json.NewDecoder(resp.Body).Decode(&signals)
    
    snapshot := Snapshot{
        Captured: time.Now(),
        Service:  "jax-signal-generator",
        Endpoint: "/api/v1/signals",
        Response: signals,
    }
    
    // Save
    f, _ := os.Create("tests/golden/signals/baseline-2026-02-13.json")
    defer f.Close()
    enc := json.NewEncoder(f)
    enc.SetIndent("", "  ")
    return enc.Encode(snapshot)
}
```

**Validation**:
```powershell
# Capture baseline
go run tests/golden/capture.go

# Verify files created
Test-Path tests/golden/signals/baseline-2026-02-13.json
Test-Path tests/golden/executions/baseline-2026-02-13.json
```

---

#### Task 0.2: Replay Harness (Day 2)
**Files to Create**:
```
tests/
  replay/
    harness.go
    harness_test.go
    fixtures/
      market_2026-01-15_aapl.json
      market_2026-01-20_msft.json
      market_2026-02-05_tsla.json
```

**Implementation**:
```go
// tests/replay/harness.go
package replay

import (
    "context"
    "encoding/json"
    "os"
    "time"
)

type Fixture struct {
    Name            string                 `json:"name"`
    Description     string                 `json:"description"`
    FixedTime       time.Time              `json:"fixed_time"`
    Seed            int64                  `json:"seed"`
    MarketData      map[string]Quote       `json:"market_data"`
    StrategyParams  map[string]interface{} `json:"strategy_params"`
    ExpectedSignals []Signal               `json:"expected_signals"`
}

func LoadFixture(path string) (*Fixture, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    
    var fixture Fixture
    json.NewDecoder(f).Decode(&fixture)
    return &fixture, nil
}

func ReplayStrategy(ctx context.Context, fixture *Fixture, executor StrategyExecutor) ([]Signal, error) {
    // Create deterministic context
    ctx = WithFixedClock(ctx, fixture.FixedTime)
    ctx = WithSeed(ctx, fixture.Seed)
    
    // Execute
    return executor.Generate(ctx, fixture.MarketData, fixture.StrategyParams)
}

func VerifyDeterminism(t *testing.T, fixture *Fixture, executor StrategyExecutor) {
    // Run 10 times
    results := make([][]Signal, 10)
    for i := 0; i < 10; i++ {
        signals, err := ReplayStrategy(context.Background(), fixture, executor)
        require.NoError(t, err)
        results[i] = signals
    }
    
    // All must match
    for i := 1; i < len(results); i++ {
        assert.Equal(t, results[0], results[i], 
            "replay %d produced different result", i)
    }
}
```

**Validation**:
```powershell
# Run replay tests
go test -v ./tests/replay/... -run TestDeterminism

# Expected output:
# PASS: TestDeterminism_RSI (10 replays identical)
# PASS: TestDeterminism_MACD (10 replays identical)
```

---

#### Task 0.3: Deterministic Clock (Day 3)
**Files to Create**:
```
libs/
  testing/
    clock.go
    clock_test.go
    seed.go
```

**Implementation**:
```go
// libs/testing/clock.go
package testing

import (
    "context"
    "time"
)

type clockKey struct{}

type Clock interface {
    Now() time.Time
}

type SystemClock struct{}

func (c *SystemClock) Now() time.Time {
    return time.Now()
}

type FixedClock struct {
    t time.Time
}

func (c *FixedClock) Now() time.Time {
    return c.t
}

func WithFixedClock(ctx context.Context, t time.Time) context.Context {
    return context.WithValue(ctx, clockKey{}, &FixedClock{t: t})
}

func GetClock(ctx context.Context) Clock {
    if clock, ok := ctx.Value(clockKey{}).(Clock); ok {
        return clock
    }
    return &SystemClock{}
}

// Helper for tests
func FixedTime(s string) time.Time {
    t, _ := time.Parse(time.RFC3339, s)
    return t
}
```

**Refactor Existing Code**:
```go
// libs/strategies/rsi_momentum.go (BEFORE)
func (s *RSIMomentum) Generate(market map[string]Quote) []Signal {
    now := time.Now()  // ❌ Non-deterministic
    // ...
}

// libs/strategies/rsi_momentum.go (AFTER)
func (s *RSIMomentum) Generate(ctx context.Context, market map[string]Quote) []Signal {
    clock := testing.GetClock(ctx)
    now := clock.Now()  // ✅ Deterministic in tests
    // ...
}
```

**Validation**:
```powershell
# Run tests with fixed clock
go test -v ./libs/strategies/... -run TestRSI_Deterministic

# Should pass 10/10 times with identical results
```

---

#### Task 0.4: CI Golden Test Runner (Day 3)
**Files to Create**:
```
.github/
  workflows/
    golden-tests.yml
scripts/
  run-golden-tests.ps1
```

**Implementation**:
```yaml
# .github/workflows/golden-tests.yml
name: Golden Tests

on:
  pull_request:
    paths:
      - 'libs/strategies/**'
      - 'services/jax-signal-generator/**'
      - 'services/jax-trade-executor/**'

jobs:
  golden-tests:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Start Services
        run: |
          docker-compose up -d jax-signal-generator jax-trade-executor
          sleep 10
      
      - name: Run Golden Tests
        run: |
          go test -v ./tests/golden/... -tags=golden
      
      - name: Compare with Baseline
        run: |
          # Fail if outputs don't match baseline
          ./scripts/compare-golden-outputs.sh
```

**Exit Criteria for Phase 0**:
- [x] Golden snapshots captured for signals, executions, orchestration
- [x] Replay harness validates 10/10 deterministic runs
- [x] All strategies refactored to use deterministic clock
- [x] CI enforces golden test matching on every PR

---

## Phase 1: Artifact System (Database + Domain Model)

**Duration**: Week 2  
**Risk**: Low (new code, no impact on existing services)  
**Goal**: Build artifact promotion infrastructure

### Tasks

#### Task 1.1: Database Migration (Day 4)
**File**: `db/migrations/002_artifacts.sql`

```sql
-- Strategy Artifacts
CREATE TABLE strategy_artifacts (
    artifact_id VARCHAR(255) PRIMARY KEY,
    schema_version VARCHAR(10) NOT NULL,
    strategy_name VARCHAR(100) NOT NULL,
    strategy_version VARCHAR(20) NOT NULL,
    code_ref VARCHAR(100) NOT NULL,
    params JSONB NOT NULL,
    data_window JSONB NOT NULL,
    validation JSONB NOT NULL,
    risk_profile JSONB NOT NULL,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload_hash VARCHAR(71) NOT NULL UNIQUE,
    payload_uri TEXT NOT NULL,
    signature TEXT,
    state VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    CHECK (state IN ('DRAFT', 'VALIDATED', 'REVIEWED', 'APPROVED', 'ACTIVE', 'DEPRECATED', 'REVOKED'))
);

CREATE INDEX idx_artifacts_state ON strategy_artifacts(state);
CREATE INDEX idx_artifacts_strategy ON strategy_artifacts(strategy_name, strategy_version);
CREATE INDEX idx_artifacts_created_at ON strategy_artifacts(created_at DESC);

-- Approvals
CREATE TABLE artifact_approvals (
    id SERIAL PRIMARY KEY,
    artifact_id VARCHAR(255) NOT NULL REFERENCES strategy_artifacts(artifact_id),
    approver_id VARCHAR(100) NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approval_type VARCHAR(20) NOT NULL,
    notes TEXT,
    CHECK (approval_type IN ('TECHNICAL', 'RISK', 'COMPLIANCE'))
);

-- Promotions (audit log)
CREATE TABLE artifact_promotions (
    id SERIAL PRIMARY KEY,
    artifact_id VARCHAR(255) NOT NULL REFERENCES strategy_artifacts(artifact_id),
    from_state VARCHAR(20) NOT NULL,
    to_state VARCHAR(20) NOT NULL,
    promoted_by VARCHAR(100) NOT NULL,
    promoted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT
);

-- Validation reports
CREATE TABLE artifact_validation_reports (
    id SERIAL PRIMARY KEY,
    artifact_id VARCHAR(255) NOT NULL REFERENCES strategy_artifacts(artifact_id),
    backtest_run_id UUID NOT NULL,
    report_uri TEXT NOT NULL,
    metrics JSONB NOT NULL,
    determinism_verified BOOLEAN NOT NULL DEFAULT FALSE,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Link trades to artifacts
ALTER TABLE trades 
ADD COLUMN artifact_id VARCHAR(255) REFERENCES strategy_artifacts(artifact_id),
ADD COLUMN artifact_hash VARCHAR(71);

CREATE INDEX idx_trades_artifact ON trades(artifact_id);
```

**Apply Migration**:
```powershell
# Run migration
$env:DATABASE_URL = "postgresql://jax:password@localhost:5433/jax"
migrate -path db/migrations -database $env:DATABASE_URL up

# Verify
psql $env:DATABASE_URL -c "\dt" | Select-String "strategy_artifacts"
```

---

#### Task 1.2: Artifact Domain Model (Day 4-5)
**Files to Create**:
```
internal/
  domain/
    artifacts/
      artifact.go
      artifact_test.go
      state.go
      validator.go
```

**Implementation**: (See ADR-0012-FIRST-CODE-CHANGES.md Change #5 for full code)

Key features:
- `ComputeHash()` - SHA-256 of canonical JSON
- `VerifyHash()` - Validate integrity
- `IsLoadableByTrader()` - Only APPROVED, non-REVOKED

**Validation**:
```powershell
go test -v ./internal/domain/artifacts/...

# Expected:
# PASS: TestArtifact_ComputeHash
# PASS: TestArtifact_VerifyHash
# PASS: TestArtifact_IsLoadableByTrader
```

---

#### Task 1.3: Ports + Postgres Implementation (Day 5-6)
**Files to Create**:
```
internal/
  ports/
    artifact_store.go
  integrations/
    postgres/
      artifact_store.go
      artifact_store_test.go
```

**Implementation**: (See ADR-0012-FIRST-CODE-CHANGES.md Change #6-7)

**Validation**:
```powershell
# Integration test with real Postgres
go test -v ./internal/integrations/postgres/... -tags=integration

# Expected:
# PASS: TestArtifactStore_SaveAndRetrieve
# PASS: TestArtifactStore_GetLatestApproved
# PASS: TestArtifactStore_Promote
```

---

#### Task 1.4: Seed Test Artifacts (Day 6)
**File**: `db/seeds/001_test_artifacts.sql`

```sql
-- Seed approved RSI artifact for testing
INSERT INTO strategy_artifacts (
    artifact_id, 
    schema_version, 
    strategy_name, 
    strategy_version,
    code_ref,
    params,
    data_window,
    validation,
    risk_profile,
    created_by,
    payload_hash,
    payload_uri,
    state
) VALUES (
    'strat_rsi_momentum_2026-02-01T00:00:00Z',
    '1.0.0',
    'rsi_momentum',
    '1.0.0',
    'git:abc123def456',
    '{"rsi_period": 14, "entry": 30, "exit": 70}'::jsonb,
    '{"from": "2024-01-01T00:00:00Z", "to": "2026-01-31T23:59:59Z", "symbols": ["AAPL", "MSFT"]}'::jsonb,
    '{"backtest_run_id": "550e8400-e29b-41d4-a716-446655440000", "metrics": {"sharpe": 1.42, "max_drawdown": 0.12, "win_rate": 0.54}, "determinism_seed": 42, "report_uri": "file:///tmp/report.html"}'::jsonb,
    '{"max_position_pct": 0.20, "max_daily_loss": 1000.0, "allowed_order_types": ["LMT"]}'::jsonb,
    'system',
    'sha256:' || encode(sha256('test_payload'::bytea), 'hex'),
    'file:///tmp/artifacts/rsi_momentum_v1.json',
    'APPROVED'
);

-- Seed approval record
INSERT INTO artifact_approvals (artifact_id, approver_id, approval_type, notes)
VALUES (
    'strat_rsi_momentum_2026-02-01T00:00:00Z',
    'admin',
    'TECHNICAL',
    'Initial approved strategy for testing'
);
```

**Exit Criteria for Phase 1**:
- [x] Database schema created and migrated
- [x] Artifact domain model with hash verification
- [x] Postgres store implementation with tests
- [x] Test artifact seeded and loadable

---

## Phase 2: Trader Runtime Skeleton

**Duration**: Week 2  
**Risk**: Low (new code, runs in parallel)  
**Goal**: Create cmd/trader that loads artifacts but doesn't execute yet

### Tasks

#### Task 2.1: Trader Entrypoint (Day 7)
**Files to Create**:
```
cmd/
  trader/
    main.go
    Dockerfile
    
internal/
  app/
    trader/
      runtime.go
      runtime_test.go
      config.go
```

**Implementation**: (See ADR-0012-FIRST-CODE-CHANGES.md Change #8-9)

**Key Features**:
- Load approved artifacts on startup
- Verify SHA-256 hash
- Log artifact metadata to audit trail
- Health check endpoint
- Graceful shutdown

**Validation**:
```powershell
# Build
go build -o bin/trader.exe cmd/trader/main.go

# Run
$env:DATABASE_URL = "postgresql://jax:password@localhost:5433/jax"
./bin/trader.exe

# Expected output:
# 🚀 Jax Trader Runtime starting...
# 📦 Loading approved strategy artifacts...
# ✅ Loaded artifact: strat_rsi_momentum_2026-02-01T00:00:00Z (hash: sha256:abc123...)
# ✅ Trader runtime started (no execution yet - skeleton only)
```

---

#### Task 2.2: Import Boundary Enforcement (Day 7)
**Files to Create**:
```
.github/
  workflows/
    trader-deps-check.yml
scripts/
  check-trader-deps.ps1
internal/
  policy/
    importcheck/
      rules.go
      rules_test.go
```

**Implementation**: (See ADR-0012-FIRST-CODE-CHANGES.md Change #10)

**Validation**:
```powershell
# Local check
./scripts/check-trader-deps.ps1

# Expected:
# 🔍 Checking trader runtime dependencies...
# ✅ No forbidden dependencies found

# Try importing forbidden package (should fail)
# In cmd/trader/main.go: import "jax-trading-assistant/libs/agent0"
./scripts/check-trader-deps.ps1
# ❌ FORBIDDEN DEPENDENCIES DETECTED: agent0
```

---

#### Task 2.3: Trader Docker Image (Day 8)
**File**: `cmd/trader/Dockerfile`

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /trader cmd/trader/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /trader .

EXPOSE 8100
HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8100/health || exit 1

CMD ["./trader"]
```

**Validation**:
```powershell
# Build image
docker build -t jax-trader:latest -f cmd/trader/Dockerfile .

# Run container
docker run -d `
    -p 8100:8100 `
    -e DATABASE_URL="postgresql://jax:password@host.docker.internal:5433/jax" `
    --name jax-trader `
    jax-trader:latest

# Check health
Invoke-RestMethod -Uri http://localhost:8100/health

# Expected: {"status": "healthy", "artifact_loaded": true}
```

**Exit Criteria for Phase 2**:
- [x] cmd/trader compiles and runs
- [x] Loads approved artifacts from database
- [x] Verifies SHA-256 hash on startup
- [x] Logs artifact metadata to stdout/audit
- [x] CI enforces import boundaries
- [x] Docker image builds and runs

---

## Phase 3: Collapse Internal HTTP Services

**Duration**: Week 3-4  
**Risk**: Medium (touching production code)  
**Goal**: Remove HTTP hops between jax-api, jax-orchestrator, jax-signal-generator

### Tasks

#### Task 3.1: Extract Orchestration Module (Day 9-10)
**Files to Create**:
```
internal/
  modules/
    orchestration/
      service.go
      service_test.go
      pipeline.go
```

**Move From**:
- `services/jax-orchestrator/internal/app/orchestrator.go` → `internal/modules/orchestration/service.go`

**Implementation**:
```go
// internal/modules/orchestration/service.go
package orchestration

import (
    "context"
    "jax-trading-assistant/internal/ports"
)

type Service struct {
    memoryClient ports.MemoryService
    agent0Client ports.Agent0Service
    dexterClient ports.DexterService
    db           *pgxpool.Pool
}

func NewService(
    memory ports.MemoryService,
    agent0 ports.Agent0Service,
    dexter ports.DexterService,
    db *pgxpool.Pool,
) *Service {
    return &Service{
        memoryClient: memory,
        agent0Client: agent0,
        dexterClient: dexter,
        db:           db,
    }
}

func (s *Service) Orchestrate(ctx context.Context, req *OrchestrationRequest) (*OrchestrationResponse, error) {
    // Move existing logic from jax-orchestrator
    
    // 1. Recall relevant memories
    memories, err := s.memoryClient.Recall(ctx, req.Symbol, req.SignalContext)
    if err != nil {
        return nil, err
    }
    
    // 2. Plan with Agent0
    plan, err := s.agent0Client.Plan(ctx, req.Signal, memories)
    if err != nil {
        return nil, err
    }
    
    // 3. Execute plan
    result, err := s.agent0Client.Execute(ctx, plan)
    if err != nil {
        return nil, err
    }
    
    // 4. Store orchestration run
    runID, err := s.db.Exec(ctx, `
        INSERT INTO orchestration_runs (signal_id, plan, result, created_at)
        VALUES ($1, $2, $3, NOW())
    `, req.SignalID, plan, result)
    
    return &OrchestrationResponse{
        RunID:  runID,
        Plan:   plan,
        Result: result,
    }, nil
}
```

**Validation**:
```powershell
# Unit test
go test -v ./internal/modules/orchestration/...

# Expected:
# PASS: TestService_Orchestrate
# PASS: TestService_Orchestrate_WithMemories
```

---

#### Task 3.2: Replace Signal Generator HTTP Call (Day 10-11)
**File**: `services/jax-signal-generator/internal/generator/generator.go`

**BEFORE**:
```go
// services/jax-signal-generator/internal/generator/generator.go
type Generator struct {
    orchestratorClient *orchestrator.HTTPClient  // ❌ HTTP client
}

func (g *Generator) generateAndOrchestrate(signal Signal) error {
    if signal.Confidence >= 0.75 {
        // HTTP call
        resp, err := g.orchestratorClient.Post("/orchestrate", signal)
        // ...
    }
}
```

**AFTER**:
```go
// services/jax-signal-generator/internal/generator/generator.go
type Generator struct {
    orchestrator ports.Orchestrator  // ✅ Interface
}

func (g *Generator) generateAndOrchestrate(ctx context.Context, signal Signal) error {
    if signal.Confidence >= 0.75 {
        // In-process call
        resp, err := g.orchestrator.Orchestrate(ctx, &orchestration.OrchestrationRequest{
            SignalID: signal.ID,
            Signal:   signal,
        })
        // ...
    }
}
```

**Validation**:
```powershell
# Start signal generator with in-process orchestration
go run services/jax-signal-generator/cmd/jax-signal-generator/main.go

# Trigger signal generation
# Watch logs for orchestration (no HTTP calls in logs)

# Compare database state with golden baseline
$current = psql $env:DATABASE_URL -c "SELECT * FROM orchestration_runs ORDER BY created_at DESC LIMIT 10"
$baseline = Get-Content tests/golden/orchestration/baseline-2026-02-13.json

Compare-Object $current $baseline
# Should be identical
```

---

#### Task 3.3: Replace API → Orchestrator HTTP Call (Day 11-12)
**File**: `services/jax-api/internal/infra/http/handlers_orchestration_v1.go`

**BEFORE**:
```go
func (h *OrchestrationHandler) StartOrchestration(w http.ResponseWriter, r *http.Request) {
    orchestratorURL := os.Getenv("JAX_ORCHESTRATOR_URL")
    
    // HTTP POST
    resp, err := http.Post(orchestratorURL+"/orchestrate", "application/json", payload)
    // ...
}
```

**AFTER**:
```go
func (h *OrchestrationHandler) StartOrchestration(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req orchestration.OrchestrationRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // In-process call
    resp, err := h.orchestrator.Orchestrate(r.Context(), &req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(resp)
}
```

**Validation**:
```powershell
# Contract test - verify API response unchanged
go test -v ./services/jax-api/internal/infra/http/... -run TestOrchestrationAPI_ContractParity

# Performance benchmark
go test -bench=BenchmarkOrchestration ./services/jax-api/...

# Expected improvement: 20-40ms faster (no HTTP serialization)
```

---

#### Task 3.4: Keep HTTP Endpoint as Compatibility Shim (Day 12)
**File**: `services/jax-orchestrator/cmd/jax-orchestrator-http/main.go`

```go
// Keep running HTTP server for backward compatibility during transition
func main() {
    // Initialize shared orchestration service
    orchestrationSvc := orchestration.NewService(memoryClient, agent0Client, dexterClient, db)
    
    // HTTP endpoint delegates to shared module
    http.HandleFunc("/orchestrate", func(w http.ResponseWriter, r *http.Request) {
        var req orchestration.OrchestrationRequest
        json.NewDecoder(r.Body).Decode(&req)
        
        resp, err := orchestrationSvc.Orchestrate(r.Context(), &req)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        
        json.NewEncoder(w).Encode(resp)
    })
    
    log.Println("Orchestrator HTTP shim running on :8091 (for backward compatibility)")
    http.ListenAndServe(":8091", nil)
}
```

**Exit Criteria for Phase 3**:
- [x] Orchestration module extracted to internal/modules/orchestration
- [x] Signal generator uses in-process orchestration (no HTTP)
- [x] jax-api uses in-process orchestration (no HTTP)
- [x] Database side effects match golden baseline
- [x] API contract parity tests pass
- [x] Latency improved (measured via benchmarks)
- [x] Old HTTP endpoint still works (backward compatibility)

---

## Phase 4: Migrate Trade Execution to Trader

**Duration**: Week 5-6  
**Risk**: **CRITICAL** (money at stake)  
**Goal**: Move trade execution logic into cmd/trader with parallel validation

### Tasks

#### Task 4.1: Extract Execution Module (Day 13-14)
**Files to Create**:
```
internal/
  modules/
    execution/
      engine.go
      engine_test.go
      replay_test.go
```

**Move From**:
- `services/jax-trade-executor/cmd/jax-trade-executor/main.go` → `internal/modules/execution/engine.go`
- `libs/trading/executor/executor.go` → `internal/modules/execution/executor.go`

**Key Changes**:
```go
// internal/modules/execution/engine.go
package execution

type Engine struct {
    ibBridge     ports.BrokerClient
    riskEngine   ports.RiskEngine
    auditLogger  ports.AuditLogger
    artifactID   string  // NEW: Track artifact for every decision
    artifactHash string
}

func (e *Engine) ExecuteSignal(ctx context.Context, signal Signal, artifact *artifacts.Artifact) (*Trade, error) {
    // Log artifact context (CRITICAL for audit)
    e.auditLogger.Log(ctx, "execution_start", map[string]interface{}{
        "signal_id":     signal.ID,
        "artifact_id":   artifact.ID,
        "artifact_hash": artifact.PayloadHash,
        "code_ref":      artifact.Strategy.CodeRef,
    })
    
    // Apply risk checks from artifact
    positionSize, err := e.riskEngine.CalculatePositionSize(
        signal,
        artifact.RiskProfile.MaxPositionPct,
        artifact.RiskProfile.MaxDailyLoss,
    )
    if err != nil {
        return nil, err
    }
    
    // Create order
    order := &Order{
        Symbol:   signal.Symbol,
        Action:   signal.Action,
        Quantity: positionSize,
        Type:     artifact.RiskProfile.AllowedOrderTypes[0], // Use artifact config
    }
    
    // Execute via IB Bridge
    trade, err := e.ibBridge.PlaceOrder(ctx, order)
    if err != nil {
        return nil, err
    }
    
    // Store trade with artifact reference
    trade.ArtifactID = artifact.ID
    trade.ArtifactHash = artifact.PayloadHash
    
    // Log completion
    e.auditLogger.Log(ctx, "execution_complete", map[string]interface{}{
        "trade_id":      trade.ID,
        "artifact_id":   artifact.ID,
        "position_size": positionSize,
    })
    
    return trade, nil
}
```

**Validation**:
```powershell
# Determinism replay tests
go test -v ./internal/modules/execution/... -run TestExecution_Determinism

# Expected: 100/100 replays produce identical results
```

---

#### Task 4.2: Wire Execution into Trader Runtime (Day 14-15)
**File**: `internal/app/trader/runtime.go`

```go
// internal/app/trader/runtime.go
type Runtime struct {
    cfg              *Config
    db               *pgxpool.Pool
    artifacts        map[string]*artifacts.Artifact  // strategy_name -> artifact
    
    // Modules
    marketDataSvc    ports.MarketDataService
    riskEngine       ports.RiskEngine
    executionEngine  *execution.Engine
    signalStore      ports.SignalStore
    auditLogger      ports.AuditLogger
}

func (r *Runtime) Run(ctx context.Context) error {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return nil
            
        case <-ticker.C:
            // 1. Fetch approved signals
            signals, err := r.signalStore.GetApprovedSignals(ctx)
            if err != nil {
                r.logger.Error("failed to fetch signals", "error", err)
                continue
            }
            
            // 2. Execute each signal
            for _, signal := range signals {
                // Get artifact for this strategy
                artifact, ok := r.artifacts[signal.StrategyName]
                if !ok {
                    r.logger.Warn("no approved artifact for strategy", "strategy", signal.StrategyName)
                    continue
                }
                
                // Execute with artifact
                trade, err := r.executionEngine.ExecuteSignal(ctx, signal, artifact)
                if err != nil {
                    r.logger.Error("execution failed", "signal_id", signal.ID, "error", err)
                    continue
                }
                
                r.logger.Info("trade executed",
                    "trade_id", trade.ID,
                    "signal_id", signal.ID,
                    "artifact_id", artifact.ID,
                    "position_size", trade.Quantity,
                )
            }
        }
    }
}
```

---

#### Task 4.3: Parallel Run (Shadow Mode) (Day 16-20)
**Goal**: Run new trader in read-only mode alongside old executor

**File**: `cmd/trader/shadow_mode.go`

```go
// shadow_mode.go
type ShadowValidator struct {
    legacyDB *pgxpool.Pool
    newDB    *pgxpool.Pool
}

func (v *ShadowValidator) CompareDecisions(ctx context.Context) error {
    // Query old executor's decisions
    legacyTrades, err := v.queryLegacyTrades(ctx)
    if err != nil {
        return err
    }
    
    // Query new trader's decisions
    shadowTrades, err := v.queryShadowTrades(ctx)
    if err != nil {
        return err
    }
    
    // Compare
    discrepancies := []Discrepancy{}
    for signalID, legacyTrade := range legacyTrades {
        shadowTrade, ok := shadowTrades[signalID]
        if !ok {
            discrepancies = append(discrepancies, Discrepancy{
                SignalID: signalID,
                Issue:    "shadow did not execute",
            })
            continue
        }
        
        // Compare position sizes (allow 0.01 rounding difference)
        if math.Abs(legacyTrade.Quantity - shadowTrade.Quantity) > 0.01 {
            discrepancies = append(discrepancies, Discrepancy{
                SignalID:      signalID,
                Issue:         "position size mismatch",
                LegacyValue:   legacyTrade.Quantity,
                ShadowValue:   shadowTrade.Quantity,
            })
        }
    }
    
    if len(discrepancies) > 0 {
        log.Printf("❌ Found %d discrepancies", len(discrepancies))
        for _, d := range discrepancies {
            log.Printf("  Signal %s: %s (legacy=%.2f, shadow=%.2f)", 
                d.SignalID, d.Issue, d.LegacyValue, d.ShadowValue)
        }
        return fmt.Errorf("discrepancies found")
    }
    
    log.Println("✅ Shadow validation passed - all decisions match")
    return nil
}
```

**Docker Compose Setup**:
```yaml
# docker-compose.yml (add to existing)
services:
  jax-trade-executor:
    # EXISTING: Keep running (writes to production tables)
    image: jax-trade-executor:latest
    
  trader-shadow:
    # NEW: Run in shadow mode (writes to separate tables for comparison)
    image: jax-trader:latest
    environment:
      - MODE=shadow
      - DATABASE_URL=postgresql://jax:password@postgres:5432/jax_shadow
      - COMPARE_WITH_LEGACY=true
    depends_on:
      - postgres
      - jax-trade-executor
```

**Validation Process**:
```powershell
# Day 16: Start shadow run
docker-compose up -d trader-shadow

# Day 16-20: Monitor for discrepancies (run every hour)
for ($i = 0; $i -lt 120; $i++) {
    Write-Host "Shadow validation run $i..."
    
    # Compare decisions
    $result = docker exec trader-shadow /app/compare-decisions.sh
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "❌ Discrepancy detected on run $i"
        exit 1
    }
    
    Write-Host "✅ Run $i: All decisions match"
    Start-Sleep -Seconds 3600  # 1 hour
}

Write-Host "🎉 Shadow validation complete: 120 hours with ZERO discrepancies"
```

---

#### Task 4.4: Cutover (Day 21)
**Goal**: Switch production traffic to new trader

**Cutover Steps**:
```powershell
# 1. Final validation
docker exec trader-shadow /app/compare-decisions.sh
if ($LASTEXITCODE -ne 0) {
    Write-Error "Cannot cutover - discrepancies found"
    exit 1
}

# 2. Promote shadow to production
docker-compose stop jax-trade-executor
docker-compose up -d trader-production

# 3. Monitor for 24 hours
# (Keep old executor running as hot standby)

# 4. If successful, decommission old executor
docker-compose rm -f jax-trade-executor
```

**Rollback Plan**:
```powershell
# If issues detected within 24 hours:
docker-compose stop trader-production
docker-compose up -d jax-trade-executor

# Investigate discrepancies
# Fix issues
# Retry shadow validation
```

**Exit Criteria for Phase 4**:
- [x] Execution engine extracted to internal/modules/execution
- [x] Trader runtime executes trades with artifact tracking
- [x] Shadow mode runs for 5 days (120 hours) with ZERO discrepancies
- [x] Position sizes match legacy executor within 0.01
- [x] All trades linked to artifact_id + artifact_hash in database
- [x] Audit logs include artifact context
- [x] Production cutover successful
- [x] Old executor decommissioned

---

## Phase 5: Research Runtime + Artifact Builder

**Duration**: Week 7-8  
**Risk**: Low (new features, no production impact)  
**Goal**: Complete the vision with research runtime

### Tasks

#### Task 5.1: Research Runtime Skeleton (Day 22-23)
**Files to Create**:
```
cmd/
  research/
    main.go
    Dockerfile
    
internal/
  app/
    research/
      runtime.go
      config.go
```

**Implementation**:
```go
// cmd/research/main.go
package main

import (
    "jax-trading-assistant/internal/app/research"
    "jax-trading-assistant/libs/agent0"
    "jax-trading-assistant/libs/dexter"
)

func main() {
    log.Println("🔬 Jax Research Runtime starting...")
    
    cfg, err := research.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }
    
    // Research runtime CAN import agent0/dexter (trader cannot)
    agent0Client := agent0.NewClient(cfg.Agent0URL)
    dexterClient := dexter.NewClient(cfg.DexterURL)
    hindsightClient := hindsight.NewClient(cfg.HindsightURL)
    
    // Initialize modules
    backtester := research.NewBacktester(cfg.BacktestConfig)
    artifactBuilder := research.NewArtifactBuilder(db, objectStore)
    
    // Start runtime
    runtime := research.NewRuntime(
        backtester,
        artifactBuilder,
        agent0Client,
        dexterClient,
        hindsightClient,
    )
    
    runtime.Run(context.Background())
}
```

---

#### Task 5.2: Backtest Engine (Day 23-25)
**Files to Create**:
```
internal/
  modules/
    backtest/
      engine.go
      engine_test.go
      determinism_test.go
```

**Implementation**:
```go
// internal/modules/backtest/engine.go
package backtest

type Engine struct {
    historicalData ports.MarketDataProvider
    strategyLoader ports.StrategyLoader
}

func (e *Engine) Run(ctx context.Context, config *BacktestConfig) (*Result, error) {
    // Set deterministic seed
    rand.Seed(config.Seed)
    
    // Load historical data
    data, err := e.historicalData.GetRange(config.DataWindow.From, config.DataWindow.To, config.Symbols)
    if err != nil {
        return nil, err
    }
    
    // Load strategy
    strategy, err := e.strategyLoader.Load(config.StrategyName, config.Params)
    if err != nil {
        return nil, err
    }
    
    // Run backtest
    trades := []Trade{}
    portfolio := NewPortfolio(config.InitialCapital)
    
    for _, bar := range data {
        // Generate signals with fixed clock
        ctx = testing.WithFixedClock(ctx, bar.Timestamp)
        signals := strategy.Generate(ctx, map[string]Quote{bar.Symbol: bar})
        
        // Execute signals
        for _, signal := range signals {
            trade := portfolio.Execute(signal)
            trades = append(trades, trade)
        }
    }
    
    // Calculate metrics
    metrics := calculateMetrics(trades, portfolio)
    
    return &Result{
        Metrics:         metrics,
        Trades:          trades,
        FinalPortfolio:  portfolio,
        DeterminismSeed: config.Seed,
    }, nil
}

func calculateMetrics(trades []Trade, portfolio *Portfolio) Metrics {
    returns := portfolio.GetReturns()
    
    return Metrics{
        Sharpe:       sharpeRatio(returns),
        MaxDrawdown:  maxDrawdown(returns),
        WinRate:      float64(countWinningTrades(trades)) / float64(len(trades)),
        TotalTrades:  len(trades),
        ProfitFactor: profitFactor(trades),
    }
}
```

**Validation**:
```powershell
# Determinism test
go test -v ./internal/modules/backtest/... -run TestBacktest_Determinism

# Run same backtest 10 times with same seed
# Expected: Identical results every time
```

---

#### Task 5.3: Artifact Builder (Day 25-26)
**Files to Create**:
```
internal/
  modules/
    artifacts/
      builder.go
      builder_test.go
```

**Implementation**:
```go
// internal/modules/artifacts/builder.go
package artifacts

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
    
    "jax-trading-assistant/internal/domain/artifacts"
)

type Builder struct {
    store       ports.ArtifactStore
    objectStore ports.ObjectStore
}

func (b *Builder) BuildFromBacktest(
    strategyName string,
    strategyVersion string,
    params map[string]interface{},
    backtestResult *backtest.Result,
    createdBy string,
) (*artifacts.Artifact, error) {
    // Get current git commit
    codeRef := getCurrentGitCommit()
    
    // Build artifact
    artifact := &artifacts.Artifact{
        ID:            fmt.Sprintf("strat_%s_%s", strategyName, time.Now().Format(time.RFC3339)),
        SchemaVersion: "1.0.0",
        Strategy: artifacts.StrategyRef{
            Name:    strategyName,
            Version: strategyVersion,
            CodeRef: fmt.Sprintf("git:%s", codeRef),
            Params:  params,
        },
        DataWindow: artifacts.DataWindow{
            From:    backtestResult.DataWindow.From,
            To:      backtestResult.DataWindow.To,
            Symbols: backtestResult.DataWindow.Symbols,
        },
        Validation: artifacts.ValidationReport{
            BacktestRunID:   uuid.New().String(),
            Metrics:         backtestResult.Metrics,
            DeterminismSeed: backtestResult.DeterminismSeed,
            ReportURI:       "", // Will set after upload
        },
        RiskProfile: artifacts.RiskProfile{
            MaxPositionPct:    0.20,
            MaxDailyLoss:      1000.0,
            AllowedOrderTypes: []string{"LMT"},
        },
        CreatedBy: createdBy,
        CreatedAt: time.Now(),
        State:     artifacts.StateDraft,
    }
    
    // Compute hash
    canonical, err := artifact.CanonicalJSON()
    if err != nil {
        return nil, err
    }
    
    hash := sha256.Sum256(canonical)
    artifact.PayloadHash = "sha256:" + hex.EncodeToString(hash[:])
    
    // Upload blob to object store
    blobKey := fmt.Sprintf("artifacts/%s.json", artifact.ID)
    uri, err := b.objectStore.Put(context.Background(), blobKey, canonical)
    if err != nil {
        return nil, err
    }
    artifact.PayloadURI = uri
    
    // Upload backtest report
    reportHTML := generateBacktestReport(backtestResult)
    reportKey := fmt.Sprintf("reports/%s.html", artifact.ID)
    reportURI, err := b.objectStore.Put(context.Background(), reportKey, []byte(reportHTML))
    if err != nil {
        return nil, err
    }
    artifact.Validation.ReportURI = reportURI
    
    // Save to database
    if err := b.store.Save(context.Background(), artifact); err != nil {
        return nil, err
    }
    
    log.Printf("✅ Artifact created: %s (hash: %s)", artifact.ID, artifact.PayloadHash[:16]+"...")
    
    return artifact, nil
}
```

---

#### Task 5.4: Approval CLI Tool (Day 27)
**Files to Create**:
```
cmd/
  artifact-approver/
    main.go
```

**Implementation**:
```go
// cmd/artifact-approver/main.go
package main

import (
    "flag"
    "fmt"
)

func main() {
    artifactID := flag.String("id", "", "Artifact ID to approve")
    approverID := flag.String("approver", "", "Approver identity")
    approvalType := flag.String("type", "TECHNICAL", "Approval type (TECHNICAL, RISK, COMPLIANCE)")
    notes := flag.String("notes", "", "Approval notes")
    flag.Parse()
    
    if *artifactID == "" || *approverID == "" {
        log.Fatal("Usage: artifact-approver -id <artifact_id> -approver <name>")
    }
    
    // Load artifact
    artifact, err := store.GetByID(context.Background(), *artifactID)
    if err != nil {
        log.Fatal(err)
    }
    
    // Verify hash
    if err := artifact.VerifyHash(); err != nil {
        log.Fatal("❌ Hash verification failed:", err)
    }
    
    // Show backtest report
    fmt.Printf("Artifact: %s\n", artifact.ID)
    fmt.Printf("Strategy: %s v%s\n", artifact.Strategy.Name, artifact.Strategy.Version)
    fmt.Printf("Backtest Metrics:\n")
    fmt.Printf("  Sharpe: %.2f\n", artifact.Validation.Metrics.Sharpe)
    fmt.Printf("  Max Drawdown: %.2f%%\n", artifact.Validation.Metrics.MaxDrawdown*100)
    fmt.Printf("  Win Rate: %.2f%%\n", artifact.Validation.Metrics.WinRate*100)
    fmt.Printf("Report: %s\n", artifact.Validation.ReportURI)
    
    // Confirm
    fmt.Print("\nApprove this artifact? (yes/no): ")
    var confirm string
    fmt.Scanln(&confirm)
    
    if confirm != "yes" {
        fmt.Println("Approval cancelled")
        return
    }
    
    // Record approval
    err = store.Promote(context.Background(), *artifactID, artifacts.StateApproved, *approverID, *notes)
    if err != nil {
        log.Fatal("Approval failed:", err)
    }
    
    fmt.Println("✅ Artifact approved and ready for trader runtime")
}
```

**Usage**:
```powershell
# Approve an artifact
go run cmd/artifact-approver/main.go `
    -id strat_rsi_momentum_2026-02-13T15:30:00Z `
    -approver john.doe `
    -type TECHNICAL `
    -notes "Backtest metrics look solid"

# Output:
# Artifact: strat_rsi_momentum_2026-02-13T15:30:00Z
# Strategy: rsi_momentum v1.1.0
# Backtest Metrics:
#   Sharpe: 1.42
#   Max Drawdown: 12.00%
#   Win Rate: 54.00%
# Report: s3://jax-artifacts/reports/strat_rsi_momentum_2026-02-13T15:30:00Z.html
# 
# Approve this artifact? (yes/no): yes
# ✅ Artifact approved and ready for trader runtime
```

**Exit Criteria for Phase 5**:
- [x] cmd/research entrypoint created
- [x] Backtest engine with deterministic execution
- [x] Artifact builder creates artifacts from backtests
- [x] Approval CLI tool for promoting artifacts
- [x] Research runtime can call Agent0/Dexter/Hindsight
- [x] Trader runtime CANNOT import research packages (CI enforced)

---

## Phase 6: Decommission Old Services

**Duration**: Week 8  
**Risk**: Medium (deployment changes)  
**Goal**: Remove deprecated services, update deployment

### Tasks

#### Task 6.1: Update docker-compose.yml (Day 28)
**File**: `docker-compose.yml`

**BEFORE** (11 services):
```yaml
services:
  jax-api:
  jax-orchestrator:
  jax-signal-generator:
  jax-trade-executor:
  jax-market:
  jax-memory:
  agent0-service:
  ib-bridge:
  hindsight:
  postgres:
  prometheus:
```

**AFTER** (6 services):
```yaml
version: '3.8'

services:
  # Production trader runtime
  trader:
    build:
      context: .
      dockerfile: cmd/trader/Dockerfile
    ports:
      - "8100:8100"
    environment:
      - DATABASE_URL=postgresql://jax:${POSTGRES_PASSWORD}@postgres:5432/jax
      - IB_BRIDGE_URL=http://ib-bridge:8092
    depends_on:
      postgres:
        condition: service_healthy
      ib-bridge:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8100/health"]
      interval: 30s
      timeout: 3s
      retries: 3
  
  # Research runtime
  research:
    build:
      context: .
      dockerfile: cmd/research/Dockerfile
    ports:
      - "8101:8101"
    environment:
      - DATABASE_URL=postgresql://jax:${POSTGRES_PASSWORD}@postgres:5432/jax
      - AGENT0_SERVICE_URL=http://agent0-service:8093
      - DEXTER_SERVICE_URL=http://dexter:8094
      - HINDSIGHT_URL=http://hindsight:8888
    depends_on:
      - postgres
      - agent0-service
      - hindsight
  
  # External boundaries (keep)
  ib-bridge:
    build: ./services/ib-bridge
    ports:
      - "8092:8092"
    environment:
      - IB_GATEWAY_HOST=${IB_GATEWAY_HOST:-127.0.0.1}
      - IB_GATEWAY_PORT=${IB_GATEWAY_PORT:-4001}
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8092/health"]
  
  hindsight:
    build: ./services/hindsight
    ports:
      - "8888:8888"
  
  agent0-service:
    build: ./services/agent0-service
    ports:
      - "8093:8093"
    environment:
      - MEMORY_SERVICE_URL=http://trader:8100/memory
  
  # Infrastructure
  postgres:
    image: postgres:15
    ports:
      - "5433:5432"
    environment:
      - POSTGRES_USER=jax
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=jax
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U jax"]
  
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./observability/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
  
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}

volumes:
  postgres-data:
```

**Validation**:
```powershell
# Stop old services
docker-compose down

# Start new stack
docker-compose up -d

# Check health
docker-compose ps

# Expected:
# trader         running (healthy)
# research       running
# ib-bridge      running (healthy)
# postgres       running (healthy)
# prometheus     running
# grafana        running
```

---

#### Task 6.2: Remove Deprecated Service Directories (Day 28)
**Directories to Archive** (not delete - keep for reference):
```
services/
  jax-api/              → Move to archive/jax-api-deprecated/
  jax-orchestrator/     → Move to archive/jax-orchestrator-deprecated/
  jax-signal-generator/ → Move to archive/jax-signal-generator-deprecated/
  jax-trade-executor/   → Move to archive/jax-trade-executor-deprecated/
  jax-market/           → Move to archive/jax-market-deprecated/
  jax-memory/           → Move to archive/jax-memory-deprecated/
```

**Keep**:
```
services/
  ib-bridge/            ✅ External boundary
  hindsight/            ✅ Vendored Python service
  agent0-service/       ✅ Research integration
```

```powershell
# Archive deprecated services
New-Item -ItemType Directory -Path archive -Force
Move-Item services/jax-api archive/jax-api-deprecated
Move-Item services/jax-orchestrator archive/jax-orchestrator-deprecated
Move-Item services/jax-signal-generator archive/jax-signal-generator-deprecated
Move-Item services/jax-trade-executor archive/jax-trade-executor-deprecated
Move-Item services/jax-market archive/jax-market-deprecated
Move-Item services/jax-memory archive/jax-memory-deprecated

# Add README
@"
# Deprecated Services

These services were part of the microservices architecture (pre-Feb 2026).
They have been replaced by the modular monolith architecture (cmd/trader + cmd/research).

Kept for historical reference only - DO NOT USE IN PRODUCTION.

See Docs/ADR-0012-two-runtime-modular-monolith.md for migration details.
"@ | Out-File -FilePath archive/README.md
```

---

#### Task 6.3: Update Documentation (Day 29)
**Files to Update**:
1. `README.md` - Update architecture section
2. `Docs/QUICKSTART.md` - New startup instructions
3. `Docs/ARCHITECTURE.md` - Reflect new structure
4. `Docs/DEPLOYMENT.md` - New deployment guide

**Example**:
```markdown
<!-- README.md -->
## Architecture

Jax uses a **modular monolith** architecture with two runtime entrypoints:

### Production Trader Runtime (`cmd/trader`)
- Executes approved trading strategies only
- Deterministic, auditable decisions
- Loads strategies from artifact store
- Verifies SHA-256 hashes on startup
- Strict risk controls

### Research Runtime (`cmd/research`)
- Backtesting and strategy discovery
- Agent0/Dexter/Hindsight integration
- Generates strategy artifacts
- Publishes to approval queue

### Artifact Promotion Gate
Research strategies must be approved before trader can use them:
1. Research generates artifact + backtest report
2. Human reviewer approves (via CLI tool)
3. Trader loads only APPROVED artifacts
4. Every trade linked to artifact hash (audit trail)

### Quick Start
```powershell
# Start production stack
docker-compose up -d

# Services:
# - trader (port 8100)
# - research (port 8101)
# - ib-bridge (port 8092)
# - postgres (port 5433)
```

---

#### Task 6.4: Operational Runbook (Day 29-30)
**File**: `Docs/OPERATIONS.md`

```markdown
# Operational Runbook

## Daily Operations

### Starting the Platform
```powershell
# 1. Ensure IB Gateway is running
# 2. Start services
docker-compose up -d

# 3. Verify health
docker-compose ps
Invoke-RestMethod http://localhost:8100/health  # Trader
Invoke-RestMethod http://localhost:8101/health  # Research
```

### Monitoring

**Trader Health**:
```powershell
# Check artifact loaded
Invoke-RestMethod http://localhost:8100/health | ConvertFrom-Json

# Expected:
# {
#   "status": "healthy",
#   "artifacts_loaded": 2,
#   "last_execution": "2026-02-13T15:30:00Z"
# }
```

**Metrics** (Grafana):
- Dashboard: http://localhost:3001
- Login: admin / ${GRAFANA_PASSWORD}
- Key metrics:
  - Trader: Executions/min, Position sizes, API latency
  - Research: Backtest duration, Artifact generation rate

### Approving New Strategies

```powershell
# 1. Research runtime generates artifact
# (check logs for artifact ID)

# 2. Review backtest report
# Navigate to: s3://jax-artifacts/reports/{artifact_id}.html

# 3. Approve
go run cmd/artifact-approver/main.go `
    -id {artifact_id} `
    -approver "your.name" `
    -type TECHNICAL `
    -notes "Reviewed backtest metrics"

# 4. Restart trader to load new artifact
docker-compose restart trader
```

### Emergency Procedures

**Revoke Strategy**:
```powershell
# If strategy is behaving badly in production
psql $env:DATABASE_URL -c `
    "UPDATE strategy_artifacts SET state = 'REVOKED' WHERE artifact_id = '{id}'"

# Restart trader (will no longer load revoked artifact)
docker-compose restart trader
```

**Rollback to Previous Artifact**:
```powershell
# Promote older artifact back to APPROVED
psql $env:DATABASE_URL -c `
    "UPDATE strategy_artifacts SET state = 'APPROVED' 
     WHERE artifact_id = '{previous_artifact_id}'"

# Revoke current artifact
psql $env:DATABASE_URL -c `
    "UPDATE strategy_artifacts SET state = 'REVOKED' 
     WHERE artifact_id = '{current_artifact_id}'"

# Restart trader
docker-compose restart trader
```

### Troubleshooting

**Trader won't start**:
```powershell
# Check logs
docker logs jax-trader

# Common issues:
# - No approved artifacts in database
# - Database connection failed
# - IB Bridge unavailable

# Solution: Seed test artifact
psql $env:DATABASE_URL -f db/seeds/001_test_artifacts.sql
docker-compose restart trader
```

**Determinism test failing**:
```powershell
# Run replay test
go test -v ./tests/replay/... -run TestDeterminism

# If fails:
# 1. Check for time.Now() usage (should use testing.Clock)
# 2. Check for map iteration (use sorted keys)
# 3. Check for external API calls (should be mocked in tests)
```
```

**Exit Criteria for Phase 6**:
- [x] docker-compose.yml updated (6 services, down from 11)
- [x] Deprecated services archived
- [x] Documentation updated
- [x] Operational runbook created
- [x] Migration complete end-to-end test passed

---

## Final Validation & Acceptance

**Day 30**: End-to-end system test

### Test Scenario 1: Research → Approval → Trader
```powershell
# 1. Generate new strategy artifact via research runtime
docker exec jax-research /app/run-backtest.sh rsi_momentum --params "rsi_period=14"

# Expected: Artifact created in DRAFT state

# 2. Approve artifact
docker exec jax-research go run cmd/artifact-approver/main.go `
    -id {generated_artifact_id} `
    -approver test.user

# Expected: Artifact promoted to APPROVED

# 3. Restart trader
docker-compose restart trader

# Expected: Trader loads new artifact, logs hash

# 4. Verify trader uses new artifact
$health = Invoke-RestMethod http://localhost:8100/health | ConvertFrom-Json
$health.artifacts_loaded -contains {generated_artifact_id}

# Expected: true
```

### Test Scenario 2: Emergency Revocation
```powershell
# 1. Revoke artifact mid-trading
psql $env:DATABASE_URL -c `
    "UPDATE strategy_artifacts SET state = 'REVOKED' WHERE artifact_id = '{id}'"

# 2. Trader detects revocation (next health check)
docker logs jax-trader | Select-String "artifact revoked"

# Expected: Trader stops using revoked artifact

# 3. Verify no new trades with revoked artifact
psql $env:DATABASE_URL -c `
    "SELECT COUNT(*) FROM trades 
     WHERE artifact_id = '{id}' 
     AND created_at > NOW() - INTERVAL '5 minutes'"

# Expected: 0
```

### Test Scenario 3: Deterministic Replay
```powershell
# Replay trades from last week
go test -v ./tests/replay/... -run TestReplay_LastWeek

# Expected: All replayed trades match historical decisions exactly
```

---

## Success Metrics

After Phase 6 completion:

| Metric | Before (Microservices) | After (Modular Monolith) | Improvement |
|--------|------------------------|--------------------------|-------------|
| **Latency (signal → execution)** | 150ms | 80ms | 47% faster |
| **Deployment complexity** | 11 containers | 6 containers | 45% simpler |
| **Deterministic testing** | ❌ Impossible | ✅ Enabled | N/A |
| **Artifact audit trail** | ❌ None | ✅ Complete | N/A |
| **Research/prod separation** | ⚠️ Mixed | ✅ Enforced | N/A |
| **Import boundary violations** | ⚠️ Possible | ❌ CI blocks | N/A |

---

## Timeline Summary

| Phase | Duration | Risk | Can Rollback? |
|-------|----------|------|---------------|
| 0: Foundation | Week 1 | Very Low | N/A (new code) |
| 1: Artifact System | Week 2 | Low | ✅ Yes (tables only) |
| 2: Trader Skeleton | Week 2 | Low | ✅ Yes (parallel) |
| 3: Collapse HTTP | Week 3-4 | Medium | ✅ Yes (shims remain) |
| 4: Trade Execution | Week 5-6 | **CRITICAL** | ✅ Yes (5-day shadow) |
| 5: Research Runtime | Week 7-8 | Low | ✅ Yes (additive) |
| 6: Decommission | Week 8 | Medium | ✅ Yes (archives kept) |

**Total: 8 weeks** (with AI assistance; 6-9 months for human team)

---

## Risk Register

| Risk | Mitigation | Owner | Status |
|------|------------|-------|--------|
| Behavior drift during migration | Golden tests + replay harness (Phase 0) | Dev | ✅ Mitigated |
| Trade execution regression | 5-day shadow validation (Phase 4) | Dev | ✅ Mitigated |
| Artifact hash collision | SHA-256 (cryptographically secure) | - | ✅ Negligible |
| Import boundary violation | CI enforcement (Phase 2) | CI/CD | ✅ Mitigated |
| Determinism failure | Fixed clock, seeded RNG (Phase 0) | Dev | ✅ Mitigated |
| Production outage | Parallel runs, rollback plans | Ops | ✅ Mitigated |

---

## Approval & Sign-Off

**Ready to proceed?**

Review this plan and confirm:
- [ ] Timeline acceptable (8 weeks)
- [ ] Risk mitigations sufficient
- [ ] Checkpoints clear
- [ ] Rollback strategy understood
- [ ] Resources available

**Next Step**: Begin Phase 0 (Foundation & Safety Net) - Week 1

Let me know if any adjustments needed before we start implementation! 🚀
