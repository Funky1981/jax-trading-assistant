# AI News Trading Productization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current research/paper-trading shell into a usable local AI trading workflow where news and market data create reviewable opportunities, ETF trades enter approval, approved opportunities become paper broker orders, and users understand what each page is for.

**Architecture:** Keep Jax World News Monitor, Agent0, Jax Research, and Jax Trader as separate services. Add a Jax-side opportunity pipeline that consumes Jax World News Monitor research triggers, market quotes/candles, and Agent0 analysis, then writes durable `strategy_signals` and `candidate_trades` rows for the existing AI Trading and Approvals UI. Operator screens must show only real/provenance-labelled data by default and must never route users to an empty dead-end.

**Tech Stack:** Go backend in `cmd/trader` and `internal/modules/*`, React/Vite frontend in `frontend/src`, Postgres via Docker Compose, Playwright/Vitest/Go tests for verification.

---

## Review Findings

1. **AI suggestions are advisory only.** `AI Trading -> Ask Jax` proves Agent0 is connected, but the result is not converted into a candidate, approval, or paper order. It is a detached answer.
2. **The scanner is not producing opportunities.** Runtime `GET /api/v1/ai/overview` reports `signalsPending=0`, `candidates=0`, and `approvals=0`. The UI is showing an empty queue because the backend pipeline is not creating rows.
3. **ETF order policy dead-ends.** Manual ETF entries are intentionally blocked and point to approvals, but approvals are empty because no candidate/approval generator is running.
4. **Macro Events currently show fixture data.** Runtime DB check returned `macro_events.source = test` for all rows. Operator pages should not show test fixtures as live macro intelligence.
5. **Jax World News Monitor ingestion is only research inbox.** `POST /api/v1/research/events/world-monitor` accepts payloads and writes research-only rows, but nothing promotes accepted triggers into candidates.
6. **Analysis page is under-explained.** It inspects completed backtests, dataset provenance, trades, and event correlation. It does not currently explain that it is not the AI decision page.
7. **Backtest research works after recent fixes, but it is not the production loop.** It creates completed backtest runs from dataset snapshots; it does not generate actionable live/paper opportunities.

## Target User Workflow

1. Jax World News Monitor produces a relevant research trigger.
2. Jax ingests the trigger and labels source/provenance.
3. Jax enriches the trigger with market data, recent candles, technical state, and optional Agent0 reasoning.
4. Jax creates a `candidate_trade` if evidence passes minimum gates.
5. If candidate is an ETF, Jax creates an approval item and AI Trading shows it as `Approval required`.
6. User opens the approval, sees plain-English evidence, approves/rejects.
7. Approved paper candidates create broker instructions/orders in paper mode only.
8. Analysis is used after the fact to inspect backtests and paper-trading performance.

---

## Task 1: Hide and Purge Fixture Macro Data From Operator Screens

**Files:**
- Modify: `cmd/trader/macro_api.go`
- Modify: `frontend/src/pages/MacroEventsPage.tsx`
- Modify: `cmd/trader/macro_api_test.go`
- Create: `scripts/clean-fixture-runtime-data.ps1`

- [x] **Step 1: Add source filtering to macro API**

In `cmd/trader/macro_api.go`, update `macroEventsHandler` to exclude fixture/test rows by default:

```go
includeFixtures := strings.EqualFold(r.URL.Query().Get("includeFixtures"), "true")
where := "WHERE me.source NOT IN ('test', 'fixture') AND COALESCE(me.raw_payload->>'fixture', 'false') <> 'true'"
if includeFixtures {
    where = ""
}
```

Apply `where` to both the list query and the count query.

- [x] **Step 2: Add a visible empty state**

In `frontend/src/pages/MacroEventsPage.tsx`, show:

```tsx
No live macro events have been ingested yet. Connect Jax World News Monitor or a macro calendar feed to populate this page.
```

Only show fixture data when the URL/query explicitly includes `includeFixtures=true`.

- [x] **Step 3: Add cleanup script**

Create `scripts/clean-fixture-runtime-data.ps1`:

