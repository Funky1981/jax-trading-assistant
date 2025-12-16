
# 01 – System Overview (Jax Trading Assistant)

## 1. Project Name and Purpose

**Working name:** Jax Trading Assistant  
(You can rename the product later without changing the internals.)

Jax is a **modular, event‑driven trading assistant** whose core rules are fully
deterministic and testable. LLMs are used only for:

- Research, explanation, summarisation.
- Strategy design in an offline “lab”.
- Workflow glue around clearly defined tools.

The system combines:

- **go‑utcp** – a universal tool‑calling layer (“muscles” / nervous system).
- **Dexter** – a financial research agent (fundamentals + narrative).
- An **Agent0‑style lab** – for experimenting with and optimising strategies.
- **Jax Core (Go)** – deterministic trading logic, risk, backtests.
- **React Dashboard** – human control panel and monitoring UI.

The design assumes: **no LLM directly places or sizes trades**. Humans remain
in control of execution, and all trading rules are explicit and testable.

---

## 2. Core Concepts

### 2.1 Deterministic vs Agentic

- **Deterministic layer (Jax Core)**  
  - Event detection (earnings, gaps, volatility, volume spikes).  
  - Strategy rules (when to enter/exit).  
  - Risk and position sizing.  
  - Backtesting logic.

- **Agentic layer (Dexter + Agent0 Lab)**  
  - Explanations (“why does this setup make sense?”).  
  - Research (“what is happening with this ticker / sector?”).  
  - Strategy exploration (“what rules work best historically?”).  
  - Reporting (“summarise this week’s trades and performance”).

The deterministic layer is what you would defend to a risk committee.  
The agentic layer is a set of smart assistants around it.

---

### 2.2 Tools and UTCP

Everything the system does is expressed as **tools** in the UTCP ecosystem.
Examples:

- `market.get_candles`
- `market.get_quote`
- `risk.position_size`
- `backtest.run_strategy`
- `dexter.research_company`
- `storage.save_trade`

go‑utcp gives the Go code a unified way to discover and call those tools.

---

## 3. High‑Level Data Flow

### 3.1 Idea / Event Generation

1. Jax uses UTCP tools to pull market data and corporate events.
2. Event detection logic evaluates:
   - Earnings surprises.
   - Large gaps.
   - Volume / volatility spikes.
3. Valid events become **Event** domain objects.

### 3.2 Trade Setup Creation

1. Strategy rules (loaded from config) decide:
   - Direction (long/short).
   - Entry trigger style (market open, breakout, pullback, etc.).
   - Stop‑loss placement rule (ATR, structure‑based, percentage).  
2. This creates **TradeSetup** objects linked to the originating Event.

### 3.3 Risk and Position Sizing

1. Risk engine calls `risk.position_size` / `risk.r_multiple`.  
2. Returns a **RiskResult** with:
   - Position size.
   - Total risk.
   - R multiple to the nearest target.

### 3.4 Research Attachments (Dexter)

1. For each trade setup, Jax can (optionally) call `dexter.research_company`.  
2. Dexter returns a research bundle (summary, key points, metrics).  
3. Jax stores that alongside the trade for the UI and journalling.

### 3.5 Agent0 Lab Loop (Offline)

1. Agent0‑style lab defines **LabTasks** describing parameter grids and symbol sets.  
2. Lab runs backtests via `backtest.run_strategy`.  
3. Lab scores configurations and writes:
   - JSON **StrategyConfig** files.  
   - Markdown **reports** explaining what worked.  
4. Jax Core loads these configs and uses them for live idea generation.

### 3.6 Human‑in‑the‑Loop

1. React dashboard displays:
   - Events, setups, risk, research, and backtests.  
2. Human can:
   - Approve, ignore, or manually adjust trade ideas.  
   - Trigger backtests and see lab reports.  
   - Adjust risk settings and strategy selections.

(Optional) Broker integration is a separate step and stays behind explicit
“place order” actions.

---

## 4. Tech Stack Summary

- **Backend Core:** Go + go‑utcp, HTTP server, clean architecture.
- **Research Engine:** Dexter (TypeScript/Bun), accessible via HTTP/CLI.
- **Lab:** Go process orchestrating backtests and generating configs/reports.
- **Frontend:** React + TypeScript, querying the Go HTTP API.
- **Storage:** start with SQLite / Postgres + simple JSON where convenient.

---

## 5. Recommended Build Order for Codex / AI

Codex (or any coding agent) should work through these files in order:

1. `02_utcp_providers.md` – implement the UTCP client + provider wrappers.
2. `03_dexter_integration.md` – wrap Dexter as UTCP tools and test them.
3. `04_jax_core.md` – build deterministic event/risk/trade engine + HTTP API.
4. `05_agent0_lab.md` – implement lab process for strategy experiments.
5. `06_react_dashboard.md` – implement the dashboard on top of the HTTP API.

Each of the following spec files will include explicit “Tasks for AI” sections
with concrete steps and suggested file paths.
