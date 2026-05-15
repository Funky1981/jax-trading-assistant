# ETF Trading Modes and News Strategy Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a user-friendly trading mode and strategy-selection system, integrate the ETF news strategy pack, and enable Jax to scan automatically for paper-trading candidates with notifications and approval-gated paper execution.

**Architecture:** Add a trading-mode catalog above strategy instances, then bridge the live watcher to the existing `libs/strategytypes` framework so mode-selected strategies can run against candles, news, and event context. ETF news strategies should emit normal Jax candidates, reuse existing ETF/risk/approval/execution gates, and publish notifications when a trade candidate is qualified.

**Tech Stack:** Go backend in `cmd/trader`, `internal/modules/*`, `internal/trader/*`, and `libs/strategytypes`; React/Vite frontend in `frontend/src`; PostgreSQL-backed event/candidate/strategy instance tables; existing IB bridge and ETF instrument catalog.

---

## Scope

This plan extends the existing ETF readiness work. It does not enable live trading, options, futures, leveraged ETFs, inverse ETFs, volatility products, or autonomous live broker execution.

The first working target is:

- operator picks a trading mode in the UI
- operator picks a strategy inside that mode
- operator configures an instance without editing raw JSON
- Jax scans automatically while the instance is enabled
- ETF news strategies generate candidate trades
- hard ETF and risk gates qualify or block candidates
- Jax sends an in-app notification when a qualified candidate appears
- human approval remains required before the paper broker path submits an order

Auto paper-submit can be added only after this flow has passing evidence. It must be controlled by a disabled-by-default flag and must remain paper-only.

## Existing Strategy Inventory To Preserve

Runtime artifact strategies in `libs/strategies`:

- `rsi_momentum_v1`
- `macd_crossover_v1`
- `ma_crossover_v1`

Strategy type framework in `libs/strategytypes`:

- `same_day_earnings_drift_v1`
- `same_day_news_repricing_v1`
- `news_shock_momentum_v1`
- `opening_range_to_close_v1`
- `event_gap_continuation_v1`
- `panic_reversion_v1`
- `pairs_event_relative_v1`
- `index_flow_v1`

ETF strategy pack to add:

- `etf_news_market_panic_reversal_v1`
- `etf_news_sector_momentum_v1`
- `etf_news_rates_bonds_rotation_v1`

## File Structure

Create:

- `internal/modules/tradingmodes/catalog.go` - static mode catalog and default risk/data requirements.
- `internal/modules/tradingmodes/catalog_test.go` - catalog tests.
- `cmd/trader/trading_modes_handlers.go` - `/api/v1/trading-modes` HTTP handlers.
- `cmd/trader/trading_modes_handlers_test.go` - handler tests.
- `internal/modules/etfnews/classifier.go` - ETF-specific news classification and ETF mapping helpers.
- `internal/modules/etfnews/classifier_test.go` - classifier and mapping tests.
- `libs/strategytypes/strategy_etf_news_common.go` - shared ETF strategy helpers.
- `libs/strategytypes/strategy_etf_news_market_panic.go` - market panic reversal strategy.
- `libs/strategytypes/strategy_etf_news_sector_momentum.go` - sector momentum strategy.
- `libs/strategytypes/strategy_etf_news_rates_rotation.go` - rates/bonds rotation strategy.
- `internal/trader/strategytypegenerator/generator.go` - live generator for `libs/strategytypes`.
- `internal/trader/strategytypegenerator/generator_test.go` - generator tests.
- `internal/modules/notifications/service.go` - notification fanout service.
- `internal/modules/notifications/service_test.go` - notification tests.
- `cmd/trader/notification_handlers.go` - notification preference and test-send endpoints.
- `frontend/src/data/trading-modes-service.ts` - frontend data client.
- `frontend/src/data/notifications-service.ts` - frontend notification client.
- `frontend/src/hooks/useLiveEvents.ts` - SSE subscription hook.
- `frontend/src/pages/TradingModesPage.tsx` - mode and strategy picker page.
- `frontend/src/pages/TradingModesPage.test.tsx` - UI tests.
- `frontend/e2e/trading-modes.spec.ts` - mode picker e2e coverage.
- `config/strategy-instances/etf-news-market-panic-paper-v1.json` - disabled seed instance.
- `config/strategy-instances/etf-news-sector-momentum-paper-v1.json` - disabled seed instance.
- `config/strategy-instances/etf-news-rates-rotation-paper-v1.json` - disabled seed instance.

Modify:

