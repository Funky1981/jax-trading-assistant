---
title: "Strategy Selection Rules (Ensemble)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["meta", "allocation"]
---

## Principle

Don’t pick a ‘winner’. Run an **ensemble** with:
- small allocations,
- regime gating,
- risk overlay,
- and hard kill switches.

## Example (illustrative)
- If P(trend) high → allocate to Trend Following + Breakout; reduce Mean Reversion.
- If P(range) high → allocate to Mean Reversion + Pairs; reduce Breakout.
- If P(stress) high → reduce overall exposure; tighten risk limits; prefer trend + defensive overlays.
- If liquidity stress high → reduce frequency; widen slippage model; disable market making.

## Always-on overlay
- Vol targeting
- Drawdown control
- Exposure caps
