# ADR-0012 Phase 4: Artifact-based Promotion Gate - COMPLETE ✅

**Date:** February 13, 2026  
**Status:** Implementation Complete  
**ADR:** ADR-0012 Two-Runtime Modular Monolith

---

## 🎯 Summary

Phase 4 of the ADR-0012 modular-monolith migration implements the **Artifact-based Promotion Gate** — an immutable, auditable system that gates all strategy execution behind cryptographically verified, human-approved artifacts. No strategy can generate signals or trigger trades unless it has been promoted through a defined approval workflow and its SHA-256 hash passes verification at load time.

---

## ✅ Completed Tasks

### 1. Database Schema (`db/postgres/006_strategy_artifacts.sql`)
- ✅ `artifact_approval_state` enum with 7 states: DRAFT → VALIDATED → REVIEWED → APPROVED → ACTIVE → DEPRECATED → REVOKED
- ✅ `strategy_artifacts` table with JSONB payload, SHA-256 hash, immutability constraints
- ✅ `artifact_approvals` table with full workflow state tracking
- ✅ `artifact_promotions` table as audit log for every state transition
- ✅ `artifact_validation_reports` table for test/backtest results
- ✅ ALTER: `strategy_signals` gains `artifact_id` FK
- ✅ ALTER: `trades` gains `artifact_id` and `artifact_hash` columns
- ✅ Views: `approved_artifacts`, `latest_artifacts`, `artifact_history`
- ✅ Trigger: `update_artifact_approvals_updated_at`
- ✅ Migration applied and verified in PostgreSQL

### 2. Domain Models (`internal/domain/artifacts/artifact.go`)
- ✅ `Artifact` struct with canonical JSON serialization (sorted keys)
- ✅ `CanonicalPayload()` — deterministic JSON for hashing
- ✅ `ComputeHash()` — SHA-256 of canonical payload
- ✅ `VerifyHash()` — recompute and compare
- ✅ `ApprovalState` enum with `ValidTransitions` map
- ✅ `IsApproved()`, `IsUsable()`, `CanTransitionTo()` state predicates
- ✅ Supporting types: `StrategyInfo`, `DataWindow`, `ValidationInfo`, `RiskProfile`, `Approval`, `Promotion`, `ValidationReport`
- ✅ Factory functions: `NewArtifact()`, `NewApproval()`

### 3. Database Store (`internal/domain/artifacts/store.go`)
- ✅ `CreateArtifact` — INSERT with hash verification
- ✅ `GetArtifactByID`, `GetArtifactByHash` — exact retrieval
- ✅ `ListApprovedArtifacts` — returns only APPROVED/ACTIVE artifacts
- ✅ `GetLatestApprovedArtifact` — latest approved for a given strategy name
- ✅ `CreateApproval`, `GetApproval` — approval lifecycle
- ✅ `UpdateApprovalState` — transactional state transition with audit log (writes to `artifact_promotions`)
- ✅ `CreateValidationReport` — stores test/backtest results

### 4. Artifact Loader (`internal/modules/artifacts/loader.go`)
- ✅ `LoadApprovedStrategies()` — loads only APPROVED/ACTIVE artifacts from DB
- ✅ Hash verification at load time (tampered artifacts rejected)
- ✅ Registers concrete strategies with artifact tracking metadata (`Extra` map)
- ✅ `RefreshStrategies()` — hot-reload via SIGHUP or admin endpoint
- ✅ Error propagation from `Registry.Register()` (bug fixed during testing)

### 5. Audit Trail Integration
- ✅ `libs/contracts/domain/signal.go` — added `ArtifactID` field
- ✅ `libs/strategies/strategy.go` — added `Extra` map to `StrategyMetadata`
- ✅ Signal generation (`internal/trader/signalgenerator/inprocess.go`) — populates `artifact_id` from registry metadata, stores in `strategy_signals` table
- ✅ Execution engine (`internal/modules/execution/engine.go`) — `Signal` and `TradeResult` carry `ArtifactID` + `ArtifactHash`
- ✅ Execution service (`internal/modules/execution/service.go`) — `GetSignal` JOINs `strategy_artifacts` for hash, `StoreTrade` writes `artifact_id` and `artifact_hash`

### 6. Promotion Workflow API (`cmd/trader/handlers_artifacts.go`)
- ✅ `GET /api/v1/artifacts` — list all artifacts
- ✅ `POST /api/v1/artifacts` — create new artifact (starts as DRAFT)
- ✅ `GET /api/v1/artifacts/{id}` — get artifact by ID with hash verification
- ✅ `POST /api/v1/artifacts/{id}/promote` — advance approval state (enforces valid transitions)
- ✅ `POST /api/v1/artifacts/{id}/validate` — run placeholder validation, auto-promote to VALIDATED
- ✅ Registered on `http.ServeMux` in `cmd/trader/main.go`

