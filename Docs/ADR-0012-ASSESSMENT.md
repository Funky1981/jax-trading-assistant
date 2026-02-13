# ADR-0012 Migration Assessment - Executive Summary

**Date**: February 13, 2026  
**Prepared by**: Senior Staff Engineer Review  
**Status**: Comprehensive codebase review completed

---

## TL;DR

**Viability**: ✅ **YES** - Migration to modular monolith with two runtimes is technically feasible  
**Timeline**: ⏱️ **6-9 months** - This is a ground-up re-architecture, not incremental refactoring  
**Risk Level**: ⚠️ **HIGH** - Rewriting working production system that handles real money  
**Current State**: ❌ **NOT STARTED** - ADR-0012 is a proposal only; zero migration code exists  

**Recommendation**: 
1. **Option A (Conservative)**: Focus on incremental improvements to current architecture (2-3 weeks, low risk)
2. **Option B (Pragmatic)**: Proof-of-concept for artifact system only (6-8 weeks, medium risk)
3. **Option C (Ambitious)**: Full ADR-0012 migration (6-9 months, high risk) ← Current proposal

---

## What You Have Today (Working)

### ✅ Production System Status
- **9 microservices** running locally via docker-compose
- **IB paper trading integration** functional (Phase 3 Complete)
- **Trade execution engine** operational (Phase 4 Complete)
- **AI orchestration** with Agent0/Memory/Dexter (Phase 4 Orchestration Complete)
- **Signal generation** with auto-trigger for high-confidence signals
- **Security hardening** (JWT, CORS, rate limiting)

### Current Service Architecture

```
Frontend (React :5173)
    ↓
jax-api (:8081) ─────────┬──→ jax-orchestrator (:8091)
    ├──→ jax-trade-executor (:8097)
    ├──→ ib-bridge (:8092)
    └──→ postgres (:5432)
    
jax-signal-generator (:8096) ──→ jax-orchestrator
    
jax-orchestrator ──┬──→ jax-memory (:8090)
                   ├──→ agent0-service (:8093)
                   └──→ dexter (:8094) [optional]
                   
jax-memory ──→ hindsight (:8888)

jax-trade-executor ──→ ib-bridge ──→ IB Gateway

jax-market (:8095) ──┬──→ ib-bridge
                     ├──→ Polygon API
                     └──→ Alpaca API
```

**Key Characteristic**: Distributed system with HTTP calls between services on localhost.

---

## What ADR-0012 Proposes (Not Implemented)

### Target Architecture

```
cmd/trader (Production Runtime)
    ├── Artifact Loader (SHA-256 verify)
    ├── Market Data Module
    ├── Strategy Engine (deterministic)
    ├── Risk Engine
    ├── Execution Engine
    └── Audit Logger
    
cmd/research (Experimental Runtime)
    ├── Backtest Engine
    ├── Dexter/Agent0 Integration
    ├── Hindsight Analysis
    └── Artifact Builder
    
Artifact Promotion Gate
    ├── Postgres Metadata Tables
    ├── S3/MinIO Blob Storage
    └── Approval Workflow (DRAFT→APPROVED)
    
External Boundaries (Keep Separate)
    ├── ib-bridge (Python FastAPI)
    ├── hindsight (Python vector memory)
    ├── postgres
    └── observability stack
```

**Key Characteristic**: Modular monolith with two deployables, in-process module calls.

---

## Viability Assessment

### ✅ What Makes Migration Viable

1. **Clean library separation already exists**
   - [libs/strategies](libs/strategies/), [libs/trading/executor](libs/trading/executor/), [libs/marketdata](libs/marketdata/) are well-factored
   - Go workspace structure supports multi-module composition
   - UTCP abstraction allows transport swapping (HTTP → local)

2. **Database is already centralized**
   - Single Postgres instance (all services share)
   - Schema is well-designed with proper migrations

3. **External dependencies are vendored**
   - Hindsight, Dexter, Agent0 are copies (not submodules)
   - Easy to wrap as internal integrations

4. **Risk controls already implemented**
   - Position sizing, risk constraints, audit logging exist
   - Can be reused in new architecture

### ⚠️ What Makes Migration Risky

1. **Zero testing infrastructure for baseline validation**
   - ❌ No golden tests for signal/execution decisions
   - ❌ No replay harness for deterministic behavior
   - ❌ No fixture infrastructure
   - **Impact**: Cannot verify behavior preservation during migration

2. **Working production system handling real money**
   - Current system executes live paper trades via IB Gateway
   - Any regression could cause financial loss
   - High bar for confidence before cutover

3. **Architectural philosophy mismatch**
   - Current: Distributed microservices (HTTP boundaries)
   - Proposed: Modular monolith (in-process boundaries)
   - **Not** incremental - this is a fundamental re-architecture

