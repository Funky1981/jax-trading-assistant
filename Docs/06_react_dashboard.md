
# 06 – React Dashboard (Jax UI)

Goal: build a **React + TypeScript** dashboard that speaks to the Jax Core
HTTP API and gives you a clear, trader‑friendly view of:

- Events and trade setups.
- Risk metrics.
- Dexter research.
- Backtests and strategies.
- Journals and logs.

This spec focuses on structure and API usage rather than pixel‑perfect design.

---

## 1. Frontend Project Layout

Create a separate `frontend` folder:

```text
jax-trading-assistant/
  frontend/
    package.json
    tsconfig.json
    vite.config.ts (or CRA / Next, your choice)
    src/
      main.tsx
      App.tsx
      api/
        events.ts
        trades.ts
        risk.ts
        backtests.ts
        strategies.ts
      components/
        EventCard.tsx
        TradeSetupCard.tsx
        ResearchPanel.tsx
        RiskForm.tsx
        BacktestTable.tsx
      pages/
        OverviewPage.tsx
        EventsPage.tsx
        TradesPage.tsx
        TradeDetailPage.tsx
        BacktestsPage.tsx
        BacktestDetailPage.tsx
        StrategiesPage.tsx
        JournalPage.tsx
      hooks/
        useEvents.ts
        useTrades.ts
        useBacktests.ts
```

Use React Router + React Query (or similar) for routing and data fetching.

---

## 2. API Contracts (Backend)

Jax Core should expose endpoints compatible with these expectations:

- `GET /events?symbol=&type=&limit=`
- `GET /trades?symbol=&strategyId=&limit=`
- `GET /trades/{id}`
- `POST /symbols/{symbol}/process`
- `POST /risk/calc`
- `POST /backtest/run`
- `GET /backtest/runs`
- `GET /backtest/runs/{id}`
- `GET /strategies`

Shape examples:

### 2.1 Trade (list view)

```json
{
  "id": "ts_001",
  "symbol": "AAPL",
  "direction": "long",
  "entry": 100.0,
  "stop": 95.0,
  "targets": [110.0, 115.0],
  "strategyId": "earnings_gap_v1",
  "event": {
    "type": "earnings_surprise",
    "time": "2025-01-25T14:30:00Z",
    "payload": {
      "surprisePct": 18.0
    }
  },
  "risk": {
    "positionSize": 60,
    "totalRisk": 300.0,
    "rMultiple": 2.0
  }
}
```

### 2.2 Trade (detail view)

Same as list, but can include `research` field from Dexter:

```json
{
  "research": {
    "summary": "...",
    "key_points": ["...", "..."],
    "metrics": { "pe_ratio": 28.5 }
  }
}
```

---

## 3. API Modules (frontend/src/api)

Each file encapsulates calls to one area of the backend.

Example `api/trades.ts`:

```ts
export interface TradeSummary {
  id: string;
  symbol: string;
  direction: "long" | "short";
  entry: number;
  stop: number;
  targets: number[];
  strategyId: string;
}

export interface RiskResult {
  positionSize: number;
  totalRisk: number;
  rMultiple: number;
}

export interface ResearchBundle {
  summary: string;
  key_points: string[];
  metrics: Record<string, number>;
}

export interface TradeDetail extends TradeSummary {
  event: any;
  risk: RiskResult;
  research?: ResearchBundle;
}

export async function fetchTrades(params: {
  symbol?: string;
  strategyId?: string;
  limit?: number;
}): Promise<TradeSummary[]> {
  const query = new URLSearchParams();
  if (params.symbol) query.set("symbol", params.symbol);
  if (params.strategyId) query.set("strategyId", params.strategyId);
  if (params.limit) query.set("limit", String(params.limit));
  const res = await fetch(`/api/trades?${query.toString()}`);
  if (!res.ok) throw new Error("Failed to fetch trades");
  return res.json();
}

export async function fetchTradeDetail(id: string): Promise<TradeDetail> {
  const res = await fetch(`/api/trades/${id}`);
  if (!res.ok) throw new Error("Failed to fetch trade");
  return res.json();
}
```

