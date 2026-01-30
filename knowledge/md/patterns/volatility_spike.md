---
title: "Volatility Spike"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["pattern", "volatility"]
---
Detects sudden increases in realized volatility (RV) or ATR that often coincide with regime changes.

## Typical uses
- Reduce exposure (vol targeting)
- Switch from mean reversion → trend / risk-off
- Widen execution slippage assumptions

## Common implementations
- RV(20) / RV(100) ratio
- ATR percentile relative to trailing window
- GARCH-style vol forecasts (advanced)

## Failure modes
- Vol spikes can mean reversal *or* continuation depending on context.