4. **Unclear business value**
   - Current system is functional and meeting requirements
   - Migration benefits (reduced latency, simpler deployment) not quantified
   - Opportunity cost: 6-9 months of feature development foregone

---

## What Must Remain Separate

Based on actual codebase analysis, these **MUST** stay as separate processes:

| Service | Reason | Keep/Consolidate |
|---------|--------|------------------|
| **ib-bridge** | Python runtime, IB TWS protocol, session management isolation | **KEEP FOREVER** |
| **hindsight** | Python vector memory service, different release cadence | **KEEP (short-term)** |
| **Postgres** | Infrastructure, not app logic | **KEEP FOREVER** |
| **Prometheus/Grafana** | Observability stack | **KEEP FOREVER** |
| **Frontend** | React/Vite client | **KEEP FOREVER** |
| jax-orchestrator | Can collapse into cmd/research | **CONSOLIDATE** |
| jax-signal-generator | Can collapse into cmd/research | **CONSOLIDATE** |
| jax-trade-executor | Can collapse into cmd/trader | **CONSOLIDATE** |
| jax-memory | UTCP facade (evaluate later) | **EVALUATE** |
| jax-market | Data ingestion (evaluate later) | **EVALUATE** |

---

## Dependency Conflicts Analysis

### Can Trader Runtime Be "Clean"?

**✅ YES** - with proper package boundaries:

**Research-Only Dependencies (Trader CANNOT import)**:
- `libs/agent0` - AI planning client
- `libs/dexter` - Company research client
- `internal/integrations/hindsight` - Vector memory client
- `internal/app/research` - Research runtime composition

**Trader-Safe Dependencies (Trader CAN import)**:
- `libs/strategies` - Strategy implementations (with artifact gate)
- `libs/trading/executor` - Trade execution logic
- `libs/marketdata` - Multi-provider market data client
- `libs/database` - Database connection management
- `libs/resilience` - Circuit breakers, retries
- `internal/integrations/ib` - IB Bridge client
- `internal/integrations/postgres` - Database adapters

**Enforcement Mechanism**:
```powershell
# CI check (added to GitHub Actions)
$deps = go list -deps ./cmd/trader/...
if ($deps -match "agent0|dexter|research") {
    Write-Error "Trader imports forbidden packages"
    exit 1
}
```

**Verdict**: Dependency separation is **achievable** and **enforceable**.

---

## Migration Complexity Analysis

### 🟢 Easy Wins (1-2 weeks each)

