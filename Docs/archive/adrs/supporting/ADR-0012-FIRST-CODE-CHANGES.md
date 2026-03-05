# ADR-0012: Concrete First Code Changes

**Last Updated**: 2026-02-13  
**Status**: Phase 0 - Baseline & Testing Infrastructure

This document outlines the **first 10 code changes** to implement ADR-0012's modular monolith migration.

---

## Phase 0: Baseline + Testing Infrastructure (Weeks 1-3)

### Change 1: Create Golden Test Infrastructure

**Purpose**: Capture current behavior before refactoring

**Files to Create**:
```
tests/
  golden/
    signals/
      README.md
      baseline-2026-02-13.json
    executions/
      README.md
      baseline-2026-02-13.json
  replay/
    harness.go
    harness_test.go
    fixtures/
      market_aapl_2026-01-15.json
      market_msft_2026-01-20.json
```

**Implementation**:

```go
// tests/golden/capture.go
package golden

import (
    "encoding/json"
    "os"
    "time"
)

type SignalSnapshot struct {
    Timestamp   time.Time
    MarketData  map[string]Quote
    Signals     []Signal
    Metadata    map[string]string
}

func CaptureSignalGeneration() error {
    // 1. Query jax-signal-generator for current signals
    resp, err := http.Get("http://localhost:8096/api/v1/signals")
    if err != nil {
        return err
    }
    
    var signals []Signal
    json.NewDecoder(resp.Body).Decode(&signals)
    
    // 2. Query market data snapshot
    marketResp, err := http.Get("http://localhost:8095/api/v1/market/snapshot")
    if err != nil {
        return err
    }
    
    var market map[string]Quote
    json.NewDecoder(marketResp.Body).Decode(&market)
    
    // 3. Save golden snapshot
    snapshot := SignalSnapshot{
        Timestamp:  time.Now(),
        MarketData: market,
        Signals:    signals,
        Metadata: map[string]string{
            "version": "baseline-2026-02-13",
        },
    }
    
    f, err := os.Create("tests/golden/signals/baseline-2026-02-13.json")
    if err != nil {
        return err
    }
    defer f.Close()
    
    encoder := json.NewEncoder(f)
    encoder.SetIndent("", "  ")
    return encoder.Encode(snapshot)
}
```

**PowerShell Script** to capture baseline:
```powershell
# scripts/capture-golden-baseline.ps1

Write-Host "Capturing golden baseline for signal generation..."

# Ensure services are running
docker-compose up -d jax-signal-generator jax-market postgres

# Wait for services to be healthy
Start-Sleep -Seconds 10

# Run capture
go run tests/golden/capture.go

Write-Host "✅ Golden baseline captured to tests/golden/signals/baseline-2026-02-13.json"
```

---

### Change 2: Create Replay Harness

**Purpose**: Deterministic replay of historical signals

**File**: `tests/replay/harness.go`

```go
package replay

import (
    "context"
    "encoding/json"
    "os"
    "time"
)

type Fixture struct {
    Name            string
    MarketSnapshot  map[string]Quote
    StrategyParams  map[string]interface{}
    ExpectedSignals []Signal
    FixedTime       time.Time
    Seed            int64
}

func LoadFixture(path string) (*Fixture, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    
    var fixture Fixture
    if err := json.NewDecoder(f).Decode(&fixture); err != nil {
        return nil, err
    }
    
    return &fixture, nil
}

func ReplayStrategy(ctx context.Context, fixture *Fixture, strategy StrategyExecutor) ([]Signal, error) {
    // Set deterministic clock
    clockOverride := FixedClock{t: fixture.FixedTime}
    ctx = context.WithValue(ctx, "clock", &clockOverride)
    
    // Set deterministic RNG seed
    if fixture.Seed != 0 {
        rand.Seed(fixture.Seed)
    }
    
    // Execute strategy with fixed inputs
    signals := strategy.Generate(ctx, fixture.MarketSnapshot, fixture.StrategyParams)
    
    return signals, nil
}

func VerifyReplayMatch(actual, expected []Signal) error {
    if len(actual) != len(expected) {
        return fmt.Errorf("signal count mismatch: got %d, want %d", len(actual), len(expected))
    }
    
    for i := range actual {
        if !signalsEqual(actual[i], expected[i]) {
            return fmt.Errorf("signal %d mismatch: got %+v, want %+v", i, actual[i], expected[i])
        }
    }
    
    return nil
}
```

