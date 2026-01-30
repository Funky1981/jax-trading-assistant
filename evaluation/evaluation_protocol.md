---
title: "Evaluation Protocol (Backtests That Don’t Lie)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["evaluation", "backtesting"]
---

## Non-negotiables
- **Walk-forward** evaluation (rolling train/test)
- **Out-of-sample** holdout (final untouched period)
- Model **fees + spread + slippage + impact** (venue-specific)
- Use **multiple testing correction** mindset (avoid overfitting)

## Overfitting controls
- Keep a registry of all tested variants
- Penalize repeated parameter sweeps
- Prefer simple rules with stable performance
- Use probabilistic/deflated performance measures when possible

## Minimum evidence package (for a candidate strategy)
- Description + assumptions + failure modes
- Universe/market selection logic
- Entry/exit logic with exact definitions
- Risk model and position sizing
- Cost model assumptions
- Results across regimes + stress periods
- Sensitivity to parameter changes
- Monitoring metrics and kill criteria

## Notes & references
Backtest overfitting and selection bias can inflate apparent Sharpe ratios; the Deflated Sharpe Ratio is one published approach to correcting for this.
