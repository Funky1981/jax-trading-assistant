# Jax Robust Profitability Layer

## Purpose

This folder adds the next production-grade layer for Jax.

The previous folders build:

```text
macro-reaction-engine/
analysis-intelligence-layer/
```

This folder makes the system more robust by adding:

```text
market regime awareness
cross-asset confirmation
economic calendar integration
confounder detection
liquidity and execution checks
position sizing
portfolio risk
strategy playbooks
walk-away logic
post-trade review
performance dashboards
stress testing
```

## Core principle

Jax does not become profitable by trading more.

Jax becomes more robust by:

```text
rejecting weak trades
avoiding misattributed moves
waiting for confirmation
sizing correctly
tracking outcomes
learning from mistakes
```

## Target system flow

```text
Event detected
  ↓
Validated research trigger
  ↓
Structured macro/news analysis
  ↓
Fundamental analysis
  ↓
Confounder check
  ↓
Market regime check
  ↓
Cross-asset confirmation
  ↓
Technical analysis
  ↓
Liquidity/execution quality check
  ↓
Position sizing and portfolio risk
  ↓
Risk manager veto
  ↓
Evidence bundle
  ↓
Trade grade
  ↓
Human approval
  ↓
Paper trade
  ↓
Post-trade review
  ↓
Memory update
```

## Non-negotiable rules

```text
No live trading from this layer
No broker order without human approval
No candidate when market regime conflicts with the setup
No candidate when cross-asset confirmation is missing
No candidate when a major confounder exists
No candidate when liquidity/execution quality is poor
No candidate when position sizing cannot be calculated
No candidate when portfolio exposure limits are breached
No candidate when walk-away rules trigger
```

## Folder order

```text
01_MARKET_REGIME_ENGINE.md
02_CROSS_ASSET_CONFIRMATION.md
03_ECONOMIC_CALENDAR_INTEGRATION.md
04_CONFOUNDER_ENGINE.md
05_LIQUIDITY_EXECUTION_QUALITY.md
06_POSITION_SIZING_AND_PORTFOLIO_RISK.md
07_STRATEGY_PLAYBOOKS.md
08_WALK_AWAY_ENGINE.md
09_POST_TRADE_REVIEW_AND_LEARNING.md
10_PERFORMANCE_DASHBOARD.md
11_MONTE_CARLO_AND_STRESS_TESTING.md
12_CODEX_MASTER_PROMPT.md
13_PHASED_CODEX_TICKETS.md
14_FOLDER_PLACEMENT.md
```

## Desired Jax behaviour

Most events should end with:

```text
No trade.
Reason: insufficient confirmation / poor risk / conflicting regime / confounder detected.
```

That is success.

A trade candidate should be rare, boring, and evidence-heavy.
