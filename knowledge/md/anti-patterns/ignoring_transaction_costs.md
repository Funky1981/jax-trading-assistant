---
title: "Ignoring Transaction Costs / Slippage"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "execution", "costs"]
---
## Why it’s tempting
Backtests look incredible when you assume perfect fills and tiny fees.

## Why it’s dangerous / usually wrong
Costs and impact can drive expected returns to zero, especially for high turnover strategies like momentum or intraday mean reversion.

## How to detect it in Jax (guardrails)
- Enforce a venue-specific cost model.
- Require sensitivity analysis: +25%, +50% costs.
- Compare simulated vs realized slippage in paper trading.

## Safer alternatives
- Trade more liquid instruments.
- Reduce turnover.
- Use execution algorithms and limit orders where appropriate (but model adverse selection).

## References
- Korajczyk & Sadka (2004) on momentum and trading costs.