**Test**:
```go
// tests/replay/harness_test.go
func TestReplay_DeterministicSignalGeneration(t *testing.T) {
    fixture, err := LoadFixture("fixtures/market_aapl_2026-01-15.json")
    require.NoError(t, err)
    
    strategy := strategies.NewRSIMomentum()
    
    // Run replay 10 times
    var results [][]Signal
    for i := 0; i < 10; i++ {
        signals, err := ReplayStrategy(context.Background(), fixture, strategy)
        require.NoError(t, err)
        results = append(results, signals)
    }
    
    // All runs must produce identical signals
    for i := 1; i < len(results); i++ {
        assert.NoError(t, VerifyReplayMatch(results[i], results[0]))
    }
}
```

---

### Change 3: Create Deterministic Clock Interface

**Purpose**: Ban `time.Now()` in business logic

**File**: `libs/testing/clock.go`

```go
package testing

import "time"

// Clock provides testable time access
type Clock interface {
    Now() time.Time
}

// SystemClock uses real system time
type SystemClock struct{}

func (c *SystemClock) Now() time.Time {
    return time.Now()
}

// FixedClock returns a fixed time (for tests)
type FixedClock struct {
    t time.Time
}

func (c *FixedClock) Now() time.Time {
    return c.t
}

// NewFixedClock creates a clock frozen at the given time
func NewFixedClock(t time.Time) *FixedClock {
    return &FixedClock{t: t}
}

// GetClock extracts clock from context or returns system clock
func GetClock(ctx context.Context) Clock {
    if clock, ok := ctx.Value("clock").(Clock); ok {
        return clock
    }
    return &SystemClock{}
}
```

**Refactor Signal Generator** to use Clock:
```go
// libs/strategies/rsi_momentum.go (BEFORE)
func (s *RSIMomentum) Generate(market map[string]Quote, params map[string]interface{}) []Signal {
    now := time.Now()  // ❌ NON-DETERMINISTIC
    // ...
}

// libs/strategies/rsi_momentum.go (AFTER)
func (s *RSIMomentum) Generate(ctx context.Context, market map[string]Quote, params map[string]interface{}) []Signal {
    clock := testing.GetClock(ctx)
    now := clock.Now()  // ✅ DETERMINISTIC in tests
    // ...
}
```

---

## Phase 1: Artifact Database Schema (Week 4)

### Change 4: Create Artifact Database Migration

**File**: `db/migrations/002_artifacts.sql`

