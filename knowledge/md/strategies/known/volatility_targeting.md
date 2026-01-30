---
title: "Volatility Targeting / Volatility Scaling"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "risk", "overlay"]
---
## What it is
Scales exposure **down when volatility is high** and **up when volatility is low**, aiming for steadier risk and improved risk-adjusted returns.

## Why it can work (edge hypothesis)
If expected returns don’t rise proportionally with volatility, scaling can improve Sharpe and reduce crash exposure.

## Best conditions
- Markets with volatility clustering
- When high vol predicts poorer subsequent performance

## Worst conditions / known failure modes
- Sharp vol drops after scaling down (whipsaw)
- If your vol estimator lags too much

## Signal definition (implementation-friendly)
Example:
- Target portfolio vol = V*
- Estimate recent vol Vhat (e.g., 20-day RV)
- Scale exposure by (V* / Vhat) with caps (min/max leverage)

## Execution notes (the part that kills most backtests)
- Use conservative caps; avoid forced leverage.
- Ensure margin requirements are modeled.
- Vol estimates must be robust to jumps.

## Risk & sizing
- Hard caps on leverage
- Separate “risk-off” state when vol exceeds extreme threshold
- Combine with drawdown-based de-risking

## Monitoring & kill criteria
- Realized vs target vol tracking
- Margin usage and stress tests

## References (starting points)
- Moreira & Muir (2017) *Volatility-Managed Portfolios*.
