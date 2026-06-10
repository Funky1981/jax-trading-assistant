# World Monitor Jax Awareness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local-only World Monitor to Jax bridge that accepts untrusted research triggers, stores them in a Jax research inbox, persists accepted events into existing event tables, and never creates trades, orders, execution instructions, runtime overrides, or approvals.

**Architecture:** Keep World Monitor, the adapter, and Jax as separate systems. Jax receives adapter payloads through a protected research-trigger endpoint, validates them, writes a dedicated inbox/audit row, and only persists accepted triggers into existing `event_raw`, `event_normalized`, and `event_symbol_map` tables. Candidate trade creation remains outside this ingestion path and must require later Jax evidence checks plus human approval.

**Tech Stack:** Go, `net/http`, `pgxpool`, PostgreSQL migrations under `db/postgres/migrations`, existing Jax event storage in `cmd/trader/event_store.go`, package tests run through `scripts/go-verify.ps1`.

---

## Source Review

Reviewed docs:

- `Docs/plans/world-monitor-jax-awareness/README.md`
- `Docs/plans/world-monitor-jax-awareness/01-local-runtime-setup.md`
- `Docs/plans/world-monitor-jax-awareness/02-signal-contract.md`
- `Docs/plans/world-monitor-jax-awareness/03-jax-ingestion-flow.md`
- `Docs/plans/world-monitor-jax-awareness/04-guardrails-and-safety.md`
- `Docs/plans/world-monitor-jax-awareness/05-codex-implementation-prompts.md`
- `Docs/plans/world-monitor-jax-awareness/06-test-plan.md`
- `Docs/plans/world-monitor-jax-awareness/07-separate-systems-control-layer.md`

Relevant repo findings:

- Existing Jax event storage already lives in `cmd/trader/event_store.go` and writes to `event_sources`, `event_raw`, `event_normalized`, and `event_symbol_map`.
- Existing event API routes are registered from `cmd/trader/codex_api.go` and exposed by the frontend API server in `cmd/trader/frontend_api.go`.
- Candidate trades and execution instructions are separate from events in `internal/modules/candidates`, `internal/modules/approvals`, `db/postgres/migrations/000014_candidate_trades.up.sql`, and `000015_candidate_approvals.up.sql`.
- Latest migration is `000029_sentiment_feature_layer.up.sql`; the World Monitor inbox should use `000030_world_monitor_research_inbox.*.sql`.
- Current working tree is dirty with unrelated `Docs/plans/ai-ready` moves/deletions. Do not touch or revert those as part of this work.

## Branch And Private Remote Setup

Do this before implementation, from `C:\Projects\jax-trading-assistant`.

- [ ] **Step 1: Inspect current state**

Run:

```powershell
git status --short --branch
git remote -v
```

Expected:

```text
origin  https://github.com/Funky1981/jax-trading-assistant.git (fetch)
origin  https://github.com/Funky1981/jax-trading-assistant.git (push)
```

- [ ] **Step 2: Create a private working branch**

Run:

```powershell
git switch -c personal/world-monitor-jax-awareness
```

Expected: new local branch `personal/world-monitor-jax-awareness`.

- [ ] **Step 3: Add your own remote**

Replace the URL with your own GitHub repo or fork.

```powershell
git remote add personal https://github.com/<your-user>/jax-trading-assistant.git
git remote -v
```

Expected: both `origin` and `personal` exist.

- [ ] **Step 4: Protect against accidental pushes to upstream**

Option A keeps fetching from upstream but disables pushes to `origin`:

```powershell
git remote set-url --push origin DISABLED
git remote -v
```

Expected:

```text
origin   https://github.com/Funky1981/jax-trading-assistant.git (fetch)
origin   DISABLED (push)
personal https://github.com/<your-user>/jax-trading-assistant.git (fetch)
personal https://github.com/<your-user>/jax-trading-assistant.git (push)
```

- [ ] **Step 5: Push only to your remote**

Run:

```powershell
git push -u personal personal/world-monitor-jax-awareness
```