```sql
-- Strategy Artifacts: Immutable approved strategy packages
CREATE TABLE strategy_artifacts (
    artifact_id VARCHAR(255) PRIMARY KEY,
    schema_version VARCHAR(10) NOT NULL,
    strategy_name VARCHAR(100) NOT NULL,
    strategy_version VARCHAR(20) NOT NULL,
    code_ref VARCHAR(100) NOT NULL,  -- git:commit-sha
    params JSONB NOT NULL,
    data_window JSONB NOT NULL,
    validation JSONB NOT NULL,
    risk_profile JSONB NOT NULL,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload_hash VARCHAR(71) NOT NULL UNIQUE,  -- 'sha256:' + 64 hex
    payload_uri TEXT NOT NULL,  -- S3/MinIO URI
    signature TEXT,  -- Optional KMS/GPG signature
    state VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    CHECK (state IN ('DRAFT', 'VALIDATED', 'REVIEWED', 'APPROVED', 'ACTIVE', 'DEPRECATED', 'REVOKED'))
);

CREATE INDEX idx_artifacts_state ON strategy_artifacts(state);
CREATE INDEX idx_artifacts_strategy ON strategy_artifacts(strategy_name, strategy_version);
CREATE INDEX idx_artifacts_created_at ON strategy_artifacts(created_at DESC);

-- Artifact Approvals: Track approver identity and timestamp
CREATE TABLE artifact_approvals (
    id SERIAL PRIMARY KEY,
    artifact_id VARCHAR(255) NOT NULL REFERENCES strategy_artifacts(artifact_id),
    approver_id VARCHAR(100) NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approval_type VARCHAR(20) NOT NULL,  -- 'TECHNICAL', 'RISK', 'COMPLIANCE'
    notes TEXT,
    CHECK (approval_type IN ('TECHNICAL', 'RISK', 'COMPLIANCE'))
);

CREATE INDEX idx_approvals_artifact ON artifact_approvals(artifact_id);

-- Artifact Promotions: State transition audit log
CREATE TABLE artifact_promotions (
    id SERIAL PRIMARY KEY,
    artifact_id VARCHAR(255) NOT NULL REFERENCES strategy_artifacts(artifact_id),
    from_state VARCHAR(20) NOT NULL,
    to_state VARCHAR(20) NOT NULL,
    promoted_by VARCHAR(100) NOT NULL,
    promoted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason TEXT
);

CREATE INDEX idx_promotions_artifact ON artifact_promotions(artifact_id);
CREATE INDEX idx_promotions_time ON artifact_promotions(promoted_at DESC);

-- Artifact Validation Reports: Backtest results and reports
CREATE TABLE artifact_validation_reports (
    id SERIAL PRIMARY KEY,
    artifact_id VARCHAR(255) NOT NULL REFERENCES strategy_artifacts(artifact_id),
    backtest_run_id UUID NOT NULL,
    report_uri TEXT NOT NULL,  -- S3/MinIO URI to HTML/PDF report
    metrics JSONB NOT NULL,
    determinism_verified BOOLEAN NOT NULL DEFAULT FALSE,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_validation_artifact ON artifact_validation_reports(artifact_id);

-- Link trades to artifacts for audit trail
ALTER TABLE trades 
ADD COLUMN artifact_id VARCHAR(255) REFERENCES strategy_artifacts(artifact_id),
ADD COLUMN artifact_hash VARCHAR(71);

CREATE INDEX idx_trades_artifact ON trades(artifact_id);

COMMENT ON TABLE strategy_artifacts IS 'Immutable versioned strategy packages approved for production';
COMMENT ON COLUMN strategy_artifacts.payload_hash IS 'SHA-256 hash of canonical JSON for verification';
COMMENT ON COLUMN strategy_artifacts.payload_uri IS 'Object store URI (s3://bucket/artifacts/...)';
COMMENT ON COLUMN trades.artifact_id IS 'Which artifact made this trading decision (audit trail)';
```

**Apply Migration**:
```powershell
# scripts/migrate.ps1 (update)
$env:DATABASE_URL = "postgresql://jax:password@localhost:5433/jax"
migrate -path db/migrations -database $env:DATABASE_URL up
```

---

### Change 5: Create Artifact Domain Model

**File**: `internal/domain/artifacts/artifact.go`

