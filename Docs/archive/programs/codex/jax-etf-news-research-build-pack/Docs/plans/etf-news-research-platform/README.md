# Jax ETF News Research Platform Build Pack

## Purpose

This pack gives Codex a structured implementation plan for turning Jax into an ETF-only, news-driven, historical-research and paper-trading assistant.

The target product:

> Jax researches news events, compares historic ETF reactions, checks if news was already priced in, identifies other news that may have affected price, generates evidence-backed ETF paper-trade candidates, and sends beginner-friendly approval summaries to mobile before any order can be placed.

## Non-Negotiables

- ETF-only.
- Paper trading only until long-running testing proves quality.
- No options.
- No leveraged ETFs in phase 1.
- No inverse ETFs in phase 1.
- No volatility ETFs in phase 1.
- Human approval before execution.
- AI explains and ranks; it does not directly trade.
- Postgres remains the source of truth.
- Existing Jax architecture should be extended, not replaced.

## Main Branch Current-State Assumptions

Jax main already has:

- `cmd/trader` deterministic trading/API runtime.
- `cmd/research` orchestration/research/backtest/memory runtime.
- Postgres + pgvector.
- ETF phase-one allowlist policy.
- Candidate trade and approval tables.
- Event storage foundation.
- Market data tables for quotes/candles.
- Memory banks: `research`, `trades`, `signals`, `reflections`.
- Paper trading UAT/docs.
- IB bridge/paper trading plumbing.
- Testing/readiness API surfaces.

Main gaps:

- ETF-only is not universal across defaults.
- Historical event-study analytics are missing.
- ETF-aware news relevance is incomplete.
- Priced-in scoring is not first-class.
- Confounder/overlapping-news analysis is missing.
- Mobile approval workflow is not fully designed.
- Beginner UX needs strategy cards and evidence summaries.

## Recommended Build Order

1. ETF-only universal defaults.
2. Historical event-study schema.
3. Historical ETF/news backfill pipeline.
4. ETF-aware event classification and mapping.
5. Priced-in scoring engine.
6. Research evidence bundle generation.
7. Strategy integration for three ETF news strategies.
8. Mobile approval notifications.
9. Beginner-friendly UI.
10. Production live-feed hardening.