```powershell
$ErrorActionPreference = "Stop"
docker compose exec -T postgres psql -U jax -d jax -v ON_ERROR_STOP=1 -c "
DELETE FROM macro_candidate_trades WHERE macro_event_id IN (SELECT id FROM macro_events WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true');
DELETE FROM macro_evidence_bundles WHERE macro_event_id IN (SELECT id FROM macro_events WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true');
DELETE FROM macro_event_etf_map WHERE macro_event_id IN (SELECT id FROM macro_events WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true');
DELETE FROM macro_events WHERE source IN ('test','fixture') OR raw_payload->>'fixture' = 'true';
"
```

- [x] **Step 4: Test**

Run:

```powershell
go test ./cmd/trader
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/clean-fixture-runtime-data.ps1
```

Expected:
- Macro API default list does not return `source=test`.
- Macro page no longer presents test rows as live data.

- [x] **Step 5: Commit**

```powershell
git add cmd/trader/macro_api.go frontend/src/pages/MacroEventsPage.tsx cmd/trader/macro_api_test.go scripts/clean-fixture-runtime-data.ps1
git commit -m "fix: hide fixture macro data from operator screens"
```

---

## Task 2: Build Jax World News Monitor Trigger Promotion Into Candidate Trades

**Files:**
- Create: `cmd/trader/world_monitor_opportunity_promoter.go`
- Create: `cmd/trader/world_monitor_opportunity_promoter_test.go`
- Modify: `cmd/trader/world_monitor_research_inbox.go`
- Modify: `cmd/trader/main.go` or `cmd/trader/frontend_api.go`

- [x] **Step 1: Define promotion input/output**

Create `cmd/trader/world_monitor_opportunity_promoter.go`:

```go
type worldMonitorOpportunityPromoter struct {
    pool *pgxpool.Pool
    now  func() time.Time
}

type promotedOpportunity struct {
    CandidateID string
    ApprovalID  string
    Route       string
    Status      string
}
```

- [x] **Step 2: Load eligible inbox rows**

Select `world_monitor_research_inbox` rows where:

```sql
status = 'new'
AND normalized_event_id IS NOT NULL
AND candidate_id IS NULL
AND source_count >= 1
AND confidence >= 0.55
```

For phase 1, cap each run at `10` rows ordered by `received_at ASC`.

- [x] **Step 3: Create signal-linked candidate rows**

For each eligible trigger, create one candidate per primary ETF in `possible_affected_etfs`. Insert into `candidate_trades` using:

```text
status = awaiting_approval
symbol = ETF symbol
side = BUY or WATCH based on event type and confidence
source = world-monitor
strategy_id = world_monitor_research_v1
reasoning = headline + mapping reason
confidence = trigger confidence
```

Store the source inbox ID, source event ID, headline, URLs, and Agent0 status in metadata/rule trace.

- [x] **Step 4: Queue ETF candidates for approval**

Use the existing candidate lifecycle so `GET /api/v1/approvals/queue` returns `candidate_trades.status = 'awaiting_approval'`. Do not pre-create `candidate_approvals` decision rows; those rows are created only when a human approves, rejects, snoozes, or requests reanalysis. Route all ETFs through approval; do not allow direct execution.

- [x] **Step 5: Mark inbox row promoted**

Add or reuse a status such as:

```text
candidate_created
```

Set `candidate_id` on `world_monitor_research_inbox`.

- [x] **Step 6: Test**

Create `world_monitor_opportunity_promoter_test.go` with a valid Jax World News Monitor trigger for `QQQ`. Assert:
- one `strategy_signals` row exists and is linked
- one `candidate_trades` row exists
- the candidate appears in the approval queue
- no execution instruction or broker order exists before human approval

Run:

```powershell
go test ./cmd/trader -run WorldMonitor
go test ./internal/modules/approvals ./internal/modules/candidates
```

- [ ] **Step 7: Commit**

```powershell
git add cmd/trader/world_monitor_opportunity_promoter.go cmd/trader/world_monitor_opportunity_promoter_test.go cmd/trader/world_monitor_research_inbox.go
git commit -m "feat: promote Jax World News Monitor triggers to approval candidates"
```

---

## Task 3: Add a Manual “Promote to Opportunity” Button for AI Suggestions

