# Macro Reaction Engine Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use Markdown checkbox syntax for tracking.

**Goal:** Build the first macro-reaction-engine slice: structured macro event storage, validation, dedupe, and ETF allowlist mapping without creating candidates, approvals, orders, or broker writes.

**Architecture:** Add a dedicated `internal/modules/macroevents` package for macro event normalization and persistence. Store accepted macro events in new `macro_events` and `macro_event_etf_map` tables while preserving the existing World Monitor inbox boundary as research-only input.

**Tech Stack:** Go, pgx/pgxpool, PostgreSQL migrations, existing ETF catalog in `internal/modules/instruments`, existing migration tests under `db/postgres/migrations`.

---

## First Folder

Work first on:

```text
Docs/plans/jax_complete_trading_readiness_docs/macro-reaction-engine/
```

This is the correct first folder because `README_COMPLETE_PACK.md` defines the build order as:

```text
1. World Monitor awareness
2. macro-reaction-engine
3. analysis-intelligence-layer
4. robust-profitability-layer
```

World Monitor awareness is already completed, so the next dependency is the macro reaction engine. Within that folder, start with:

```text
01_MACRO_EVENT_MODEL_AND_CALENDAR_DATA.md
```

Do not start with chart reactions, candidate generation, UI, or backtesting until macro events can be stored and rejected safely.

## Files

- Create: `db/postgres/migrations/000031_macro_events.up.sql`
- Create: `db/postgres/migrations/000031_macro_events.down.sql`
- Create: `db/postgres/migrations/macro_events_schema_test.go`
- Create: `internal/modules/macroevents/model.go`
- Create: `internal/modules/macroevents/validation.go`
- Create: `internal/modules/macroevents/mapping.go`
- Create: `internal/modules/macroevents/store.go`
- Create: `internal/modules/macroevents/service.go`
- Create: `internal/modules/macroevents/validation_test.go`
- Create: `internal/modules/macroevents/mapping_test.go`
- Create: `internal/modules/macroevents/service_test.go`
- Modify later, not in Phase 1: `cmd/trader/codex_api.go`
- Modify later, not in Phase 1: `cmd/trader/world_monitor_research_inbox.go`

## Safety Boundaries

- Do not write to `candidate_trades`.
- Do not write to `candidate_events`.
- Do not call `internal/modules/execution`.
- Do not add a broker/order API endpoint.
- Do not change runtime mode.
- Low-confidence or unsupported macro inputs should be rejected or stored as blocked/quarantined, not turned into trades.

---

### Task 1: Add Macro Event Schema Test

**Files:**
- Create: `db/postgres/migrations/macro_events_schema_test.go`

- [x] **Step 1: Write the failing migration test**

```go
package migrations

import (
	"path/filepath"
	"testing"
)

func TestMacroEventsMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000031_macro_events.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000031_macro_events.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS macro_events",
		"source_event_id TEXT NOT NULL",
		"event_type TEXT NOT NULL",
		"event_time_utc TIMESTAMPTZ NOT NULL",
		"surprise_value NUMERIC",
		"surprise_percent NUMERIC",
		"direction TEXT NOT NULL",
		"raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb",
		"UNIQUE(source, source_event_id)",
		"chk_macro_events_direction",
		"chk_macro_events_confidence",
		"CREATE TABLE IF NOT EXISTS macro_event_etf_map",
		"macro_event_id UUID NOT NULL REFERENCES macro_events(id)",
		"UNIQUE(macro_event_id, symbol)",
		"chk_macro_event_etf_map_confidence",
		"idx_macro_events_type_time",
		"idx_macro_events_source_event",
		"idx_macro_event_etf_map_symbol",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_macro_event_etf_map_symbol",
		"DROP TABLE IF EXISTS macro_event_etf_map",
		"DROP TABLE IF EXISTS macro_events",
	} {
		requireContains(t, down, fragment)
	}
}
```

- [x] **Step 2: Run the failing test**

Run:

```powershell
go test ./db/postgres/migrations -run TestMacroEventsMigrationDefinesRequiredTablesConstraintsAndIndexes -count=1
```

Expected: fail because `000031_macro_events.up.sql` does not exist yet.

---

### Task 2: Add Macro Event Migrations

**Files:**
- Create: `db/postgres/migrations/000031_macro_events.up.sql`
- Create: `db/postgres/migrations/000031_macro_events.down.sql`

- [x] **Step 1: Add the up migration**

```sql
CREATE TABLE IF NOT EXISTS macro_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    region TEXT NOT NULL,
    event_time_utc TIMESTAMPTZ NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT,
    actual_value NUMERIC,
    expected_value NUMERIC,
    previous_value NUMERIC,
    unit TEXT,
    surprise_value NUMERIC,
    surprise_percent NUMERIC,
    direction TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    normalized_event_id UUID REFERENCES event_normalized(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'accepted',
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source, source_event_id),
    CONSTRAINT chk_macro_events_direction CHECK (
        direction IN (
            'hawkish_rates',
            'dovish_rates',
            'risk_on',
            'risk_off',
            'inflation_hot',
            'inflation_cool',
            'growth_strong',
            'growth_weak',
            'unclear'
        )
    ),
    CONSTRAINT chk_macro_events_confidence CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT chk_macro_events_status CHECK (
        status IN ('accepted', 'rejected', 'quarantined')
    )
);

CREATE TABLE IF NOT EXISTS macro_event_etf_map (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    macro_event_id UUID NOT NULL REFERENCES macro_events(id) ON DELETE CASCADE,
    symbol TEXT NOT NULL,
    theme TEXT NOT NULL,
    mapping_reason TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(macro_event_id, symbol),
    CONSTRAINT chk_macro_event_etf_map_confidence CHECK (confidence >= 0 AND confidence <= 1)
);

CREATE INDEX IF NOT EXISTS idx_macro_events_type_time
    ON macro_events(event_type, event_time_utc DESC);
CREATE INDEX IF NOT EXISTS idx_macro_events_source_event
    ON macro_events(source, source_event_id);
CREATE INDEX IF NOT EXISTS idx_macro_events_status_time
    ON macro_events(status, event_time_utc DESC);
CREATE INDEX IF NOT EXISTS idx_macro_event_etf_map_symbol
    ON macro_event_etf_map(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_macro_event_etf_map_event
    ON macro_event_etf_map(macro_event_id);
```

- [x] **Step 2: Add the down migration**

```sql
DROP INDEX IF EXISTS idx_macro_event_etf_map_event;
DROP INDEX IF EXISTS idx_macro_event_etf_map_symbol;
DROP INDEX IF EXISTS idx_macro_events_status_time;
DROP INDEX IF EXISTS idx_macro_events_source_event;
DROP INDEX IF EXISTS idx_macro_events_type_time;
DROP TABLE IF EXISTS macro_event_etf_map;
DROP TABLE IF EXISTS macro_events;
```

- [x] **Step 3: Run the migration test**

Run:

```powershell
go test ./db/postgres/migrations -run TestMacroEventsMigrationDefinesRequiredTablesConstraintsAndIndexes -count=1
```

Expected: pass.

- [x] **Step 4: Commit**

```powershell
git add db/postgres/migrations/000031_macro_events.* db/postgres/migrations/macro_events_schema_test.go
git commit -m "feat: add macro event schema"
```

---

### Task 3: Add Domain Model and Validation

**Files:**
- Create: `internal/modules/macroevents/model.go`
- Create: `internal/modules/macroevents/validation.go`
- Create: `internal/modules/macroevents/validation_test.go`

- [x] **Step 1: Write validation tests**

Cover these cases in `validation_test.go`:

```text
valid NFP payload accepted
missing expected value rejected for NFP
unsupported event type rejected
stale event rejected
low confidence quarantined
payload containing broker/order/runtime override language rejected
```

Use a fixed `now` value:

```go
now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
```

- [x] **Step 2: Add model types**

Define these types in `model.go`:

```go
type EventType string
type Direction string
type Status string

type EventInput struct {
	Source        string
	SourceEventID string
	EventType     EventType
	Region        string
	EventTimeUTC  time.Time
	Headline      string
	Summary       string
	ActualValue   *decimalValue
	ExpectedValue *decimalValue
	PreviousValue *decimalValue
	Unit          string
	Direction     Direction
	Confidence    float64
	RawPayload    map[string]any
	AffectedETFs  []string
}

type ValidationResult struct {
	Valid  bool
	Status Status
	Reason string
}
```

Use a small internal numeric wrapper if the package does not already have a shared decimal type. Keep DB conversion inside the store.

- [x] **Step 3: Add deterministic validation**

Validation must enforce:

```text
source required
source_event_id required
supported event_type only
region required and phase 1 only accepts US
event_time_utc required
event must not be older than 24 hours
headline required
numeric releases require actual and expected
confidence must be > 0 and <= 1
confidence < 0.5 returns quarantined
affected_etfs required
trade/order/runtime override payload keys reject
```

- [x] **Step 4: Run validation tests**

Run:

```powershell
go test ./internal/modules/macroevents -run "TestValidate" -count=1
```

Expected: pass.

---

### Task 4: Add ETF Mapping Guard

**Files:**
- Create: `internal/modules/macroevents/mapping.go`
- Create: `internal/modules/macroevents/mapping_test.go`

- [x] **Step 1: Write mapping tests**

Cover:

```text
known ETF symbols are accepted
unknown symbols are rejected
inverse/leveraged/options/single-stock symbols are rejected through existing instrument catalog evaluation
empty ETF list is rejected
symbols normalize to uppercase and dedupe
```

- [x] **Step 2: Implement mapping guard**

Use `internal/modules/instruments.LoadDefaultCatalog()` and `catalog.Evaluate(symbol, "paper")`.

The function shape should be:

```go
func ValidateAndNormalizeETFs(symbols []string) ([]ETFMapping, error)
```

Each `ETFMapping` should include:

```text
symbol
theme
mapping_reason
confidence
```

For Phase 1, theme can be deterministic from symbol groups:

```text
QQQ -> growth/technology
SPY -> broad_market
IWM -> small_caps
TLT -> rates_duration
XLF -> financials
XLE -> energy
GLD -> safe_haven
```

- [x] **Step 3: Run mapping tests**

Run:

```powershell
go test ./internal/modules/macroevents -run "TestValidateAndNormalizeETFs" -count=1
```

Expected: pass.

---

### Task 5: Add Store and Service

**Files:**
- Create: `internal/modules/macroevents/store.go`
- Create: `internal/modules/macroevents/service.go`
- Create: `internal/modules/macroevents/service_test.go`

- [x] **Step 1: Write service tests with a fake store**

Cover:

```text
valid event persists once
duplicate source/source_event_id returns existing receipt
quarantined low-confidence event stores status quarantined and no ETF map
invalid unsupported event stores status rejected when configured to retain rejects
no candidate/order/trade APIs are called
```

- [x] **Step 2: Implement service**

Service shape:

```go
type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service
func (s *Service) Ingest(ctx context.Context, input EventInput) (Receipt, error)
```

Receipt shape:

```go
type Receipt struct {
	MacroEventID    string
	Status          Status
	RejectionReason string
	Duplicate       bool
	MappedETFs      []string
}
```

- [x] **Step 3: Implement Postgres store**

Store must:

```text
insert macro_events
insert macro_event_etf_map rows only for accepted events
return existing row on unique source/source_event_id
never write candidate_trades, candidate_events, approvals, executions, or orders
```

- [x] **Step 4: Run service tests**

Run:

```powershell
go test ./internal/modules/macroevents -count=1
```

Expected: pass.

- [x] **Step 5: Commit**

```powershell
git add internal/modules/macroevents
git commit -m "feat: add macro event ingestion service"
```

