# Swing V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Swing Trading as a separate, paper-only, approval-gated ETF trading option without weakening the existing intraday/news workflow.

**Architecture:** Extend the existing `internal/modules/tradingmodes` catalog and candidate evidence pipeline with explicit horizon policy. Swing candidates remain ETF-only, paper-only, and approval-first; they add multi-day thesis, daily revalidation, and overnight-risk controls rather than changing Manual Trading or existing intraday behavior.

**Tech Stack:** Go services in `cmd/trader` and `internal/modules`, Postgres migrations, React frontend under `frontend/src`, existing candidate/approval APIs, Vitest and Go tests.

---

## Current Blockers To Close First

- Do not move `Docs/plans/jax_complete_trading_readiness_docs` to `Completed` until unchecked macro Phase 1 and TA/FA UAT items are completed or explicitly superseded.
- Keep Swing paper-only. No live execution enablement belongs in Swing V2.
- Do not reuse intraday flatten-by-close assumptions for swing candidates.
- Do not let AI create execution instructions. AI can classify, summarize, and score evidence only.

## Task 1: Add Swing Trading Modes

**Files:**

- Modify: `internal/modules/tradingmodes/catalog.go`
- Modify: `internal/modules/tradingmodes/catalog_test.go`

- [ ] **Step 1: Write failing catalog tests**

Add tests requiring two new modes:

```go
func TestDefaultCatalogIncludesSwingResearchMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_swing_research")
	if !ok {
		t.Fatalf("expected etf_swing_research mode")
	}
	if mode.RuntimeMode != "research" {
		t.Fatalf("runtime mode = %q, want research", mode.RuntimeMode)
	}
	if mode.ExecutionPolicy != "no_execution" {
		t.Fatalf("execution policy = %q, want no_execution", mode.ExecutionPolicy)
	}
	if !mode.RiskDefaults.ApprovalRequired {
		t.Fatal("swing research must remain approval-gated before promotion")
	}
}

func TestDefaultCatalogIncludesSwingPaperMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_swing_paper")
	if !ok {
		t.Fatalf("expected etf_swing_paper mode")
	}
	if mode.RuntimeMode != "paper" {
		t.Fatalf("runtime mode = %q, want paper", mode.RuntimeMode)
	}
	if mode.ExecutionPolicy != "candidate_approval_only" {
		t.Fatalf("execution policy = %q, want candidate_approval_only", mode.ExecutionPolicy)
	}
	if mode.RiskDefaults.FlattenBy != "daily_revalidation" {
		t.Fatalf("flatten policy = %q, want daily_revalidation", mode.RiskDefaults.FlattenBy)
	}
}
```

- [ ] **Step 2: Run failing tests**

Run:

```powershell
go test ./internal/modules/tradingmodes
```

Expected: FAIL because the two modes do not exist.

- [ ] **Step 3: Add catalog modes**

Add:

- `etf_swing_research`: research runtime, no execution, ETF universe, daily/swing data requirements.
- `etf_swing_paper`: paper runtime, `candidate_approval_only`, smaller risk, daily revalidation.

Use existing ETF universe symbols and strategy refs:

- `etf_swing_macro_rates_rotation_v1`
- `etf_swing_sector_event_momentum_v1`
- `etf_swing_risk_on_off_reversal_v1`

- [ ] **Step 4: Verify tests pass**

Run:

```powershell
go test ./internal/modules/tradingmodes
```

Expected: PASS.

## Task 2: Add Horizon Policy Types

**Files:**

- Create: `internal/modules/tradingmodes/horizon.go`
- Create: `internal/modules/tradingmodes/horizon_test.go`

- [ ] **Step 1: Write horizon validation tests**

Required behavior:

- Intraday rejects overnight risk.
- Swing requires daily review.
- Swing rejects `MaxHoldDays > 10`.
- Swing defaults to no weekend hold.

- [ ] **Step 2: Implement horizon policy**

Create:

```go
type TradingHorizon string

const (
	HorizonResearchOnly TradingHorizon = "research_only"
	HorizonIntraday     TradingHorizon = "intraday"
	HorizonSwing        TradingHorizon = "swing"
)

type CandidateHorizonPolicy struct {
	Horizon              TradingHorizon `json:"horizon"`
	HoldTargetDays       int            `json:"holdTargetDays,omitempty"`
	MaxHoldDays          int            `json:"maxHoldDays,omitempty"`
	FlattenByClose       bool           `json:"flattenByClose"`
	OvernightRiskAllowed bool           `json:"overnightRiskAllowed"`
	WeekendHoldAllowed   bool           `json:"weekendHoldAllowed"`
	RequiresDailyReview  bool           `json:"requiresDailyReview"`
	RevalidationSchedule string         `json:"revalidationSchedule,omitempty"`
	ThesisInvalidators   []string       `json:"thesisInvalidators,omitempty"`
}
```