```go
package artifacts

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"
)

type State string

const (
    StateDraft      State = "DRAFT"
    StateValidated  State = "VALIDATED"
    StateReviewed   State = "REVIEWED"
    StateApproved   State = "APPROVED"
    StateActive     State = "ACTIVE"
    StateDeprecated State = "DEPRECATED"
    StateRevoked    State = "REVOKED"
)

type Artifact struct {
    ID            string          `json:"artifact_id"`
    SchemaVersion string          `json:"schema_version"`
    Strategy      StrategyRef     `json:"strategy"`
    DataWindow    DataWindow      `json:"data_window"`
    Validation    ValidationReport `json:"validation"`
    RiskProfile   RiskProfile     `json:"risk_profile"`
    CreatedBy     string          `json:"created_by"`
    CreatedAt     time.Time       `json:"created_at"`
    PayloadHash   string          `json:"hash"`
    PayloadURI    string          `json:"payload_uri,omitempty"`
    Signature     string          `json:"signature,omitempty"`
    State         State           `json:"state"`
}

type StrategyRef struct {
    Name    string                 `json:"name"`
    Version string                 `json:"version"`
    CodeRef string                 `json:"code_ref"`  // git:commit-sha
    Params  map[string]interface{} `json:"params"`
}

type DataWindow struct {
    From    time.Time `json:"from"`
    To      time.Time `json:"to"`
    Symbols []string  `json:"symbols"`
}

type ValidationReport struct {
    BacktestRunID   string  `json:"backtest_run_id"`
    Metrics         Metrics `json:"metrics"`
    DeterminismSeed int64   `json:"determinism_seed"`
    ReportURI       string  `json:"report_uri"`
}

type Metrics struct {
    Sharpe       float64 `json:"sharpe"`
    MaxDrawdown  float64 `json:"max_drawdown"`
    WinRate      float64 `json:"win_rate"`
    TotalTrades  int     `json:"total_trades"`
    ProfitFactor float64 `json:"profit_factor"`
}

type RiskProfile struct {
    MaxPositionPct      float64  `json:"max_position_pct"`
    MaxDailyLoss        float64  `json:"max_daily_loss"`
    AllowedOrderTypes   []string `json:"allowed_order_types"`
    MaxPositionSizeUSD  float64  `json:"max_position_size_usd,omitempty"`
}

// CanonicalJSON generates deterministic JSON for hashing
func (a *Artifact) CanonicalJSON() ([]byte, error) {
    // Marshal without indentation, sorted keys
    return json.Marshal(a)
}

// ComputeHash calculates SHA-256 of canonical JSON
func (a *Artifact) ComputeHash() (string, error) {
    canonical, err := a.CanonicalJSON()
    if err != nil {
        return "", err
    }
    
    hash := sha256.Sum256(canonical)
    return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// VerifyHash checks if artifact hash matches content
func (a *Artifact) VerifyHash() error {
    computed, err := a.ComputeHash()
    if err != nil {
        return err
    }
    
    if computed != a.PayloadHash {
        return fmt.Errorf("hash mismatch: expected %s, got %s", a.PayloadHash, computed)
    }
    
    return nil
}

// IsLoadableByTrader checks if artifact can be loaded by production trader
func (a *Artifact) IsLoadableByTrader() bool {
    return a.State == StateApproved && a.State != StateRevoked
}
```

---

### Change 6: Create Ports (Interfaces)

**File**: `internal/ports/artifact_store.go`

```go
package ports

import (
    "context"
    "jax-trading-assistant/internal/domain/artifacts"
)

// ArtifactStore manages artifact persistence
type ArtifactStore interface {
    // Save persists an artifact to database
    Save(ctx context.Context, artifact *artifacts.Artifact) error
    
    // GetByID retrieves artifact by ID
    GetByID(ctx context.Context, id string) (*artifacts.Artifact, error)
    
    // GetLatestApproved retrieves the latest APPROVED artifact for a strategy
    GetLatestApproved(ctx context.Context, strategyName string) (*artifacts.Artifact, error)
    
    // ListByState retrieves all artifacts in a given state
    ListByState(ctx context.Context, state artifacts.State) ([]*artifacts.Artifact, error)
    
    // Promote transitions an artifact to a new state
    Promote(ctx context.Context, artifactID string, toState artifacts.State, promotedBy string, reason string) error
}

// ObjectStore manages binary artifact blobs
type ObjectStore interface {
    // Put saves artifact blob and returns URI
    Put(ctx context.Context, key string, data []byte) (string, error)
    
    // Get retrieves artifact blob by URI
    Get(ctx context.Context, uri string) ([]byte, error)
    
    // Delete removes artifact blob
    Delete(ctx context.Context, uri string) error
}
```