---

### Task 6: Phase 1 Verification

**Files:**
- No new files.

- [x] **Step 1: Run focused backend tests**

```powershell
go test ./db/postgres/migrations ./internal/modules/macroevents -count=1
```

Expected: pass.

- [x] **Step 2: Run broader backend tests**

```powershell
go test ./cmd/trader ./internal/modules/... ./libs/... -count=1
```

Expected: pass.

- [x] **Step 3: Check no forbidden writes were added**

```powershell
rg -n "candidate_trades|candidate_events|orders|execution|broker" internal/modules/macroevents cmd/trader
```

Original Phase 1 expectation: no matches in `internal/modules/macroevents`; existing matches elsewhere are allowed if pre-existing.

2026-06-17 closure result: later phases intentionally added paper-only macro candidate persistence in `internal/modules/macroevents/store.go`, and validation tests intentionally include forbidden broker/order payload keys. The Phase 1 ingestion boundary is therefore validated by the narrower service contract and tests instead of this package-wide grep: `Service.Ingest` depends only on `eventStore`, stores accepted/quarantined/rejected macro events, maps ETFs only for accepted events, and has no broker/order/execution interface.

- [x] **Step 4: Commit verification docs if needed**

Only add a short note if there is a useful result to preserve:

```powershell
git add Docs/plans/jax_complete_trading_readiness_docs/macro-reaction-engine/12_PHASE_1_IMPLEMENTATION_PLAN.md
git commit -m "docs: plan macro reaction engine phase 1"
```

## Defer Until Later Phases

- Do not create chart reaction snapshots until Phase 2.
- Do not create scenario playbooks until Phase 3.
- Do not create priced-in/confounder checks until Phase 4.
- Do not create evidence bundles until Phase 5.
- Do not create paper candidates until Phase 6.
- Do not add UI/API routes until Phase 7.

## Acceptance Checklist

- [x] `macro_events` table exists.
- [x] `macro_event_etf_map` table exists.
- [x] Valid NFP/CPI/Fed events persist.
- [x] Invalid events reject or quarantine.
- [x] Duplicate `source/source_event_id` is idempotent.
- [x] ETF mapping uses the existing allowlist/catalog.
- [x] No candidate trade is created.
- [x] No order or broker execution path is added.
- [x] Focused tests pass.
- [x] Broader backend tests pass.

## Completion Evidence

Closed on 2026-06-17.

Implementation evidence:

- Schema and migration: `db/postgres/migrations/000031_macro_events.up.sql`, `db/postgres/migrations/000031_macro_events.down.sql`, `db/postgres/migrations/macro_events_schema_test.go`.
- Domain validation: `internal/modules/macroevents/model.go`, `internal/modules/macroevents/validation.go`, `internal/modules/macroevents/validation_test.go`.
- ETF allowlist mapping: `internal/modules/macroevents/mapping.go`, `internal/modules/macroevents/mapping_test.go`.
- Ingestion store/service: `internal/modules/macroevents/store.go`, `internal/modules/macroevents/service.go`, `internal/modules/macroevents/service_test.go`.
- Read-only API integration: `cmd/trader/macro_api.go`, `cmd/trader/macro_api_test.go`.
- No trading boundary: macro ingestion tests use the narrow `eventStore` interface only, and `TestServiceIngestDoesNotExposeTradingHooks` prevents the ingestion service contract from growing broker/order hooks.

Verification evidence:

```powershell
go test ./db/postgres/migrations ./internal/modules/macroevents -count=1
go test ./cmd/trader ./internal/modules/... ./libs/... -count=1
```

Notes:

- Later phases added chart reactions, scenarios, priced-in checks, evidence bundles, paper-only macro candidates, and read-only API/UI surfaces. Those later additions do not change the Phase 1 boundary: macro event ingestion does not create candidate trades, approvals, broker orders, or execution records.
- Live trading remains outside this plan and must stay gated by the broker/runtime safety controls.