- [ ] **Step 3: Verify**

Run:

```powershell
go test ./internal/modules/tradingmodes
```

Expected: PASS.

## Task 3: Persist Candidate Horizon Metadata

**Files:**

- Add migration under `db/postgres/migrations/`
- Modify candidate creation code in `internal/modules/candidates`
- Modify World Monitor promotion code in `cmd/trader/world_monitor_opportunity_promoter.go`
- Test: candidate service and promoter tests

- [ ] **Step 1: Add migration test or migration smoke check**

Acceptance:

- `candidate_trades.metadata` or a dedicated horizon column stores the horizon policy.
- Existing candidates remain readable.
- Swing candidates can be queried separately from intraday candidates.

- [ ] **Step 2: Write failing service tests**

Assert proposed swing candidate metadata contains:

- `horizon: "swing"`
- `maxHoldDays <= 10`
- `requiresDailyReview: true`
- `paperOnly: true`
- `approvalRequired: true`

- [ ] **Step 3: Implement minimal persistence**

Prefer storing horizon policy in existing candidate metadata first unless query performance requires a new column.

- [ ] **Step 4: Verify**

Run:

```powershell
go test ./internal/modules/candidates ./cmd/trader -run "TestWorldMonitorOpportunity|TestCandidate"
```

## Task 4: Add Swing Research Engine Boundary

**Files:**

- Create: `internal/modules/swingresearch/`
- Modify: `cmd/trader/world_monitor_opportunity_promoter.go`
- Tests: `internal/modules/swingresearch/*_test.go`

- [ ] **Step 1: Write tests for swing thesis gating**

Required outcomes:

- Missing event source blocks.
- Missing daily candles blocks.
- Confounder present blocks or downgrades to watch.
- Valid evidence returns a swing thesis, not an order.

- [ ] **Step 2: Implement deterministic engine**

The engine should return:

- thesis summary
- mapped ETFs
- evidence IDs/source URLs
- historical reaction window
- invalidators
- daily review schedule
- risk notes
- blocker reasons

- [ ] **Step 3: Verify**

Run:

```powershell
go test ./internal/modules/swingresearch
```

## Task 5: Wire Swing As A UI Option

**Files:**

- Modify: `frontend/src/data/types.ts`
- Modify or add data service for `/api/v1/trading-modes`
- Modify relevant trading/research page navigation
- Add Vitest coverage

- [ ] **Step 1: Add frontend type coverage**

Ensure trading modes expose:

- mode ID
- display name
- execution policy
- runtime mode
- strategies
- risk defaults
- horizon label when available

- [ ] **Step 2: Add Swing option**

Add a visible Swing Trading option separate from Manual Trading and existing ETF news paper flow. Copy should say:

```text
Swing Trading researches multi-day ETF setups. It creates approval-gated paper candidates only after evidence, chart history, and daily revalidation checks pass.
```

- [ ] **Step 3: Verify**

Run:

```powershell
npm test -- --run src
npm run typecheck
```

## Task 6: Add Daily Revalidation

**Files:**

- Modify or add scheduler under `cmd/trader`
- Modify candidate metadata/status updates
- Add tests

- [ ] **Step 1: Write failing scheduler tests**

Required outcomes:

- Open swing candidate gets daily revalidation due.
- Failed invalidator marks candidate blocked or review-required.
- No broker instruction is created by revalidation.

- [ ] **Step 2: Implement scheduler**

Run only in paper runtime. Emit audit events for each decision.

- [ ] **Step 3: Verify**

Run:

```powershell
go test ./cmd/trader -run "Test.*Swing|Test.*Revalidation"
```

## Task 7: End-To-End Safety Verification

**Files:**

- Add run report under `Docs/runs/`
- Update Swing plan checklist

- [ ] **Step 1: Run backend checks**

```powershell
go test ./internal/modules/tradingmodes ./internal/modules/candidates ./cmd/trader
```

- [ ] **Step 2: Run frontend checks**

```powershell
npm test -- --run src
npm run typecheck
```

- [ ] **Step 3: Manual smoke test**

Verify:

- Swing mode appears as a separate option.
- Swing candidate cannot become an order without approval.
- Daily revalidation status is visible.
- Manual Trading still behaves as before.
- Existing intraday ETF approval workflow still behaves as before.

## Completion Criteria

- Swing is visible as a separate trading option.
- Swing candidates are paper-only and approval-gated.
- Swing evidence includes source links, event IDs, timestamps, candles, confounders, and blocker reasons.
- No live trading path is added.
- No manual ETF shortcut is added.
- Existing intraday workflow remains intact.
- Tests pass and a run report is recorded.