Codex should implement similar modules for events, risk, backtests, and
strategies.

---

## 4. Core Components

### 4.1 EventCard

Props:

- Symbol
- Event type
- Time
- Key numbers (gap %, surprise %, etc.)

Used on `/events` page and possibly in `/trades/:id` as context.

### 4.2 TradeSetupCard

Props:

- Symbol
- Direction
- Entry / Stop / Targets
- Strategy ID
- Risk summary

Rendered in a list on `/trades`, clicking opens `/trades/:id`.

### 4.3 ResearchPanel

Props:

- `research?: ResearchBundle`

Renders:

- Summary text.
- Bullet list of key points.
- Optional metrics table.

### 4.4 RiskForm

Interactive calculator:

- Inputs: account size, entry, stop, risk%.
- Calls `/risk/calc` on submit.
- Displays position size + R multiple.

### 4.5 BacktestTable

Displays a list of backtest runs with sortable columns:

- Strategy ID.
- Date range.
- Win rate.
- Avg R.
- Max drawdown.
- Sharpe.

---

## 5. Pages Overview

### 5.1 OverviewPage

Show a summary:

- Count of open / recent trades.
- List of most recent events.
- List of latest backtest runs.

### 5.2 EventsPage

- Table or cards of recent events.
- Filters: symbol, event type, date range.
- Each row links to trades generated from that event (if any).

### 5.3 TradesPage

- Table or cards of trade setups.
- Filters: symbol, strategy, direction.
- Each row expandable to show:
  - Event snippet.
  - Risk summary.

### 5.4 TradeDetailPage

- Trade details.
- Chart (later) showing recent price action around the event.
- ResearchPanel for Dexter output.
- Buttons for:
  - Mark as “journalled”.
  - Export to markdown (optional).
  - (Future) send to broker sim.

### 5.5 BacktestsPage / BacktestDetailPage

- List backtest runs, filter by strategy/date range.
- Detail page shows stats, maybe chart of equity curve later.

### 5.6 StrategiesPage

- List active strategies loaded from `config/strategies/` via backend.
- Show core parameters (name, type, min RR, risk%).
- Mark which strategies come from Lab (Agent0).

### 5.7 JournalPage

- Simple list of trades marked as journalled.
- Shows notes, Dexter summary, outcome if later added.

---

## 6. Data Fetching and Polling

Use React Query (or equivalent) for:

- `useEvents` – poll every 30–60 seconds.  
- `useTrades` – manual refetch or light polling (e.g. 30–60 seconds).  
- `useBacktests` – manual fetch only.  

Basic hook example in `hooks/useTrades.ts`:

```ts
import { useQuery } from "@tanstack/react-query";
import { fetchTrades, TradeSummary } from "../api/trades";

export function useTrades(filters: { symbol?: string; strategyId?: string }) {
  return useQuery<TradeSummary[]>({
    queryKey: ["trades", filters],
    queryFn: () => fetchTrades(filters),
    refetchInterval: 30000, // 30s
  });
}
```

---

## 7. Tasks for Codex / AI

1. Scaffold the `frontend` React + TypeScript app (Vite or similar).  
2. Install dependencies:
   - React Router.
   - React Query (or similar).
   - A UI library (e.g. Material UI).

3. Implement `api/*` modules that call the backend according to this spec.  
4. Implement core components:
   - EventCard
   - TradeSetupCard
   - ResearchPanel
   - RiskForm
   - BacktestTable

5. Implement pages and wire routing:
   - OverviewPage
   - EventsPage
   - TradesPage
   - TradeDetailPage
   - BacktestsPage
   - BacktestDetailPage
   - StrategiesPage
   - JournalPage

6. Add basic layout (header, sidebar/nav, main content).  
7. Implement light polling for events and trades.  
8. Add loading + error states for API calls (simple but visible).  
9. Confirm everything works against a stubbed or partially implemented Jax Core
   backend, then iterate.