---

### Change 7: Implement Postgres Artifact Store

**File**: `internal/integrations/postgres/artifact_store.go`

```go
package postgres

import (
    "context"
    "fmt"
    "time"
    
    "jax-trading-assistant/internal/domain/artifacts"
    "jax-trading-assistant/internal/ports"
    
    "github.com/jackc/pgx/v5/pgxpool"
)

type ArtifactStore struct {
    db *pgxpool.Pool
}

func NewArtifactStore(db *pgxpool.Pool) ports.ArtifactStore {
    return &ArtifactStore{db: db}
}

func (s *ArtifactStore) Save(ctx context.Context, artifact *artifacts.Artifact) error {
    query := `
        INSERT INTO strategy_artifacts (
            artifact_id, schema_version, strategy_name, strategy_version, 
            code_ref, params, data_window, validation, risk_profile,
            created_by, created_at, payload_hash, payload_uri, signature, state
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
    `
    
    _, err := s.db.Exec(ctx, query,
        artifact.ID,
        artifact.SchemaVersion,
        artifact.Strategy.Name,
        artifact.Strategy.Version,
        artifact.Strategy.CodeRef,
        artifact.Strategy.Params,
        artifact.DataWindow,
        artifact.Validation,
        artifact.RiskProfile,
        artifact.CreatedBy,
        artifact.CreatedAt,
        artifact.PayloadHash,
        artifact.PayloadURI,
        artifact.Signature,
        artifact.State,
    )
    
    return err
}

func (s *ArtifactStore) GetLatestApproved(ctx context.Context, strategyName string) (*artifacts.Artifact, error) {
    query := `
        SELECT artifact_id, schema_version, strategy_name, strategy_version,
               code_ref, params, data_window, validation, risk_profile,
               created_by, created_at, payload_hash, payload_uri, signature, state
        FROM strategy_artifacts
        WHERE strategy_name = $1 
          AND state = 'APPROVED'
        ORDER BY created_at DESC
        LIMIT 1
    `
    
    var artifact artifacts.Artifact
    err := s.db.QueryRow(ctx, query, strategyName).Scan(
        &artifact.ID,
        &artifact.SchemaVersion,
        &artifact.Strategy.Name,
        &artifact.Strategy.Version,
        &artifact.Strategy.CodeRef,
        &artifact.Strategy.Params,
        &artifact.DataWindow,
        &artifact.Validation,
        &artifact.RiskProfile,
        &artifact.CreatedBy,
        &artifact.CreatedAt,
        &artifact.PayloadHash,
        &artifact.PayloadURI,
        &artifact.Signature,
        &artifact.State,
    )
    
    if err != nil {
        return nil, fmt.Errorf("no approved artifact found for strategy %s: %w", strategyName, err)
    }
    
    return &artifact, nil
}

func (s *ArtifactStore) Promote(ctx context.Context, artifactID string, toState artifacts.State, promotedBy string, reason string) error {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Get current state
    var fromState artifacts.State
    err = tx.QueryRow(ctx, "SELECT state FROM strategy_artifacts WHERE artifact_id = $1", artifactID).Scan(&fromState)
    if err != nil {
        return fmt.Errorf("artifact not found: %w", err)
    }
    
    // Update state
    _, err = tx.Exec(ctx, "UPDATE strategy_artifacts SET state = $1 WHERE artifact_id = $2", toState, artifactID)
    if err != nil {
        return err
    }
    
    // Log promotion
    _, err = tx.Exec(ctx, `
        INSERT INTO artifact_promotions (artifact_id, from_state, to_state, promoted_by, promoted_at, reason)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, artifactID, fromState, toState, promotedBy, time.Now(), reason)
    if err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}