- `cmd/trader/frontend_api.go` - register trading modes and notification routes.
- `cmd/trader/codex_api.go` - keep strategy instance validation compatible with mode-created instances.
- `cmd/trader/event_classification.go` - call ETF classifier for ETF-relevant event classes.
- `cmd/trader/instance_scheduler.go` - run strategy-type generator for strategy type instances.
- `cmd/trader/trade_watcher.go` - wire generator dependencies.
- `libs/strategytypes/registry.go` - register ETF strategy types.
- `libs/strategytypes/types.go` - add optional news attributes without breaking existing callers.
- `internal/modules/candidates/service.go` - preserve richer ETF metadata on candidates.
- `frontend/src/data/types.ts` - correct strategy type metadata shape and add trading mode types.
- `frontend/src/data/instances-service.ts` - map backend `strategyId` field correctly.
- `frontend/src/app/App.tsx` - add `/trading-modes` route.
- `frontend/src/components/layout/AppShell.tsx` - add navigation entry.
- `Docs/UAT_PAPER_TRADING.md` - add ETF news mode UAT.
- `Docs/PRODUCTION_READINESS.md` - add provider and notification readiness notes.
- `scripts/etf-paper-pilot-evidence.ps1` - capture mode, strategy, notification, and candidate evidence.

Adjacent risky areas to avoid unless a task explicitly requires them:

- `Agent0/`
- `dexter/`
- live trading controls
- generated build outputs
- historical docs in `Docs/archive/`

---

### Task 1: Add Backend Trading Mode Catalog

**Files:**

- Create: `internal/modules/tradingmodes/catalog.go`
- Create: `internal/modules/tradingmodes/catalog_test.go`
- Create: `cmd/trader/trading_modes_handlers.go`
- Create: `cmd/trader/trading_modes_handlers_test.go`
- Modify: `cmd/trader/frontend_api.go`

- [ ] **Step 1: Write catalog tests**

Create `internal/modules/tradingmodes/catalog_test.go` with tests that assert:

```go
func TestDefaultCatalogIncludesETFNewsPaperMode(t *testing.T) {
	catalog := DefaultCatalog()
	mode, ok := catalog.Get("etf_news_paper")
	if !ok {
		t.Fatalf("expected etf_news_paper mode")
	}
	if mode.AssetClass != "ETF" {
		t.Fatalf("asset class = %q, want ETF", mode.AssetClass)
	}
	if mode.ExecutionPolicy != "candidate_approval_only" {
		t.Fatalf("execution policy = %q", mode.ExecutionPolicy)
	}
	want := []string{
		"etf_news_market_panic_reversal_v1",
		"etf_news_sector_momentum_v1",
		"etf_news_rates_bonds_rotation_v1",
	}
	got := make(map[string]bool)
	for _, strategy := range mode.Strategies {
		got[strategy.StrategyTypeID] = true
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("missing strategy %s", id)
		}
	}
}
```

- [ ] **Step 2: Run the failing catalog test**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./internal/modules/tradingmodes
```

Expected: fail because `internal/modules/tradingmodes` does not exist.

- [ ] **Step 3: Implement the catalog**

Create `internal/modules/tradingmodes/catalog.go` with:

```go
package tradingmodes

type Catalog struct {
	Modes []Mode `json:"modes"`
}

type Mode struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	AssetClass      string        `json:"assetClass"`
	RuntimeMode     string        `json:"runtimeMode"`
	ExecutionPolicy string        `json:"executionPolicy"`
	Universe        []string      `json:"universe"`
	RequiredData    []string      `json:"requiredData"`
	RiskDefaults    RiskDefaults  `json:"riskDefaults"`
	Strategies      []StrategyRef `json:"strategies"`
}

