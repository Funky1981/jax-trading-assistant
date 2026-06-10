# 00 — Current State

## Current Capabilities On Main

Jax already appears to have the right foundation for the target product.

### Runtime Architecture

- `cmd/trader`
  - deterministic trading/API runtime
  - frontend-facing API
  - broker/order/position/account endpoints
  - strategy/recommendation/signal endpoints
  - ETF pilot/readiness surfaces
  - trading guard and risk endpoints

- `cmd/research`
  - orchestration runtime
  - backtest support
  - research project runner
  - memory proxy
  - Agent0/Dexter integration points

- Supporting services
  - `ib-bridge`
  - `agent0-service`
  - React frontend
  - Postgres + pgvector
  - Prometheus/Grafana

## Existing Data Foundations

Known existing schema areas:

- `quotes`
- `candles`
- `event_sources`
- `event_raw`
- `event_normalized`
- `event_symbol_map`
- `candidate_trades`
- `candidate_events`
- `candidate_approvals`
- `execution_instructions`
- `memory_items`

## Existing ETF Direction

Main already has an ETF phase-one policy with:

- approved ETF allowlist
- excluded leveraged/inverse/volatility products
- stop-loss requirement
- flatten-by-close requirement
- paper-only phase-one posture
- candidate approval workflow

## Current Weak Spots

### ETF-only is not universal

Some defaults still include single-name equities. Codex should search for these examples and remove or isolate them from the ETF paper-trading path:

```text
AAPL
MSFT
GOOGL
AMZN
TSLA
META
NVDA
AMD
NFLX
```

These may remain only in historical examples/tests if explicitly marked as non-ETF examples, but not in ETF phase-one defaults.

### Historical research intelligence is missing

Jax can store events and candles, but needs first-class analytics for:

- event windows
- ETF price reactions
- confounding events
- priced-in scoring
- evidence bundles
- historical comparison

### ETF relevance is too symbol-centric

ETF movement is often caused by:

- macro news
- sector news
- constituent news
- rates/inflation
- commodity shocks
- geopolitical events

The research layer must map news to ETFs even when the ETF ticker is not mentioned directly.

## Codex Instruction

Do not rebuild the app.

Extend existing modules, tables, and flows.