**Files:**
- Modify: `frontend/src/pages/AiTradingPage.tsx`
- Modify: `frontend/src/data/ai-service.ts`
- Modify: `cmd/trader/ai_overview_handlers.go` or create `cmd/trader/ai_opportunity_handlers.go`
- Test: `frontend/src/pages/AiTradingPage.test.tsx`

- [x] **Step 1: Add endpoint**

Expose:

```text
POST /api/v1/ai/suggestions/promote
```

Request:

```json
{
  "symbol": "SPY",
  "action": "BUY",
  "confidence": 0.62,
  "reasoning": "Agent0 reasoning text",
  "risk": "low",
  "source": "agent0_manual_review"
}
```

Response:

```json
{
  "candidateId": "...",
  "approvalId": "...",
  "route": "approval_required"
}
```

- [x] **Step 2: Enforce policy**

If symbol is ETF, create `awaiting_approval`. If symbol is non-ETF and paper mode allows manual route, create `detected` with route `manual_allowed`. Never place an order from this endpoint.

- [x] **Step 3: Add UI action**

In `AiTradingPage.tsx`, after an AI suggestion appears, show:

```tsx
<Button>Send to opportunity queue</Button>
```

For ETF symbols, button copy should be:

```text
Send to approval queue
```

- [x] **Step 4: Test**

Run:

```powershell
npm run test -- --run src/pages/AiTradingPage.test.tsx
go test ./cmd/trader -run AI
```

Expected:
- Suggestion can become a durable candidate.
- AI Trading queue count updates.
- Approvals page is no longer empty after promoting an ETF suggestion.

- [ ] **Step 5: Commit**

```powershell
git add frontend/src/pages/AiTradingPage.tsx frontend/src/data/ai-service.ts cmd/trader/*ai* frontend/src/pages/AiTradingPage.test.tsx
git commit -m "feat: promote ai suggestions into opportunity queue"
```

---

## Task 4: Make Approvals Execute Paper Orders End-to-End

**Files:**
- Inspect/modify: `cmd/trader/approval_handlers.go`
- Inspect/modify: `cmd/trader/execution_instruction_worker.go`
- Inspect/modify: `internal/modules/execution/*`
- Modify: `frontend/src/pages/ApprovalsPage.tsx`
- Add/modify: Playwright spec under `frontend/e2e/trading.spec.ts`

- [ ] **Step 1: Verify approval endpoint behavior**

Use a seeded candidate and call:

```text
POST /api/v1/approvals/{candidateId}/approve
```

Expected backend result:
- approval decision saved
- execution instruction created
- candidate status transitions to `approved` or `submitted`

- [ ] **Step 2: Ensure paper-only execution worker consumes approved candidates**

Confirm `execution_instruction_worker.go` is started in local paper runtime and only allows:

```text
JAX_TRADER_RUNTIME_MODE=paper
IB_PAPER_TRADING=true
ALLOW_LIVE_TRADING=false
```

- [ ] **Step 3: Add UI confirmation**

Approvals page must clearly show:

```text
Approve for paper order
```

and not vague “Approve” when the next action can create broker instructions.

- [ ] **Step 4: Playwright test**

Write an E2E that:
1. Creates a Jax World News Monitor trigger.
2. Promotes it to an ETF candidate.
3. Opens Approvals.
4. Approves it.
5. Confirms broker order or execution instruction exists.

Run:

```powershell
cd frontend
npm run test:e2e -- --grep "approval paper order"
```

- [ ] **Step 5: Commit**

```powershell
git add cmd/trader/approval_handlers.go cmd/trader/execution_instruction_worker.go frontend/src/pages/ApprovalsPage.tsx frontend/e2e/trading.spec.ts
git commit -m "feat: execute approved opportunities in paper mode"
```

---

## Task 5: Replace Passive Scanner With a Real Scheduled Opportunity Scan

**Files:**
- Create: `cmd/trader/opportunity_scanner.go`
- Create: `cmd/trader/opportunity_scanner_test.go`
- Modify: `cmd/trader/main.go`
- Modify: `cmd/trader/ai_overview_handlers.go`

- [ ] **Step 1: Define scanner loop**

Every `scanner.intervalSeconds`, scan configured symbols from `ai_scanner_state`.