Expected: branch tracks `personal/personal/world-monitor-jax-awareness`.

## File Structure

Create or modify only these Jax files for phase 1:

- Create: `db/postgres/migrations/000030_world_monitor_research_inbox.up.sql`
- Create: `db/postgres/migrations/000030_world_monitor_research_inbox.down.sql`
- Create: `db/postgres/migrations/world_monitor_research_inbox_schema_test.go`
- Create: `cmd/trader/world_monitor_research_trigger.go`
- Create: `cmd/trader/world_monitor_research_trigger_test.go`
- Create: `cmd/trader/world_monitor_research_inbox.go`
- Create: `cmd/trader/world_monitor_research_handlers.go`
- Create: `cmd/trader/world_monitor_research_handlers_test.go`
- Modify: `cmd/trader/event_store.go`
- Modify: `cmd/trader/codex_api.go`

Create the adapter as a separate sibling repo, not inside Jax:

- Create: `C:\Projects\world-monitor-jax-adapter\go.mod`
- Create: `C:\Projects\world-monitor-jax-adapter\cmd\adapter\main.go`
- Create: `C:\Projects\world-monitor-jax-adapter\examples\macro-rates.json`
- Create: `C:\Projects\world-monitor-jax-adapter\README.md`

Do not edit:

- `Agent0/`
- `dexter/`
- `internal/modules/approvals`
- `internal/modules/execution`
- broker/order handlers
- existing candidate creation code except test assertions that prove it is untouched

## Task 1: Inbox Schema

**Files:**

- Create: `db/postgres/migrations/000030_world_monitor_research_inbox.up.sql`
- Create: `db/postgres/migrations/000030_world_monitor_research_inbox.down.sql`
- Create: `db/postgres/migrations/world_monitor_research_inbox_schema_test.go`

- [ ] **Step 1: Write the schema test**

Add a migration text test that checks the table, constraints, indexes, and down migration.

```go
package migrations

import (
	"path/filepath"
	"testing"
)

func TestWorldMonitorResearchInboxMigrationDefinesRequiredTablesConstraintsAndIndexes(t *testing.T) {
	dir := migrationsDir(t)

	up := readMigration(t, filepath.Join(dir, "000030_world_monitor_research_inbox.up.sql"))
	down := readMigration(t, filepath.Join(dir, "000030_world_monitor_research_inbox.down.sql"))

	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS world_monitor_research_inbox",
		"world_monitor_event_id TEXT NOT NULL",
		"status TEXT NOT NULL DEFAULT 'new'",
		"severity TEXT NOT NULL",
		"source_tier TEXT NOT NULL",
		"confidence_reasons JSONB NOT NULL DEFAULT '[]'::jsonb",
		"raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb",
		"normalized_event_id UUID REFERENCES event_normalized(id)",
		"candidate_id UUID REFERENCES candidate_trades(id)",
		"CONSTRAINT uq_world_monitor_source_event",
		"chk_world_monitor_inbox_status",
		"chk_world_monitor_inbox_severity",
		"chk_world_monitor_inbox_source_tier",
		"idx_world_monitor_inbox_status",
		"idx_world_monitor_inbox_event_time",
		"idx_world_monitor_inbox_normalized_event",
	} {
		requireContains(t, up, fragment)
	}

	for _, fragment := range []string{
		"DROP INDEX IF EXISTS idx_world_monitor_inbox_normalized_event",
		"DROP TABLE IF EXISTS world_monitor_research_inbox",
	} {
		requireContains(t, down, fragment)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```powershell