### 7. cmd/trader Integration (`cmd/trader/main.go`)
- ✅ Artifact store instantiation from connection pool
- ✅ Artifact loader replaces direct strategy registration
- ✅ Artifact HTTP handlers registered on API mux
- ✅ Binary compiles cleanly

---

## 🧪 Test Results

### Domain Tests (`internal/domain/artifacts/`) — 8 tests, ALL PASS
| Test | Status |
|------|--------|
| TestNewArtifact | ✅ |
| TestCanonicalPayloadDeterministic | ✅ |
| TestComputeHash | ✅ |
| TestVerifyHash | ✅ |
| TestApprovalStateMachine | ✅ |
| TestIsApproved (7 subtests) | ✅ |
| TestIsUsable (7 subtests) | ✅ |
| TestHashConsistency | ✅ |
| TestNewApproval | ✅ |

### Loader Tests (`internal/modules/artifacts/`) — 10 tests, ALL PASS
| Test | Status |
|------|--------|
| TestRegisterStrategyFromArtifact_RSI | ✅ |
| TestRegisterStrategyFromArtifact_MACD | ✅ |
| TestRegisterStrategyFromArtifact_MACrossover | ✅ |
| TestRegisterStrategyFromArtifact_UnknownStrategy | ✅ |
| TestRegisterStrategyFromArtifact_DuplicateRegistration | ✅ |
| TestLoadArtifact_HashVerification | ✅ |
| TestLoadArtifact_TamperedHashRejected | ✅ |
| TestAllThreeStrategiesRegister | ✅ |
| TestArtifactMetadataTracking | ✅ |
| TestArtifactIntegrity | ✅ |

### Regression — No regressions in existing modules
- `internal/modules/execution` — 4 tests PASS
- `internal/modules/orchestration` — 6 tests PASS
- `libs/strategies` — 15 tests PASS
- `libs/contracts` — 7 golden tests PASS
- `libs/contracts/domain` — 11 tests PASS
- `libs/contracts/adapters` — 7 tests PASS

### Pre-existing Issue (not Phase 4)
- `libs/contracts/converters/TestSignalToDomain` — case mismatch ("BUY" vs "buy"), pre-dates Phase 4

---

## 📁 Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `db/postgres/006_strategy_artifacts.sql` | 246 | Migration: tables, enum, views, triggers |
| `internal/domain/artifacts/artifact.go` | 296 | Domain models, hashing, state machine |
| `internal/domain/artifacts/store.go` | 428 | PostgreSQL CRUD with pgxpool |
| `internal/domain/artifacts/artifact_test.go` | 365 | Domain unit tests |
| `internal/modules/artifacts/loader.go` | 170 | Artifact-gated strategy loader |
| `internal/modules/artifacts/loader_test.go` | 260 | Loader unit tests |
| `cmd/trader/handlers_artifacts.go` | 460 | HTTP promotion API |

## 📝 Files Modified

| File | Change |
|------|--------|
| `cmd/trader/main.go` | Artifact store/loader/handlers, removed direct strategy registration |
| `libs/contracts/domain/signal.go` | Added `ArtifactID` field |
| `libs/strategies/strategy.go` | Added `Extra` map to `StrategyMetadata` |
| `internal/trader/signalgenerator/inprocess.go` | Populates `artifact_id` in signals |
| `internal/modules/execution/engine.go` | `ArtifactID`/`ArtifactHash` on Signal + TradeResult |
| `internal/modules/execution/service.go` | JOIN for hash, artifact columns in trades INSERT |

---

## 🔒 Security Properties

1. **Immutability** — Artifacts are write-once; any payload change invalidates the SHA-256 hash
2. **Verification at load** — Hash recomputed and compared before any strategy is registered
3. **Tamper detection** — Modified artifacts are rejected with explicit error
4. **Audit trail** — Every state transition is recorded in `artifact_promotions` with actor, reason, and timestamp
5. **Separation of concerns** — Artifact approval is independent from strategy code
6. **Full traceability** — Every signal and trade links back to its originating artifact ID and hash

---

## 🔄 State Machine

```
DRAFT → VALIDATED → REVIEWED → APPROVED → ACTIVE → DEPRECATED
  ↓         ↓          ↓          ↓          ↓          ↓
  REVOKED   REVOKED    REVOKED    REVOKED    REVOKED    REVOKED
            ↓
            DRAFT (re-draft)
```

Only APPROVED and ACTIVE artifacts are loaded by the trader.

---

## ⏭️ Future Work

- **Phase 5**: Integrate real backtesting into the `/validate` endpoint
- **Frontend**: Artifact management UI (create, review, promote)
- **CI/CD**: Automated validation pipeline on artifact creation
- **Signature verification**: Optional GPG/ECDSA signature on artifacts
- **Parameterized strategies**: Use artifact `params` to configure strategy instances