1. **jax-signal-generator → jax-orchestrator HTTP hop**
   - Current: [services/jax-signal-generator/internal/orchestrator/client.go](services/jax-signal-generator/internal/orchestrator/client.go#L18)
   - Action: Replace HTTP POST with in-process module call
   - Risk: **Low** - Both are Go services, shared database

2. **UTCP transport swapping**
   - Current: [config/providers.json](config/providers.json) has HTTP/local mix
   - Action: Change `"transport": "http"` → `"transport": "local"`
   - Risk: **Very low** - configuration change only

### 🟡 Medium Complexity (4-6 weeks)

3. **Artifact database schema + loader**
   - Action: Create 4 tables, implement loader with hash verification
   - Risk: **Low** - isolated feature, no existing dependencies

4. **cmd/trader skeleton**
   - Action: Create entrypoint, load artifacts, log audit trail
   - Risk: **Low** - new code, doesn't touch existing execution

### 🔴 High Complexity (6-8 weeks each)

5. **Migrate jax-trade-executor logic to cmd/trader**
   - Current: [services/jax-trade-executor](services/jax-trade-executor/) (working)
   - Action: Move to `internal/modules/execution`, wire into trader
   - Risk: **CRITICAL** - money at stake, must be deterministic

6. **Create cmd/research runtime**
   - Action: Backtest engine, artifact builder, Dexter/Agent0 integration
   - Risk: **Medium** - new feature, no production risk

7. **Decommission old services**
   - Action: Remove HTTP boundaries, update docker-compose
   - Risk: **CRITICAL** - production deployment change

---

## Proposed Target Module Layout

```
jax-trading-assistant/
├── cmd/
│   ├── trader/              # NEW: Production trader entrypoint
│   └── research/            # NEW: Research runtime entrypoint
│
├── internal/                # NEW: Shared runtime composition
│   ├── app/
│   │   ├── trader/          # Trader wiring
│   │   └── research/        # Research wiring
│   ├── domain/              # Business logic (no infra)
│   │   ├── strategy/
│   │   ├── risk/
│   │   ├── execution/
│   │   ├── artifacts/       # NEW: Artifact domain model
│   │   └── audit/
│   ├── ports/               # Interface definitions
│   ├── modules/             # Application services
│   │   ├── orchestration/   # Consolidate orchestrators
│   │   ├── execution/
│   │   └── artifacts/
│   ├── integrations/        # Adapter implementations
│   │   ├── ib/
│   │   ├── postgres/
│   │   ├── hindsight/       # Research-only
│   │   ├── dexter/          # Research-only
│   │   └── agent0/          # Research-only
│   └── policy/
│       └── importcheck/     # CI boundary enforcement
│
├── libs/                    # KEEP: Shared libraries
│   ├── strategies/
│   ├── trading/executor/
│   ├── marketdata/
│   ├── utcp/
│   └── ...
│
├── services/                # DEPRECATE gradually
│   ├── ib-bridge/           # KEEP (external boundary)
│   ├── hindsight/           # KEEP (Python runtime)
│   └── ...                  # Remove over time
│
└── db/
    └── migrations/
        └── 002_artifacts.sql  # NEW
```

---

## Artifact Promotion Design

### Contract (JSON Schema)

```json
{
  "artifact_id": "strat_rsi_momentum_2026-02-13T12:34:56Z",
  "schema_version": "1.0.0",
  "strategy": {
    "name": "rsi_momentum",
    "version": "2.1.3",
    "code_ref": "git:abc123...",
    "params": {"rsi_period": 14}
  },
  "validation": {
    "backtest_run_id": "uuid",
    "metrics": {"sharpe": 1.42, "max_drawdown": 0.12},
    "determinism_seed": 42,
    "report_uri": "s3://bucket/report.html"
  },
  "risk_profile": {
    "max_position_pct": 0.20,
    "max_daily_loss": 1000,
    "allowed_order_types": ["LMT"]
  },
  "hash": "sha256:...",
  "state": "APPROVED"
}
```

### State Machine

```
DRAFT → VALIDATED → REVIEWED → APPROVED → ACTIVE
                                    ↓
                                DEPRECATED
                                    ↓
                                REVOKED
```

**Rule**: Trader loads ONLY artifacts in `APPROVED` state (NOT revoked).

### Storage

**Postgres Tables** (metadata):
- `strategy_artifacts` - Main artifact registry
- `artifact_approvals` - Approver identity + timestamp
- `artifact_promotions` - State transition audit log
- `artifact_validation_reports` - Backtest results

**S3/MinIO** (binary blobs):
- Immutable artifact JSON payloads
- Validation reports (HTML/PDF)

### Trader Loading Path

```go
// 1. Query latest APPROVED artifact
artifact := store.GetLatestApproved("rsi_momentum")

// 2. Fetch blob from object store
blob := objectStore.Get(artifact.PayloadURI)

// 3. Verify SHA-256 hash
hash := sha256.Sum256(blob)
if hash != artifact.PayloadHash {
    LOG SECURITY VIOLATION
    FAIL FAST
}

// 4. Deserialize and use
strategy := json.Unmarshal(blob)

// 5. Log to audit trail
auditLogger.Log("artifact_loaded", artifact.ID, artifact.PayloadHash)
```

---

## Migration Plan (6 Phases)

| Phase | Goal | Duration | Risk | Exit Criteria |
|-------|------|----------|------|---------------|
| **0** | Baseline + contracts | 2-3 weeks | Low | Golden tests passing, replay harness working |
| **1** | Collapse signal→orchestrator hop | 2-3 weeks | Low | DB side effects match baseline |
| **2** | Collapse API→orchestrator hop | 3-4 weeks | Medium | API contract parity, latency non-regressive |
| **3** | Trader runtime skeleton | 4-6 weeks | Low | Artifact loading works, hash verified |
| **4** | Migrate trade execution to trader | 6-8 weeks | **CRITICAL** | 1 month parallel run, zero discrepancies |
| **5** | Research runtime + artifact builder | 4-6 weeks | Medium | Backtest determinism, artifacts generated |
| **6** | Decommission old services | 2-3 weeks | **CRITICAL** | Smoke tests pass, runbooks updated |

**Total: 23-33 weeks (6-9 months)**

---

## Critical Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Behavior drift during HTTP removal | **HIGH** | CRITICAL | Phase 0 golden tests (MANDATORY) |
| Determinism regressions in trader | MEDIUM | CRITICAL | Fixed-seed tests, ban time.Now() |
| Production outage during cutover | MEDIUM | CRITICAL | 1-month parallel run, rollback plan |
| Artifact hash collision | LOW | HIGH | SHA-256 (cryptographically secure) |
| Unapproved artifact loaded | LOW | CRITICAL | CI import checks + runtime verification |
| Configuration drift | MEDIUM | MEDIUM | Single config package |
| Performance regression | MEDIUM | MEDIUM | Before/after benchmarks |

---

## Testing Strategy

### Phase 0 (MANDATORY Before Any Migration)

**Golden Tests**:
```powershell
# Capture current signal generation behavior
Invoke-RestMethod -Uri "http://localhost:8096/generate" | 
    Out-File "tests/golden/signals/baseline.json"

# Verify future changes match baseline
Compare-Object (Get-Content baseline.json) (Get-Content new-output.json)
```

**Replay Tests**:
```go
// Load historical fixture
fixture := LoadFixture("market_2026-01-15.json")

// Replay 10 times with same inputs
for i := 0; i < 10; i++ {
    signals := strategy.Generate(fixture.Market, fixture.Params)
    assert.Equal(t, fixture.ExpectedSignals, signals)
}
```

**Determinism Tests**:
- Fixed clock (ban `time.Now()` in decision path)
- Fixed RNG seed
- Stable map iteration order

---

## Recommendations

### Option A: **Incremental Enhancement** (RECOMMENDED)

**Timeline**: 2-3 weeks  
**Risk**: Low  
**Cost**: Minimal  

**What to Do**:
1. ✅ Centralize HTTP client creation (1-2 days)
2. ✅ Add golden test fixtures (2-3 days)
3. ✅ Consolidate configuration (1 day)
4. ✅ Document current architecture (1 day)
5. ✅ Add comprehensive monitoring (1 week)

**What NOT to Do**:
- ❌ Don't re-architect working system
- ❌ Don't create cmd/trader or cmd/research
- ❌ Don't migrate services to monolith

**Outcome**: Improved current system without disruption.

---

### Option B: **Proof-of-Concept Artifact System** (PRAGMATIC)

**Timeline**: 6-8 weeks  
**Risk**: Medium  
**Cost**: Moderate  

**What to Do**:
1. ✅ Implement artifact database schema (Week 1)
2. ✅ Create artifact domain model + loader (Week 2)
3. ✅ Build cmd/trader skeleton (Week 3-4)
4. ✅ Run parallel with old executor for 1 month (Week 5-8)
5. ✅ **Decision point**: Continue or abandon

**What NOT to Do**:
- ❌ Don't decommission existing services yet
- ❌ Don't migrate all business logic yet

**Outcome**: Validate artifact promotion concept with minimal risk.

---

### Option C: **Full ADR-0012 Migration** (NOT RECOMMENDED WITHOUT POC)

**Timeline**: 6-9 months  
**Risk**: **EXTREME**  
**Cost**: Very high  

**Why NOT Recommended**:
- Current system is working
- No proven business case for migration
- High risk to production system
- Opportunity cost (foregone feature development)
- Testing infrastructure must be built first

**IF You Proceed**:
1. ✅ **MANDATORY**: Complete Phase 0 (golden tests, replay harness)
2. ✅ **MANDATORY**: Business case with ROI analysis
3. ✅ Executive sponsorship + dedicated team
4. ✅ 6-9 month commitment with no other priorities

---

## Concrete First Steps (If Proceeding with Option B or C)

See [ADR-0012-FIRST-CODE-CHANGES.md](ADR-0012-FIRST-CODE-CHANGES.md) for detailed implementation guide.

**Week 1**:
1. Create golden test infrastructure (Change #1)
2. Create replay harness (Change #2)
3. Create deterministic clock interface (Change #3)

**Week 2**:
4. Create artifact database migration (Change #4)
5. Create artifact domain model (Change #5)
6. Create artifact store ports (Change #6)

**Week 3**:
7. Implement Postgres artifact store (Change #7)
8. Create cmd/trader entrypoint skeleton (Change #8)
9. Create trader runtime composition (Change #9)
10. Create import boundary CI check (Change #10)

**Exit Criteria**: 
- ✅ Golden tests pass for 1 week straight
- ✅ Replay tests verify determinism
- ✅ cmd/trader starts, loads artifact, verifies hash
- ✅ CI enforces import boundaries

---

## Questions for Stakeholders

Before proceeding, answer these:

1. **Business Case**:
   - What problem does the current architecture have that justifies 6-9 months of re-architecture?
   - What is the ROI (reduced costs, faster development, improved reliability)?

2. **Risk Tolerance**:
   - Can we afford production incidents during migration?
   - Is there budget for 1-month parallel runs and extensive testing?

3. **Team Capacity**:
   - Do we have dedicated engineering resources for 6-9 months?
   - Can we pause feature development during migration?

4. **Success Criteria**:
   - How will we measure success?
   - What would cause us to rollback or abandon?

---

## Conclusion

**ADR-0012 is technically viable but operationally risky.** The current system is working well. Before committing to a 6-9 month re-architecture:

1. Start with **Option A** (incremental improvements)
2. Build comprehensive testing infrastructure
3. If business case emerges, try **Option B** (POC artifact system)
4. Only proceed to **Option C** (full migration) with proven POC and executive sponsorship

**Most important**: Don't let perfection be the enemy of good. The current microservices architecture, while not ideal, is **functional, documented, and production-ready**.
