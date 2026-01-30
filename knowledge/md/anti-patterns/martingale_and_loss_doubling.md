---
title: "Martingale / Loss Doubling"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "risk-of-ruin"]
---
## Why it’s tempting
It wins *a lot* of small trades and looks ‘safe’ in short backtests.

## Why it’s dangerous / usually wrong
Losses grow exponentially; with finite bankroll and real-world limits, you eventually hit a loss streak that wipes you out. It’s a risk-of-ruin machine.

## How to detect it in Jax (guardrails)
- Flag any sizing rule that increases size after losses beyond a small bounded step.
- Enforce max position size and max leverage.
- Require positive expected value independent of sizing.

## Safer alternatives
- Use volatility targeting + fixed risk-per-trade sizing.
- If edge exists, scale slowly using rolling risk estimates.

## References
- Martingale betting system overview and risk-of-ruin intuition.
