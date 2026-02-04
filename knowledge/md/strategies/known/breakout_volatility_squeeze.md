---
title: "Breakout (Volatility Squeeze)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "breakout", "trend", "volatility"]
---

## What it is

Trades expansions from low-volatility consolidations, aiming to catch the start of a new directional move.

## Why it can work (edge hypothesis)

Price compression can precede information releases and positioning shifts; breakouts can self-reinforce via stop orders and trend followers.

## Best conditions
- Transition from low to higher volatility
- Strong catalyst or macro impulse
- Assets prone to trending

## Worst conditions / known failure modes
- False breakouts in low liquidity
- Choppy markets
- Mean-reverting regimes

## Signal definition (implementation-friendly)

Example:
- Detect low-vol regime (BB width percentile or RV percentile)
- Entry: break above/below consolidation range with confirmation (volume/imbalance)
- Exit: trailing stop or signal reversal

## Execution notes (the part that kills most backtests)
- Slippage spikes on breakout; model it.
- Prefer liquid venues; avoid thin books.
- Consider waiting for close confirmation to reduce noise (trade-off: later entry).

## Risk & sizing
- Position size scaled by vol + capped
- Maximum gap/slippage allowance; if exceeded, skip entry
- Trailing stops to avoid round-trips

## Monitoring & kill criteria
- Track false breakout rate
- If slippage > modeled median for N trades → disable in that venue/timeframe

## References (starting points)
- Trend-following literature overlaps; breakout is a common implementation of trend logic.
