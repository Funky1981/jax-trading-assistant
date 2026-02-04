---
title: "Stop-Loss & Drawdown Control (Overlay)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "risk", "overlay", "stops"]
---

## What it is

Rules that reduce exposure after losses or when price moves against the position, aiming to cap drawdowns.

## Why it can work (edge hypothesis)

Stops can improve drawdown characteristics in some implementations, but can also reduce expected returns and increase turnover/costs.

## Best conditions
- When protecting against sudden crashes is the priority
- When combined with sensible thresholds and cost controls

## Worst conditions / known failure modes
- Noisy markets (whipsaw)
- Tight stops that cause excessive turnover
- When stops are substituted for a real edge

## Signal definition (implementation-friendly)

Example:
- Per-trade hard stop (max loss)
- Time stop (exit if not working)
- Portfolio drawdown stop (de-risk when DD exceeds threshold)
- Volatility-aware stop distances (wider in higher vol)

## Execution notes
- Costs rise as stop frequency rises.
- Stops must be tuned to venue slippage behavior.
- Consider using close-based or volatility-adjusted stops to reduce noise.

## Risk & sizing
- Stops are a **risk tool**, not a profit engine.
- Combine with position sizing and regime gating.
- Always define max daily loss / max drawdown.

## Monitoring & kill criteria
- Whipsaw rate (stop-outs that reverse)
- Turnover and slippage vs model
- Drawdown improvements vs performance drag

## References (starting points)
- Research Affiliates (2025) discussion of stop-loss impacts (example study).
- Clare (2012) discusses trade-offs and when stop-loss can reduce expected return (context).