```

---

## Phase 2: Create Trader Entrypoint Skeleton (Week 5-6)

### Change 8: Create cmd/trader Entrypoint

**File**: `cmd/trader/main.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "jax-trading-assistant/internal/app/trader"
    "jax-trading-assistant/internal/integrations/postgres"
    "jax-trading-assistant/libs/database"
)

func main() {
    fmt.Println("🚀 Jax Trader Runtime starting...")
    
    // 1. Load configuration
    cfg, err := trader.LoadConfig()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }
    
    // 2. Initialize database
    db, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("failed to connect to database: %v", err)
    }
    defer db.Close()
    
    // 3. Initialize artifact store
    artifactStore := postgres.NewArtifactStore(db)
    
    // 4. Load approved artifacts
    fmt.Println("📦 Loading approved strategy artifacts...")
    rsiArtifact, err := artifactStore.GetLatestApproved(context.Background(), "rsi_momentum")
    if err != nil {
        log.Fatalf("failed to load RSI artifact: %v", err)
    }
    
    // Verify artifact hash
    if err := rsiArtifact.VerifyHash(); err != nil {
        log.Fatalf("SECURITY VIOLATION: artifact hash verification failed: %v", err)
    }
    
    fmt.Printf("✅ Loaded artifact: %s (hash: %s)\n", rsiArtifact.ID, rsiArtifact.PayloadHash[:16]+"...")
    
    // 5. Initialize runtime
    runtime, err := trader.NewRuntime(cfg, db, rsiArtifact)
    if err != nil {
        log.Fatalf("failed to initialize runtime: %v", err)
    }
    
    // 6. Start runtime
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        fmt.Println("\n🛑 Shutdown signal received, stopping trader...")
        cancel()
    }()
    
    // Run trader
    if err := runtime.Run(ctx); err != nil {
        log.Fatalf("runtime error: %v", err)
    }
    
    fmt.Println("✅ Trader runtime stopped gracefully")
}
```

---

### Change 9: Create Trader Runtime Composition

**File**: `internal/app/trader/runtime.go`

```go
package trader

import (
    "context"
    "fmt"
    "time"
    
    "jax-trading-assistant/internal/domain/artifacts"
    "jax-trading-assistant/libs/observability"
    
    "github.com/jackc/pgx/v5/pgxpool"
)

type Runtime struct {
    cfg       *Config
    db        *pgxpool.Pool
    artifact  *artifacts.Artifact
    logger    *observability.Logger
    
    // Modules (will be added in later phases)
    // marketDataSvc  ports.MarketDataService
    // riskEngine     ports.RiskEngine
    // executionEngine ports.ExecutionEngine
}

func NewRuntime(cfg *Config, db *pgxpool.Pool, artifact *artifacts.Artifact) (*Runtime, error) {
    logger := observability.NewLogger("trader")
    
    // Log artifact metadata for audit
    logger.Info("runtime initialized with artifact",
        "artifact_id", artifact.ID,
        "strategy", artifact.Strategy.Name,
        "version", artifact.Strategy.Version,
        "hash", artifact.PayloadHash,
        "code_ref", artifact.Strategy.CodeRef,
    )
    
    return &Runtime{
        cfg:      cfg,
        db:       db,
        artifact: artifact,
        logger:   logger,
    }, nil
}

func (r *Runtime) Run(ctx context.Context) error {
    r.logger.Info("trader runtime starting")
    
    // Health check ticker
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            r.logger.Info("trader runtime shutting down")
            return nil
            
        case <-ticker.C:
            r.logger.Debug("trader runtime health check",
                "artifact", r.artifact.ID,
                "uptime", time.Since(ctx.Done().(*context.cancelCtx).deadline),
            )
            
            // TODO: Execute trading logic
            // 1. Fetch market data
            // 2. Generate signals using artifact strategy
            // 3. Apply risk checks
            // 4. Execute trades
            // 5. Log to audit trail
        }
    }
}
```

---

### Change 10: Create Import Boundary CI Check

**File**: `.github/workflows/import-boundary-check.yml`

```yaml
name: Trader Import Boundary Check