Inputs:
- latest quote
- recent candles
- Jax World News Monitor/event_raw rows in last 24h
- sentiment aggregates if present
- existing candidates to dedupe

- [ ] **Step 2: Dedupe candidates**

Do not create duplicate candidates for same:

```text
symbol + strategy_id + source_event_id + direction
```

inside a configurable expiry window.

- [ ] **Step 3: Candidate creation rule**

For phase 1:
- confidence >= scanner minimum
- at least one valid event/news trigger OR strong technical setup
- valid stop/target/risk fields
- paper runtime only

- [ ] **Step 4: Test**

Run:

```powershell
go test ./cmd/trader -run OpportunityScanner
```

Expected:
- scanner creates candidates from eligible data
- scanner does not duplicate existing candidates
- scanner does not create broker orders

- [ ] **Step 5: Commit**

```powershell
git add cmd/trader/opportunity_scanner.go cmd/trader/opportunity_scanner_test.go cmd/trader/main.go cmd/trader/ai_overview_handlers.go
git commit -m "feat: scan news and market data into opportunities"
```

---

## Task 6: Explain Analysis and Connect It From Research

**Files:**
- Modify: `frontend/src/pages/AnalysisPage.tsx`
- Modify: `frontend/src/pages/ResearchPage.tsx`
- Test: `frontend/src/pages/ResearchPage.test.tsx`

- [ ] **Step 1: Add clear Analysis description**

Add a top card:

```text
Analysis is for reviewing completed backtests. It does not create trade ideas or place orders. Use it to understand whether a strategy has worked on a selected dataset.
```

- [ ] **Step 2: Add empty state**

When no `runId` is selected:

```text
Choose a completed backtest run from the selector, or open a run from Research -> Backtests.
```

- [ ] **Step 3: Make Research rows link clearly**

In `ResearchPage.tsx`, add an explicit button:

```text
Open in Analysis
```

for completed runs.

- [ ] **Step 4: Test**

Run:

```powershell
npm run test -- --run src/pages/ResearchPage.test.tsx
npm run typecheck
```

- [ ] **Step 5: Commit**

```powershell
git add frontend/src/pages/AnalysisPage.tsx frontend/src/pages/ResearchPage.tsx frontend/src/pages/ResearchPage.test.tsx
git commit -m "fix: explain analysis and link completed research runs"
```

---

## Task 7: Add Full Workflow Test Script

**Files:**
- Create: `scripts/test-ai-news-paper-workflow.ps1`
- Create: `frontend/e2e/ai-news-paper-workflow.spec.ts`

- [ ] **Step 1: Create script**

The script should:
1. Start stack or verify stack health.
2. Clean fixture macro rows.
3. Sync datasets.
4. POST a Jax World News Monitor trigger.
5. Run promoter/scanner once.
6. Verify AI Trading count > 0.
7. Open browser E2E and approve one paper ETF opportunity.

- [ ] **Step 2: Add Playwright spec**

Spec assertions:
- AI Trading shows at least one opportunity.
- Opportunity explains source headline, symbol, confidence, route.
- Approvals page has the same opportunity.
- Approving creates paper broker instruction/order.

- [ ] **Step 3: Run**

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-ai-news-paper-workflow.ps1
```

Expected:
- No fixture macro rows shown.
- One research trigger promoted.
- One approval created.
- One paper order or execution instruction created.
- No live-trading path used.

- [ ] **Step 4: Commit**

```powershell
git add scripts/test-ai-news-paper-workflow.ps1 frontend/e2e/ai-news-paper-workflow.spec.ts
git commit -m "test: add ai news paper workflow smoke"
```

---

## Done Criteria

- AI Trading shows at least one real opportunity after a Jax World News Monitor trigger or scanner pass.
- AI suggestion can be converted into a candidate/approval, not just read.
- ETF manual entry no longer dead-ends: it points to a populated approval route or explains how to create one.
- Macro Events default view has no fixture/test rows.
- Approvals can create paper execution instructions/orders with explicit confirmation.
- Analysis page explains that it reviews completed backtests and links from Research.
- One command runs the full local workflow test.
- Live trading remains disabled.
