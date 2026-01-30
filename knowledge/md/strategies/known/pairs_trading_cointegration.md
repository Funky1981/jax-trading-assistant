---
title: "Pairs Trading (Stat Arb / Cointegration)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "stat-arb", "market-neutral", "pairs"]
---
## What it is
Trades the spread between two related assets: **short the rich / long the cheap** when their relationship deviates, expecting convergence.

## Why it can work (edge hypothesis)
Relative-value relationships can revert due to fundamentals, shared risk factors, or market-making forces.

## Best conditions
- Stable relationships
- Liquid pairs
- Moderate volatility

## Worst conditions / known failure modes
- Structural breaks (relationship changes)
- Regime shifts
- Costs/borrow constraints

## Signal definition (implementation-friendly)
Example:
- Identify candidate pairs (distance/sector/fundamentals)
- Estimate spread and z-score (or cointegration residual)
- Enter when |z| > threshold; exit when z returns near 0
- Optional: stop if |z| continues to widen (break risk)

## Execution notes (the part that kills most backtests)
- Shorting and borrow fees can make or break performance.
- Correlated crashes can hurt both legs.
- Must account for asynchronous trading hours (cross-market pairs).

## Risk & sizing
- Dollar-neutral or beta-neutral sizing
- Hard stop on spread widening
- Limit per-pair exposure + portfolio diversification across pairs

## Monitoring & kill criteria
- Relationship stability metrics (rolling cointegration tests)
- Borrow costs and availability
- Structural break detector (parameter drift)

## References (starting points)
- Gatev, Goetzmann & Rouwenhorst (2006) *Pairs Trading: Performance of a Relative-Value Arbitrage Rule*.