on:
  pull_request:
    paths:
      - 'cmd/trader/**'
      - 'internal/**'
  push:
    branches: [main]

jobs:
  check-trader-imports:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Check Trader Dependencies
        run: |
          echo "🔍 Checking trader runtime dependencies..."
          
          # Get all dependencies of cmd/trader
          DEPS=$(go list -deps ./cmd/trader/... 2>&1)
          
          # Check for forbidden dependencies
          FORBIDDEN="agent0|dexter|research"
          VIOLATIONS=$(echo "$DEPS" | grep -E "$FORBIDDEN" || true)
          
          if [ -n "$VIOLATIONS" ]; then
            echo "❌ FORBIDDEN DEPENDENCIES DETECTED:"
            echo "$VIOLATIONS"
            echo ""
            echo "Trader runtime must NOT import:"
            echo "  - libs/agent0"
            echo "  - libs/dexter"
            echo "  - internal/app/research"
            echo "  - internal/integrations/agent0"
            echo "  - internal/integrations/dexter"
            exit 1
          fi
          
          echo "✅ No forbidden dependencies found"
      
      - name: Verify Artifact Verification
        run: |
          echo "🔐 Checking artifact verification in trader..."
          
          # Ensure trader code calls artifact.VerifyHash()
          if ! grep -r "VerifyHash" cmd/trader/; then
            echo "❌ Trader MUST verify artifact hashes before loading"
            exit 1
          fi
          
          echo "✅ Artifact verification present"
```

**PowerShell Equivalent** (for local checks):
```powershell
# scripts/check-trader-deps.ps1

Write-Host "🔍 Checking trader runtime dependencies..." -ForegroundColor Cyan

# Get dependencies
$deps = go list -deps ./cmd/trader/... 2>&1 | Out-String

# Check for forbidden imports
$forbidden = @("agent0", "dexter", "research")
$violations = @()

foreach ($pkg in $forbidden) {
    if ($deps -match $pkg) {
        $violations += $pkg
    }
}

if ($violations.Count -gt 0) {
    Write-Host "❌ FORBIDDEN DEPENDENCIES DETECTED:" -ForegroundColor Red
    Write-Host ($violations -join ", ")
    exit 1
}

Write-Host "✅ No forbidden dependencies found" -ForegroundColor Green
```

---

## Summary of First 10 Changes

| # | Change | Effort | Risk | Priority |
|---|--------|--------|------|----------|
| 1 | Golden test infrastructure | 2 days | Low | **High** |
| 2 | Replay harness | 2 days | Low | **High** |
| 3 | Deterministic clock interface | 1 day | Low | **High** |
| 4 | Artifact database migration | 1 day | Low | **Critical** |
| 5 | Artifact domain model | 2 days | Low | **Critical** |
| 6 | Artifact store ports | 1 day | Low | **Critical** |
| 7 | Postgres artifact store implementation | 2 days | Low | **Critical** |
| 8 | cmd/trader entrypoint skeleton | 2 days | Low | **High** |
| 9 | Trader runtime composition | 2 days | Low | **High** |
| 10 | Import boundary CI check | 1 day | Low | **High** |

**Total: ~16 days (3 weeks)**

---

## Next Steps After First 10 Changes

After completing these foundational changes:

1. **Phase 1 (Weeks 4-6)**: Collapse jax-signal-generator → jax-orchestrator HTTP hop
2. **Phase 2 (Weeks 7-10)**: Collapse jax-api → jax-orchestrator HTTP hop
3. **Phase 3 (Weeks 11-18)**: Migrate jax-trade-executor logic into cmd/trader
4. **Phase 4 (Weeks 19-24)**: Create cmd/research runtime + artifact builder
5. **Phase 5 (Weeks 25-27)**: Decommission old service boundaries

**Estimated Total Timeline**: 6-9 months for full migration