type StrategyRef struct {
	StrategyTypeID string         `json:"strategyTypeId"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	DefaultConfig  map[string]any `json:"defaultConfig"`
}

type RiskDefaults struct {
	MaxTradesPerDay int     `json:"maxTradesPerDay"`
	MaxOpenPositions int    `json:"maxOpenPositions"`
	RiskPerTradePct float64 `json:"riskPerTradePct"`
	MinConfidence   float64 `json:"minConfidence"`
	FlattenBy       string  `json:"flattenBy"`
	ApprovalRequired bool   `json:"approvalRequired"`
}

func DefaultCatalog() Catalog {
	etfUniverse := []string{"SPY", "QQQ", "DIA", "IWM", "XLK", "XLF", "XLE", "SMH", "SOXX", "TLT", "GLD"}
	return Catalog{Modes: []Mode{
		{
			ID:              "etf_news_paper",
			Name:            "ETF News Paper",
			Description:     "Paper-only ETF strategies driven by confirmed news, macro context, chart structure, volatility, and ETF guardrails.",
			AssetClass:      "ETF",
			RuntimeMode:     "paper",
			ExecutionPolicy: "candidate_approval_only",
			Universe:        etfUniverse,
			RequiredData:    []string{"quotes", "candles_1m", "candles_5m", "news", "event_classification"},
			RiskDefaults: RiskDefaults{
				MaxTradesPerDay: 3, MaxOpenPositions: 1, RiskPerTradePct: 0.25,
				MinConfidence: 0.65, FlattenBy: "15:55", ApprovalRequired: true,
			},
			Strategies: []StrategyRef{
				{StrategyTypeID: "etf_news_market_panic_reversal_v1", Name: "Market Panic Reversal", Description: "Looks for broad ETF rebound setups after panic news and price stabilization.", DefaultConfig: map[string]any{"symbols": []string{"SPY", "QQQ", "DIA", "IWM"}, "parameters": map[string]any{"minDropPct": 1.2, "stabilizationBars": 3, "atrStopMultiple": 1.1}}},
				{StrategyTypeID: "etf_news_sector_momentum_v1", Name: "Sector News Momentum", Description: "Maps confirmed sector news into plain ETF momentum candidates.", DefaultConfig: map[string]any{"symbols": []string{"QQQ", "XLK", "SMH", "SOXX", "XLE", "XLF", "IWM", "GLD"}, "parameters": map[string]any{"minConfirmations": 2, "minMovePct": 0.4, "minVolumeMultiple": 1.2}}},
				{StrategyTypeID: "etf_news_rates_bonds_rotation_v1", Name: "Rates and Bonds Rotation", Description: "Evaluates rates and inflation news for TLT, GLD, SPY, QQQ, and XLF candidates.", DefaultConfig: map[string]any{"symbols": []string{"TLT", "GLD", "SPY", "QQQ", "XLF"}, "parameters": map[string]any{"minConfirmations": 2, "minMovePct": 0.35, "atrStopMultiple": 1.0}}},
			},
		},
		{
			ID: "research_only", Name: "Research Only", Description: "Backtest and review strategies without live scanning or broker submission.",
			AssetClass: "MULTI", RuntimeMode: "research", ExecutionPolicy: "no_execution",
			Universe: nil, RequiredData: []string{"datasets"}, RiskDefaults: RiskDefaults{ApprovalRequired: true},
			Strategies: nil,
		},
	}}
}

func (c Catalog) Get(id string) (Mode, bool) {
	for _, mode := range c.Modes {
		if mode.ID == id {
			return mode, true
		}
	}
	return Mode{}, false
}
```

- [ ] **Step 4: Add HTTP handler and route**

Create `cmd/trader/trading_modes_handlers.go`:

```go
package main

import (
	"net/http"
	"strings"

	"jax-trading-assistant/internal/modules/tradingmodes"
)

func tradingModesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonOK(w, tradingmodes.DefaultCatalog())
	}
}

func tradingModeDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/trading-modes/"), "/")
		mode, ok := tradingmodes.DefaultCatalog().Get(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		jsonOK(w, mode)
	}
}
```

Modify `cmd/trader/frontend_api.go` near the strategy route block:

```go
mux.HandleFunc("/api/v1/trading-modes", protect(tradingModesHandler()))
mux.HandleFunc("/api/v1/trading-modes/", protect(tradingModeDetailHandler()))
```

- [ ] **Step 5: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./internal/modules/tradingmodes ./cmd/trader
```

Expected: pass.

---

### Task 2: Fix Frontend Strategy Type Contracts and Add Trading Mode Client

**Files:**

- Modify: `frontend/src/data/types.ts`
- Modify: `frontend/src/data/instances-service.ts`
- Create: `frontend/src/data/trading-modes-service.ts`
- Test: affected frontend unit tests under `frontend/src`

- [ ] **Step 1: Update TypeScript types**

Replace `StrategyTypeMetadata` in `frontend/src/data/types.ts` with:

```ts
export interface StrategyTypeRequiredInputs {
  candles: string[];
  needsEarnings: boolean;
  needsNews: boolean;
}

export interface StrategyTypeParameter {
  key: string;
  type: 'int' | 'float' | 'string' | 'bool' | string;
  default: unknown;
  min?: number;
  max?: number;
  description?: string;
}

export interface StrategyTypeMetadata {
  strategyId: string;
  name: string;
  description: string;
  requiredInputs: StrategyTypeRequiredInputs;
  parameters: StrategyTypeParameter[];
}

export interface TradingModeStrategy {
  strategyTypeId: string;
  name: string;
  description: string;
  defaultConfig: Record<string, unknown>;
}

export interface TradingModeRiskDefaults {
  maxTradesPerDay: number;
  maxOpenPositions: number;
  riskPerTradePct: number;
  minConfidence: number;
  flattenBy: string;
  approvalRequired: boolean;
}

export interface TradingMode {
  id: string;
  name: string;
  description: string;
  assetClass: string;
  runtimeMode: string;
  executionPolicy: string;
  universe: string[];
  requiredData: string[];
  riskDefaults: TradingModeRiskDefaults;
  strategies: TradingModeStrategy[];
}

export interface TradingModeCatalog {
  modes: TradingMode[];
}
```

- [ ] **Step 2: Add mode service**

Create `frontend/src/data/trading-modes-service.ts`:

```ts
import { apiClient } from './http-client';
import type { TradingMode, TradingModeCatalog } from './types';

export const tradingModesService = {
  async list(): Promise<TradingMode[]> {
    const catalog = await apiClient.get<TradingModeCatalog>('/api/v1/trading-modes');
    return catalog.modes ?? [];
  },

  async get(id: string): Promise<TradingMode> {
    return apiClient.get<TradingMode>(`/api/v1/trading-modes/${encodeURIComponent(id)}`);
  },
};
```

- [ ] **Step 3: Keep strategy type list compatible**

In `frontend/src/data/instances-service.ts`, keep `listStrategyTypes()` returning backend rows directly, but update UI call sites to use `strategy.strategyId`, not `strategy.id`.

Search:

```powershell
rg "strategyTypes|strategy\\.id|strategyType\\.id|StrategyTypeMetadata" frontend/src -n
```

Update each dropdown option value to use `strategy.strategyId`.

- [ ] **Step 4: Verify**

Run:

```powershell
Set-Location frontend
npm test -- ResearchPage
npm run typecheck
```

Expected: tests and typecheck pass.

---

### Task 3: Add User-Friendly Trading Modes UI

**Files:**

- Create: `frontend/src/pages/TradingModesPage.tsx`
- Create: `frontend/src/pages/TradingModesPage.test.tsx`
- Modify: `frontend/src/app/App.tsx`
- Modify: `frontend/src/components/layout/AppShell.tsx`
- Test: `frontend/e2e/trading-modes.spec.ts`

- [ ] **Step 1: Write UI behavior test**

Create `frontend/src/pages/TradingModesPage.test.tsx` with a mocked `tradingModesService.list()` that returns `ETF News Paper`. Assert the page renders:

```ts
expect(await screen.findByText('ETF News Paper')).toBeInTheDocument();
expect(screen.getByText('Market Panic Reversal')).toBeInTheDocument();
expect(screen.getByRole('button', { name: /create disabled paper instance/i })).toBeInTheDocument();
```

- [ ] **Step 2: Build the page**

Create a page that:

- shows mode cards
- shows strategies for the selected mode
- shows allowed ETF universe as compact symbol chips
- shows required data and execution policy
- creates a disabled strategy instance from the selected strategy default config
- never enables an instance on creation

The create action should call:

```ts
instancesService.create({
  name: `${selectedStrategy.strategyTypeId}-paper`,
  strategyTypeId: selectedStrategy.strategyTypeId,
  enabled: false,
  sessionTimezone: 'America/New_York',
  flattenByCloseTime: selectedMode.riskDefaults.flattenBy,
  configJson: selectedStrategy.defaultConfig,
});
```

The button label must be `Create disabled paper instance` so the operator understands this does not start scanning immediately.

- [ ] **Step 3: Add route and navigation**

Modify `frontend/src/app/App.tsx`:

```ts
import { TradingModesPage } from '@/pages/TradingModesPage';
```

Add route:

```ts
{ path: 'trading-modes', element: <TradingModesPage /> },
```

Modify `frontend/src/components/layout/AppShell.tsx`:

```ts
import { ListChecks } from 'lucide-react';
```

Add nav item near Trading:

```ts
{ label: 'Trading Modes', path: '/trading-modes', icon: ListChecks },
```

- [ ] **Step 4: Verify**

Run:

```powershell
Set-Location frontend
npm test -- TradingModesPage AppRoutes
npm run typecheck
npm run lint
```

Expected: pass.

---

### Task 4: Add ETF News Classification and Symbol Mapping

**Files:**

- Create: `internal/modules/etfnews/classifier.go`
- Create: `internal/modules/etfnews/classifier_test.go`
- Modify: `cmd/trader/event_classification.go`

- [ ] **Step 1: Write classifier tests**

Test the following cases:

- "market panic" text maps to class `market_panic`, sentiment `bearish`, and symbols `SPY`, `QQQ`, `DIA`, `IWM`
- semiconductor positive text maps to class `sector_news`, sector `semiconductors`, and symbols `SMH`, `SOXX`
- inflation or yields rising maps to class `rates_inflation`, symbols `TLT`, `GLD`, `SPY`, `QQQ`, `XLF`
- unrelated single-stock news stays outside ETF-specific classification

- [ ] **Step 2: Implement classifier**

Create `internal/modules/etfnews/classifier.go` with:

```go
package etfnews

import "strings"

type Classification struct {
	Class       string   `json:"class"`
	Sector      string   `json:"sector,omitempty"`
	Sentiment   string   `json:"sentiment"`
	Materiality string   `json:"materiality"`
	Symbols     []string `json:"symbols"`
	Tags        []string `json:"tags"`
}

func Classify(title, summary string) (Classification, bool) {
	text := strings.ToLower(strings.TrimSpace(title + " " + summary))
	switch {
	case containsAny(text, "panic", "selloff", "risk-off", "crash", "market rout"):
		return Classification{Class: "market_panic", Sentiment: "bearish", Materiality: "high", Symbols: []string{"SPY", "QQQ", "DIA", "IWM"}, Tags: []string{"broad_market", "risk_off"}}, true
	case containsAny(text, "semiconductor", "chip", "chips", "ai accelerator", "foundry"):
		return Classification{Class: "sector_news", Sector: "semiconductors", Sentiment: sentimentFromText(text), Materiality: "medium", Symbols: []string{"SMH", "SOXX"}, Tags: []string{"sector", "semiconductors"}}, true
	case containsAny(text, "inflation", "cpi", "pce", "treasury yield", "rate cut", "rate hike", "federal reserve", "fed"):
		return Classification{Class: "rates_inflation", Sentiment: sentimentFromText(text), Materiality: "high", Symbols: []string{"TLT", "GLD", "SPY", "QQQ", "XLF"}, Tags: []string{"macro", "rates"}}, true
	case containsAny(text, "oil", "crude", "opec", "energy supply"):
		return Classification{Class: "sector_news", Sector: "energy", Sentiment: sentimentFromText(text), Materiality: "medium", Symbols: []string{"XLE"}, Tags: []string{"sector", "energy"}}, true
	case containsAny(text, "bank", "banks", "credit stress", "regional lender"):
		return Classification{Class: "sector_news", Sector: "financials", Sentiment: sentimentFromText(text), Materiality: "medium", Symbols: []string{"XLF"}, Tags: []string{"sector", "financials"}}, true
	default:
		return Classification{}, false
	}
}

func sentimentFromText(text string) string {
	switch {
	case containsAny(text, "beat", "surge", "rally", "strong", "approved", "cut rates", "rate cut"):
		return "positive"
	case containsAny(text, "miss", "weak", "falls", "drops", "higher yields", "rate hike", "inflation hot"):
		return "negative"
	default:
		return "neutral"
	}
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Bridge classifier into existing event classification**

In `cmd/trader/event_classification.go`, call `etfnews.Classify` before returning the generic class. Add ETF tags and symbols into `Attributes` only when the classifier returns true. Preserve the existing generic behavior for non-ETF events.

- [ ] **Step 4: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./internal/modules/etfnews ./cmd/trader
```

Expected: pass.

---

### Task 5: Add ETF News Strategy Types

**Files:**

- Create: `libs/strategytypes/strategy_etf_news_common.go`
- Create: `libs/strategytypes/strategy_etf_news_market_panic.go`
- Create: `libs/strategytypes/strategy_etf_news_sector_momentum.go`
- Create: `libs/strategytypes/strategy_etf_news_rates_rotation.go`
- Modify: `libs/strategytypes/registry.go`
- Modify: `libs/strategytypes/registry_test.go`

- [ ] **Step 1: Extend strategy tests**

Add test cases that build deterministic `StrategyInput` with 1m and 5m candles plus one or more `NewsEvent` rows:

- market panic reversal emits `BUY` after a drawdown and stabilization
- market panic reversal emits no signal without stabilization
- sector momentum emits `BUY` for positive semiconductor news with volume confirmation
- rates rotation emits `BUY` for TLT when rates/inflation text implies falling yields or risk-off bonds
- each strategy returns no signal when `symbols` or news category does not match the strategy

- [ ] **Step 2: Add shared helpers**

Create `libs/strategytypes/strategy_etf_news_common.go` with helper functions for:

- `latestCandle(input StrategyInput, preferred ...string) (Candle, []Candle, error)`
- `confirmedNews(input StrategyInput, categories ...string) []NewsEvent`
- `volumeMultiple(candles []Candle, lookback int) float64`
- `drawdownPct(candles []Candle, lookback int) float64`
- `atr(candles []Candle, lookback int) float64`
- `etfSignal(strategyID, symbol, direction, reason string, entry Candle, stopDistance, rewardDistance float64) Signal`

- [ ] **Step 3: Implement strategy metadata and validation**

Each strategy must declare:

```go
RequiredInputs: RequiredInputs{
	Candles: []string{"1m", "5m"},
	NeedsNews: true,
}
```

Use these parameter keys:

- `minConfirmations`
- `minMovePct`
- `minVolumeMultiple`
- `stabilizationBars`
- `atrStopMultiple`
- `rewardRiskMultiple`

Validation must enforce bounded ranges using existing `requireRangeFloat` and `requireRangeInt` helpers.

- [ ] **Step 4: Register strategies**

Modify `libs/strategytypes/registry.go`:

```go
_ = r.Register(NewETFNewsMarketPanicReversal())
_ = r.Register(NewETFNewsSectorMomentum())
_ = r.Register(NewETFNewsRatesBondsRotation())
```

- [ ] **Step 5: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./libs/strategytypes
```

Expected: pass.

---

### Task 6: Bridge Live Watcher to `libs/strategytypes`

**Files:**

- Create: `internal/trader/strategytypegenerator/generator.go`
- Create: `internal/trader/strategytypegenerator/generator_test.go`
- Modify: `cmd/trader/instance_scheduler.go`
- Modify: `cmd/trader/trade_watcher.go`

- [ ] **Step 1: Define generator contract**

Create a generator method:

```go
func (g *Generator) GenerateForInstance(ctx context.Context, inst Instance) ([]domain.Signal, error)
```

The `Instance` type must contain:

```go
type Instance struct {
	ID                 uuid.UUID
	Name               string
	StrategyTypeID     string
	SessionTimezone    string
	FlattenByCloseTime string
	Symbols            []string
	Parameters         map[string]any
}
```

- [ ] **Step 2: Load data from existing DB tables**

The generator must:

- read recent `candles` for 1m and 5m if present
- read `event_normalized` and `event_symbol_map` rows for the same symbol and current session
- convert rows to `strategytypes.NewsEvent`
- call exactly the configured strategy type, not every strategy in the registry
- convert `strategytypes.Signal` to `domain.Signal`
- store the signal in `strategy_signals` with `status='pending'`

Conversion rule:

```go
domain.Signal{
	ID:         uuid.NewString(),
	Symbol:     out.Symbol,
	Timestamp:  out.GeneratedAt,
	Type:       strings.ToLower(out.Direction),
	Confidence: confidenceFromReason(out.Reason),
	EntryPrice: out.EntryPrice,
	StopLoss:   out.StopLoss,
	TakeProfit: []float64{out.TakeProfit},
	Reason:     out.Reason,
	StrategyID: out.StrategyID,
}
```

Use `0.70` as the default confidence when the strategy does not provide a numeric confidence.

- [ ] **Step 3: Update scheduler without breaking legacy strategies**

In `cmd/trader/instance_scheduler.go`, keep legacy `signalgenerator.InProcessSignalGenerator` for legacy strategies and add a strategy-type path for any instance whose `StrategyTypeID` exists in `strategytypes.DefaultRegistry()`.

The routing rule:

```go
if strategyTypeGenerator.CanRun(inst.StrategyTypeID) {
	signals, err = strategyTypeGenerator.GenerateForInstance(ctx, toStrategyTypeInstance(inst))
} else {
	signals, err = legacyGenerator.GenerateSignals(ctx, inst.Symbols)
}
```

- [ ] **Step 4: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./internal/trader/strategytypegenerator ./cmd/trader
```

Expected: pass.

---

### Task 7: Preserve Rich ETF Candidate Metadata and Audit Trail

**Files:**

- Modify: `internal/modules/candidates/service.go`
- Test: `internal/modules/candidates/service_test.go`
- Test: `internal/modules/candidates/etf_policy_test.go`

- [ ] **Step 1: Add metadata fields to proposal/block requests**

Extend `ProposalRequest` and `BlockRequest` with:

```go
Metadata map[string]any
```

When creating candidates, merge this metadata before applying `metadataWithETFResult`.

- [ ] **Step 2: Store ETF decision context**

The strategy-type generator should pass metadata containing:

```go
map[string]any{
	"modeId": "etf_news_paper",
	"newsEventIds": []string{},
	"sourceConfirmations": 2,
	"sentimentSummary": "positive sector news confirmed by two sources",
	"volatilitySnapshot": map[string]any{"atr": 1.23, "volumeMultiple": 1.5},
	"drawdownContext": map[string]any{"lookbackBars": 30, "drawdownPct": -1.4},
	"invalidationReason": "price loses stabilization low or ETF spread exceeds policy",
	"rejectedAlternatives": []string{"SOXX rejected because SMH spread was tighter"},
}
```

Use real values where available and empty arrays for unavailable IDs. Do not invent source IDs.

- [ ] **Step 3: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./internal/modules/candidates ./internal/trader/strategytypegenerator
```

Expected: pass.

---

### Task 8: Add Notifications

**Files:**

- Create: `internal/modules/notifications/service.go`
- Create: `internal/modules/notifications/service_test.go`
- Create: `cmd/trader/notification_handlers.go`
- Modify: `cmd/trader/frontend_api.go`
- Modify: `cmd/trader/instance_scheduler.go`
- Create: `frontend/src/hooks/useLiveEvents.ts`
- Create: `frontend/src/data/notifications-service.ts`

- [ ] **Step 1: Implement notification service**

Create a service with in-app publish and optional webhook support:

```go
type Service struct {
	WebhookURL string
}

type Notification struct {
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Severity  string         `json:"severity"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

func (s Service) Publish(ctx context.Context, n Notification) error {
	publishEvent("notification.created", n)
	if strings.TrimSpace(s.WebhookURL) == "" {
		return nil
	}
	return s.postWebhook(ctx, n)
}
```

Read webhook URL from `JAX_NOTIFICATION_WEBHOOK_URL`.

- [ ] **Step 2: Publish on qualified candidates**

In `cmd/trader/instance_scheduler.go`, after `candidate.qualified`, publish:

```go
notificationSvc.Publish(ctx, notifications.Notification{
	Type: "candidate.qualified",
	Title: "ETF candidate ready for approval",
	Message: fmt.Sprintf("%s %s from %s", qualified.Symbol, signalType, inst.Name),
	Severity: "info",
	Payload: map[string]any{"candidateId": qualified.ID.String(), "symbol": qualified.Symbol},
	CreatedAt: time.Now().UTC(),
})
```

- [ ] **Step 3: Add frontend live events hook**

Create `frontend/src/hooks/useLiveEvents.ts`:

```ts
import { useEffect, useState } from 'react';

export interface LiveEvent {
  type: string;
  payload: unknown;
  sentAt: string;
}

export function useLiveEvents(eventTypes: string[]) {
  const [events, setEvents] = useState<LiveEvent[]>([]);

  useEffect(() => {
    const stream = new EventSource('/api/v1/events/stream');
    const listeners = eventTypes.map((type) => {
      const listener = (event: MessageEvent) => {
        setEvents((current) => [JSON.parse(event.data), ...current].slice(0, 20));
      };
      stream.addEventListener(type, listener);
      return { type, listener };
    });
    return () => {
      listeners.forEach(({ type, listener }) => stream.removeEventListener(type, listener));
      stream.close();
    };
  }, [eventTypes.join('|')]);

  return events;
}
```

- [ ] **Step 4: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./internal/modules/notifications ./cmd/trader
Set-Location frontend
npm run typecheck
```

Expected: pass.

---

### Task 9: Seed Disabled ETF News Strategy Instances

**Files:**

- Create: `config/strategy-instances/etf-news-market-panic-paper-v1.json`
- Create: `config/strategy-instances/etf-news-sector-momentum-paper-v1.json`
- Create: `config/strategy-instances/etf-news-rates-rotation-paper-v1.json`

- [ ] **Step 1: Add market panic instance**

Create:

```json
{
  "name": "etf-news-market-panic-paper-v1",
  "strategyTypeId": "etf_news_market_panic_reversal_v1",
  "enabled": false,
  "sessionTimezone": "America/New_York",
  "flattenByCloseTime": "15:55",
  "config": {
    "symbols": ["SPY", "QQQ", "DIA", "IWM"],
    "parameters": {
      "minConfirmations": 2,
      "minMovePct": 0.6,
      "minVolumeMultiple": 1.2,
      "stabilizationBars": 3,
      "atrStopMultiple": 1.1,
      "rewardRiskMultiple": 1.5
    }
  }
}
```

- [ ] **Step 2: Add sector momentum instance**

Use symbols `["QQQ", "XLK", "SMH", "SOXX", "XLE", "XLF", "IWM", "GLD"]` and the same paper-only defaults.

- [ ] **Step 3: Add rates rotation instance**

Use symbols `["TLT", "GLD", "SPY", "QQQ", "XLF"]` and the same paper-only defaults.

- [ ] **Step 4: Verify config validation**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./cmd/trader ./libs/strategytypes
```

Expected: pass.

---

### Task 10: Add Optional Auto Paper-Submit Gate

**Files:**

- Modify: `cmd/trader/instance_scheduler.go`
- Modify: `cmd/trader/readiness.go`
- Modify: `cmd/trader/readiness_runtime_test.go`
- Modify: `Docs/OPERATIONS.md`

- [ ] **Step 1: Keep default behavior approval-only**

The default must remain:

```powershell
ETF_AUTO_PAPER_SUBMIT=false
```

When false, candidates stop at `awaiting_approval` and only notify.

- [ ] **Step 2: Add readiness-gated auto paper submit**

Allow automatic paper submit only when all are true:

- `JAX_RUNTIME_MODE=paper`
- `ALLOW_LIVE_TRADING` is not `true`
- `EXECUTION_ENABLED=true`
- `ETF_AUTO_PAPER_SUBMIT=true`
- ETF phase-1 readiness flags are all passing
- candidate metadata mode is `etf_news_paper`
- candidate has stop-loss and flatten-by-close

If any condition fails, publish a notification with severity `warning` and leave the candidate awaiting approval.

- [ ] **Step 3: Verify**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode quick -Packages ./cmd/trader ./internal/modules/candidates ./internal/modules/execution
```

Expected: pass.

---

### Task 11: Add UAT and Pilot Evidence Coverage

**Files:**

- Modify: `Docs/UAT_PAPER_TRADING.md`
- Modify: `Docs/PRODUCTION_READINESS.md`
- Modify: `scripts/etf-paper-pilot-evidence.ps1`
- Create: `frontend/e2e/trading-modes.spec.ts`

- [ ] **Step 1: Extend UAT**

Add an ETF news mode UAT sequence:

1. Open Trading Modes.
2. Select ETF News Paper.
3. Create disabled Market Panic Reversal instance.
4. Confirm instance appears disabled.
5. Enable instance only after paper broker, provider, and ETF readiness pass.
6. Inject or replay a deterministic ETF event.
7. Confirm candidate appears in Approvals.
8. Confirm in-app notification appears.
9. Approve candidate.
10. Confirm paper order path is used.
11. Confirm exit/flatten evidence is recorded.

- [ ] **Step 2: Capture evidence**

Update `scripts/etf-paper-pilot-evidence.ps1` to capture:

- `/api/v1/trading-modes`
- `/api/v1/strategy-types`
- `/api/v1/instances`
- `/api/v1/candidates?status=awaiting_approval`
- latest `notification.created` event evidence if available
- ETF readiness summary

- [ ] **Step 3: Verify**

Run:

```powershell
.\scripts\etf-paper-pilot-evidence.ps1
Set-Location frontend
npm run test:e2e -- trading-modes.spec.ts
```

Expected: evidence artifact is created under `Docs/runs/etf-paper-pilot/` and e2e passes.

---

### Task 12: Full Regression and Launch Gate

**Files:**

- No new source files.
- Review generated evidence only.

- [ ] **Step 1: Backend focused verification**

Run:

```powershell
.\scripts\go-verify.ps1 -Mode standard -Packages ./internal/modules/tradingmodes ./internal/modules/etfnews ./internal/modules/notifications ./internal/trader/strategytypegenerator ./libs/strategytypes ./internal/modules/candidates ./cmd/trader
```

Expected: pass.

- [ ] **Step 2: Golden and replay verification**

Run:

```powershell
.\scripts\golden-check.ps1 -Mode verify
go test ./tests/replay/...
```

Expected: pass. If snapshots change, inspect the diff and document why the strategy-type bridge intentionally changed candidate output.

- [ ] **Step 3: Frontend verification**

Run:

```powershell
Set-Location frontend
npm run typecheck
npm run lint
npm test -- TradingModesPage ApprovalsPage AppRoutes
npm run test:e2e -- trading-modes.spec.ts trading.spec.ts
```

Expected: pass.

- [ ] **Step 4: Paper pilot smoke**

Run:

```powershell
.\scripts\test-platform.ps1
.\scripts\etf-paper-pilot-evidence.ps1
```

Expected: both scripts complete and produce dated run artifacts.

- [ ] **Step 5: Launch gate**

ETF news mode can be considered ready for controlled paper use only when:

- ETF catalog loads
- trading mode catalog lists `etf_news_paper`
- all three ETF news strategy types are registered
- enabled ETF news instance scans without runtime errors
- at least one deterministic replay produces a candidate
- stale quote, wide spread, excluded ETF, missing stop, outside session, max trades, and kill-switch tests all block
- notification is observed for qualified candidate
- paper order submission is approval-gated unless `ETF_AUTO_PAPER_SUBMIT=true` and every auto-submit readiness condition passes
- evidence artifact exists in `Docs/runs/etf-paper-pilot/`

---

## Implementation Order

1. Task 1 - backend trading mode catalog.
2. Task 2 - frontend contracts.
3. Task 3 - Trading Modes UI.
4. Task 4 - ETF news classification.
5. Task 5 - ETF strategy types.
6. Task 6 - live strategy-type bridge.
7. Task 7 - candidate metadata.
8. Task 8 - notifications.
9. Task 9 - disabled ETF news instances.
10. Task 10 - optional auto paper-submit gate.
11. Task 11 - UAT and evidence.
12. Task 12 - full regression and launch gate.

Do not start Task 10 until Tasks 1 through 9 are merged and verified. The approval-only flow is the first production-quality paper target.

## Self-Review

Spec coverage:

- Trading type and mode picker: Tasks 1, 2, and 3.
- ETF strategy pack integration: Tasks 4, 5, 6, 7, and 9.
- Automatic scanning: Task 6.
- Notification on trade opportunity: Task 8.
- Paper execution path: existing approval and execution modules reused by Tasks 6, 7, 10, and 12.
- News, charts, sentiment, volatility, and drawdown context: Tasks 4, 5, 6, and 7.
- Guardrails and validation: Tasks 10, 11, and 12.

Placeholder scan:

- This plan contains no deferred implementation placeholders. Each task names files, behavior, and verification commands.

Type consistency:

- Backend strategy type IDs use `strategyTypeId` in API payloads and `StrategyTypeID` in Go structs.
- Frontend strategy metadata uses backend field `strategyId`, not the previous `id` assumption.
- Candidate notification events use existing SSE event type format and the new `notification.created` event type.