go test ./db/postgres/migrations -run TestWorldMonitorResearchInboxMigrationDefinesRequiredTablesConstraintsAndIndexes -count=1
```

Expected: fail because migration files do not exist yet.

- [ ] **Step 3: Add the up migration**

Create `000030_world_monitor_research_inbox.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS world_monitor_research_inbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    world_monitor_event_id TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'new',
    rejection_reason TEXT,
    event_type TEXT NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT,
    source_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_count INTEGER NOT NULL DEFAULT 0,
    event_time TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    region TEXT,
    possible_affected_etfs JSONB NOT NULL DEFAULT '[]'::jsonb,
    asset_themes JSONB NOT NULL DEFAULT '[]'::jsonb,
    severity TEXT NOT NULL,
    source_tier TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    confidence_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    mapping_reason TEXT NOT NULL,
    dedupe_key TEXT NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    normalized_event_id UUID REFERENCES event_normalized(id) ON DELETE SET NULL,
    research_summary_id UUID REFERENCES research_summaries(id) ON DELETE SET NULL,
    candidate_id UUID REFERENCES candidate_trades(id) ON DELETE SET NULL,
    operator_decision TEXT,
    operator_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_world_monitor_source_event UNIQUE (source, source_event_id),
    CONSTRAINT uq_world_monitor_dedupe_key UNIQUE (dedupe_key),
    CONSTRAINT chk_world_monitor_inbox_status CHECK (
        status IN ('new', 'ignored', 'researching', 'candidate_created', 'rejected', 'archived')
    ),
    CONSTRAINT chk_world_monitor_inbox_severity CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT chk_world_monitor_inbox_source_tier CHECK (
        source_tier IN ('tier1', 'tier2', 'tier3', 'unknown')
    ),
    CONSTRAINT chk_world_monitor_confidence CHECK (
        confidence >= 0 AND confidence <= 1
    ),
    CONSTRAINT chk_world_monitor_source_count CHECK (
        source_count >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_status
    ON world_monitor_research_inbox(status, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_event_time
    ON world_monitor_research_inbox(event_time DESC);
CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_normalized_event
    ON world_monitor_research_inbox(normalized_event_id)
    WHERE normalized_event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_world_monitor_inbox_candidate
    ON world_monitor_research_inbox(candidate_id)
    WHERE candidate_id IS NOT NULL;
```

- [ ] **Step 4: Add the down migration**

Create `000030_world_monitor_research_inbox.down.sql`:

```sql
DROP INDEX IF EXISTS idx_world_monitor_inbox_candidate;
DROP INDEX IF EXISTS idx_world_monitor_inbox_normalized_event;
DROP INDEX IF EXISTS idx_world_monitor_inbox_event_time;
DROP INDEX IF EXISTS idx_world_monitor_inbox_status;
DROP TABLE IF EXISTS world_monitor_research_inbox;
```

- [ ] **Step 5: Run the schema test**

Run:

```powershell
go test ./db/postgres/migrations -run TestWorldMonitorResearchInboxMigrationDefinesRequiredTablesConstraintsAndIndexes -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

```powershell
git add db/postgres/migrations/000030_world_monitor_research_inbox.up.sql db/postgres/migrations/000030_world_monitor_research_inbox.down.sql db/postgres/migrations/world_monitor_research_inbox_schema_test.go
git commit -m "feat: add world monitor research inbox schema"
```

## Task 2: Research Trigger Contract And Validation

**Files:**

- Create: `cmd/trader/world_monitor_research_trigger.go`
- Create: `cmd/trader/world_monitor_research_trigger_test.go`

- [ ] **Step 1: Write validation tests**

Cover valid payloads, missing timestamp, missing URLs, low source count, stale events, trade language, unknown low-confidence events, non-ETF mappings, missing confidence reasons, and runtime override attempts.

Use `time.Now().UTC()` in tests and inject `now` into validation so freshness checks are deterministic.

- [ ] **Step 2: Run tests and verify they fail**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearchTrigger" -count=1
```

Expected: fail because the contract does not exist.

- [ ] **Step 3: Implement the typed contract**

Create request and receipt types with JSON names matching the docs:

```go
type worldMonitorResearchTrigger struct {
	Source                string         `json:"source"`
	SourceEventID         string         `json:"source_event_id"`
	EventType             string         `json:"event_type"`
	Headline              string         `json:"headline"`
	Summary               string         `json:"summary"`
	SourceURLs            []string       `json:"source_urls"`
	SourceCount           int            `json:"source_count"`
	TimestampUTC          time.Time      `json:"timestamp_utc"`
	Region                string         `json:"region"`
	PossibleAffectedETFs  []string       `json:"possible_affected_etfs"`
	AssetThemes           []string       `json:"asset_themes"`
	Severity              string         `json:"severity"`
	SourceTier            string         `json:"source_tier"`
	Confidence            float64        `json:"confidence"`
	ConfidenceReasons     []string       `json:"confidence_reasons"`
	Reason                string         `json:"reason"`
	RawPayload            map[string]any `json:"raw_payload"`
}

type worldMonitorResearchReceipt struct {
	InboxID         string `json:"inbox_id,omitempty"`
	EventID         string `json:"event_id,omitempty"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Duplicate       bool   `json:"duplicate"`
}
```

- [ ] **Step 4: Implement validation**

Rules:

- required fields: `source`, `source_event_id`, `event_type`, `headline`, `source_urls`, `source_count`, `timestamp_utc`, `possible_affected_etfs`, `confidence`, `confidence_reasons`, `reason`
- allowed event types from `02-signal-contract.md`
- allowed severity values: `low`, `medium`, `high`, `critical`
- allowed source tiers: `tier1`, `tier2`, `tier3`, `unknown`
- freshness window: default 24 hours
- source count: `>= 2`, except `tier1` can pass with one source
- direct trade language rejects the payload when found in `headline`, `summary`, `reason`, or `raw_payload`
- reject runtime mode fields such as `runtime_mode`, `execution_enabled`, `broker_order`, `order`, `position_size`, `risk_override`
- reject leveraged, inverse, and volatility ETFs using the existing instrument catalog in `internal/modules/instruments`
- reject confidence scores without `confidence_reasons`

- [ ] **Step 5: Run validation tests**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearchTrigger" -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

```powershell
git add cmd/trader/world_monitor_research_trigger.go cmd/trader/world_monitor_research_trigger_test.go
git commit -m "feat: validate world monitor research triggers"
```

## Task 3: Inbox Service And Event Persistence

**Files:**

- Create: `cmd/trader/world_monitor_research_inbox.go`
- Modify: `cmd/trader/event_store.go`
- Create or extend: `cmd/trader/world_monitor_research_trigger_test.go`

- [ ] **Step 1: Write service tests around pure behavior**

Test:

- dedupe key is stable for same `source + source_event_id + headline`
- accepted trigger maps to `persistEventInput` with `SourceID = "world-monitor"`
- accepted trigger uses `EventKind = "research_trigger"`
- `Attributes` include `eventType`, `sourceUrls`, `sourceCount`, `assetThemes`, `confidenceReasons`, `worldMonitorEventId`, and `mappingReason`
- low severity is accepted into inbox as `ignored` and is not persisted to `event_normalized`
- invalid trigger returns status `rejected`

- [ ] **Step 2: Run tests and verify they fail**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearchInbox" -count=1
```

Expected: fail because inbox service is not implemented.

- [ ] **Step 3: Add an event store method that returns IDs**

Keep existing `persistEvent` behavior intact and add a new method:

```go
type persistedEventRef struct {
	RawID        string
	NormalizedID string
}

func (s *eventStore) persistEventWithRef(ctx context.Context, in persistEventInput) (persistedEventRef, error) {
	// Move current persistEvent transaction body here and return rawID plus normalizedID.
}

func (s *eventStore) persistEvent(ctx context.Context, in persistEventInput) error {
	_, err := s.persistEventWithRef(ctx, in)
	return err
}
```

Do not change callers such as `SaveNews`, `SaveEarnings`, or `SaveMacroNews`.

- [ ] **Step 4: Implement inbox service**

Create `worldMonitorResearchInboxService` with:

- `Validate(trigger, now) validationResult`
- `Ingest(ctx, trigger) (worldMonitorResearchReceipt, error)`
- `toPersistEventInput(trigger) persistEventInput`
- `worldMonitorDedupeKey(trigger) string`

Accepted `medium`, `high`, and `critical` triggers should persist to event storage and insert/update `world_monitor_research_inbox` with status `new`. Valid `low` triggers should insert inbox status `ignored` and skip event persistence by default.

- [ ] **Step 5: Ensure the service never imports candidate, approval, execution, or broker packages**

Run:

```powershell
rg -n "internal/modules/(candidates|approvals|execution)|broker|execution_instructions|candidate_trades" cmd/trader/world_monitor_research_inbox.go cmd/trader/world_monitor_research_trigger.go
```

Expected: no matches except SQL column names in inbox insert/select if needed for `candidate_id` references.

- [ ] **Step 6: Run service tests**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearchInbox|TestWorldMonitorResearchTrigger|TestDeterministicEventIDStable|TestNormalizeSymbols" -count=1
```

Expected: pass.

- [ ] **Step 7: Commit**

```powershell
git add cmd/trader/world_monitor_research_inbox.go cmd/trader/event_store.go cmd/trader/world_monitor_research_trigger_test.go
git commit -m "feat: ingest world monitor triggers into research inbox"
```

## Task 4: HTTP Endpoint

**Files:**

- Create: `cmd/trader/world_monitor_research_handlers.go`
- Create: `cmd/trader/world_monitor_research_handlers_test.go`
- Modify: `cmd/trader/codex_api.go`

- [ ] **Step 1: Write handler tests**

Test cases:

- `POST /api/v1/research/events/world-monitor` accepts a valid high-severity trigger and returns `202`
- duplicate valid trigger returns `200` with `duplicate: true`
- invalid trigger returns `400` or `422` with `status: rejected`
- `GET` returns `405`
- payload containing `buy QQQ now` is rejected

- [ ] **Step 2: Run tests and verify they fail**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearchHandler" -count=1
```

Expected: fail because handler is not implemented.

- [ ] **Step 3: Implement handler**

Create:

```go
func worldMonitorResearchIngestHandler(pool *pgxpool.Pool) http.HandlerFunc
```

Behavior:

- require `POST`
- decode JSON with `json.Decoder`
- reject malformed JSON as `400`
- call the inbox service
- return accepted receipts as `202 Accepted`
- return duplicate receipts as `200 OK`
- return rejected receipts as `422 Unprocessable Entity`
- return persistence errors as `500`

- [ ] **Step 4: Register route**

Modify `registerCodexAPIRoutes` in `cmd/trader/codex_api.go`:

```go
mux.HandleFunc("/api/v1/research/events/world-monitor", protect(worldMonitorResearchIngestHandler(pool)))
```

Place it near existing `/api/v1/research/...` routes.

- [ ] **Step 5: Run handler tests**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearchHandler|TestWorldMonitorResearchTrigger" -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

```powershell
git add cmd/trader/world_monitor_research_handlers.go cmd/trader/world_monitor_research_handlers_test.go cmd/trader/codex_api.go
git commit -m "feat: expose world monitor research ingest endpoint"
```

## Task 5: Separate Adapter Repo Skeleton

**Files:**

- Create: `C:\Projects\world-monitor-jax-adapter\go.mod`
- Create: `C:\Projects\world-monitor-jax-adapter\cmd\adapter\main.go`
- Create: `C:\Projects\world-monitor-jax-adapter\examples\macro-rates.json`
- Create: `C:\Projects\world-monitor-jax-adapter\README.md`

- [ ] **Step 1: Create sibling repo**

```powershell
New-Item -ItemType Directory -Path C:\Projects\world-monitor-jax-adapter
Set-Location C:\Projects\world-monitor-jax-adapter
git init
```

- [ ] **Step 2: Initialize Go module**

```powershell
go mod init world-monitor-jax-adapter
```

- [ ] **Step 3: Implement local file-to-Jax adapter**

`cmd/adapter/main.go` should:

- read `-input` JSON file
- read `-jax-url`, default `http://localhost:8080`
- POST to `/api/v1/research/events/world-monitor`
- accept `-token` for the Jax API bearer token if auth is enabled
- log response status and body
- contain no broker credentials, order types, execution flags, or runtime mode controls

- [ ] **Step 4: Add example payload**

`examples/macro-rates.json` should use the contract from `02-signal-contract.md`, plus `severity`, `source_tier`, and `confidence_reasons`.

- [ ] **Step 5: Run local adapter build**

```powershell
go test ./...
go run .\cmd\adapter -input .\examples\macro-rates.json -jax-url http://localhost:8080
```

Expected: build passes; runtime request reaches Jax when Jax is running.

- [ ] **Step 6: Commit adapter repo**

```powershell
git add .
git commit -m "feat: add local world monitor to jax adapter"
```

## Task 6: No-Trade Smoke Test

**Files:**

- Create: `cmd/trader/world_monitor_research_smoke_test.go`

- [ ] **Step 1: Write smoke test**

The smoke test should ingest a valid macro rates payload and assert:

- one inbox row exists
- status is `new`
- one normalized event exists
- symbol map contains `QQQ`, `SPY`, and `TLT`
- zero candidate trades are linked to the inbox
- no candidate approval exists
- no execution instruction exists

- [ ] **Step 2: Run smoke test**

```powershell
go test ./cmd/trader -run "TestWorldMonitorResearch_NoTradeCreated" -count=1
```

Expected: pass when test database setup is available. If the repo lacks local DB test helpers for `cmd/trader`, keep the test behind the repo's existing integration-test convention and document the required `DATABASE_URL`.

- [ ] **Step 3: Run focused backend verification**

```powershell
scripts/go-verify.ps1 -Mode quick -Packages ./cmd/trader,./db/postgres/migrations
```

Expected: pass.

- [ ] **Step 4: Run standard backend verification**

```powershell
scripts/go-verify.ps1 -Mode standard -Packages ./cmd/trader,./db/postgres/migrations
```

Expected: pass.

- [ ] **Step 5: Run behavior-sensitive checks**

Because this touches event/candidate boundaries, run:

```powershell
scripts/golden-check.ps1 -Mode verify
```

Expected: no unexpected golden/replay deltas. Do not capture new baselines unless a reviewed intentional behavior change exists.

- [ ] **Step 6: Commit**

```powershell
git add cmd/trader/world_monitor_research_smoke_test.go
git commit -m "test: prove world monitor ingest cannot create trades"
```

## Final Verification

Run from `C:\Projects\jax-trading-assistant`:

```powershell
gofmt -w cmd/trader/world_monitor_research_trigger.go cmd/trader/world_monitor_research_inbox.go cmd/trader/world_monitor_research_handlers.go cmd/trader/*world_monitor*_test.go
go test ./db/postgres/migrations -count=1
go test ./cmd/trader -run "WorldMonitor|TestDeterministicEventIDStable|TestNormalizeSymbols" -count=1
scripts/go-verify.ps1 -Mode standard -Packages ./cmd/trader,./db/postgres/migrations
scripts/golden-check.ps1 -Mode verify
```

Run from `C:\Projects\world-monitor-jax-adapter`:

```powershell
gofmt -w .\cmd\adapter\main.go
go test ./...
```

## Acceptance Criteria

- World Monitor integration route accepts only research triggers.
- Invalid, stale, weak-source, low-evidence, trade-language, runtime-mode, broker/order, and disallowed ETF payloads are rejected or held as awareness-only.
- Accepted triggers are persisted in `world_monitor_research_inbox`.
- Accepted medium/high/critical triggers are persisted into existing event tables.
- No candidate, approval, order, execution instruction, broker request, runtime override, or live trading path is created by ingestion.
- Adapter remains a separate local repo/service with no broker credentials.
- Branch pushes go only to `personal`, not `origin`.

## What's Left

- Provide the actual private remote URL before running the branch setup commands.
- Decide whether low-severity valid triggers should be stored as `ignored` only or completely rejected.
- Decide whether the Jax endpoint should use existing frontend auth only or require a dedicated adapter token.
- After implementation, run DB-backed smoke tests against a local Postgres instance with migrations through `000030`.
