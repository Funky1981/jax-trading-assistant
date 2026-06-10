# Jax ETF News Strategy Pack

## Purpose

This pack defines three news-driven ETF paper-trading strategies for Jax.

These are designed for **paper trading only** until Jax has proven:
- data quality
- broker routing
- risk controls
- stop-loss enforcement
- audit logging
- human approval flow
- live-readiness gates

## Strategy Set

1. `01_market_panic_reversal_etf.md`
   - Trades broad-market panic/rebound using liquid index ETFs.

2. `02_sector_news_momentum_etf.md`
   - Trades sector/theme momentum after major confirmed news.

3. `03_rates_inflation_bonds_rotation_etf.md`
   - Trades bonds/equity rotation around rates, inflation, Fed/BoE-style macro shocks.

## Phase-1 ETF Rules

Allowed ETF universe:
- SPY
- QQQ
- DIA
- IWM
- XLK
- XLF
- XLE
- SMH
- SOXX
- TLT
- GLD

Excluded in phase 1:
- options
- leveraged ETFs
- inverse ETFs
- volatility ETFs
- thin/niche ETFs
- overnight autonomous holding
- live trading

## Absolute Guardrails

Jax must walk away if any of these fail:

- ETF not on allowlist
- market data stale
- spread too wide
- no stop-loss
- no take-profit or flatten-by-close rule
- news source not confirmed
- price already moved too far before entry
- broker is not in paper mode
- live trading enabled accidentally
- human approval missing
- daily loss limit hit
- max trades per day hit
- existing correlated ETF position already open

## Suggested Paper Limits

These are placeholders and should be configured by the operator:

- max open ETF positions: 1 initially
- max trades per day: 3
- max loss per trade: 0.5% to 1.0% of paper account
- max daily loss: 2%
- stop-loss required: yes
- take-profit required: yes
- flatten before close: yes for phase 1
- human approval: mandatory

## Important

These strategies are not financial advice. They are structured test plans for paper-trading system validation.
