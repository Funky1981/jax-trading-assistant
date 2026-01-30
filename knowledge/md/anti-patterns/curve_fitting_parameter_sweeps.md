---
title: "Curve-Fitting / Parameter Sweep Overfitting"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "overfitting", "backtesting"]
---
## Why it’s tempting
You can always find parameters that look amazing in sample.

## Why it’s dangerous / usually wrong
The ‘best’ backtest often captures random patterns that disappear live, especially when you tried many variants.

## How to detect it in Jax (guardrails)
- Track how many variants were tested.
- Penalize complexity and repeated tuning.
- Require walk-forward + untouched holdout.

## Safer alternatives
- Keep models simple.
- Use walk-forward and multiple-testing-aware evaluation.
- Prefer stable performance over peak performance.

## References
- Bailey & López de Prado (2014) Deflated Sharpe Ratio and overfitting discussion.
