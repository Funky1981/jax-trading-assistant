---
title: "Trend Following (Time-Series Momentum)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "trend", "futures", "multi-asset"]
---

## What it is

A rules-based approach that goes **long when an instrument has been rising**, and **short when it has been falling**, usually using 1–12 month lookbacks.

## Why it can work (edge hypothesis)

Behavioral under-reaction and slow-moving macro fundamentals can create persistence in returns. Trend strategies often perform in crisis environments where diversification breaks down.

## Best conditions
- Prolonged directional moves
- High dispersion across assets
- Macro-driven regimes

## Worst conditions / known failure modes
- Choppy mean-reverting ranges
- Sudden trend reversals / turning points
- Momentum crashes

## Signal definition (implementation-friendly)

Example (simple):
- Compute return over lookback L (e.g., 12 months, excluding last 1 week)
- If return > 0 → long; if return < 0 → short
- Scale position by target volatility (see volatility targeting doc)

## Execution notes (the part that kills most backtests)
- Works best in liquid instruments with lower friction (futures, majors, large-cap).
- Costs matter: rebalance frequency is a trade-off between responsiveness and friction.
- Must model roll costs (futures) and funding/borrow (shorts).

## Risk & sizing
- Volatility targeting (scale down in high vol)
- Hard cap per instrument
- Trend stop: exit if signal flips; optional trailing stop
- Portfolio diversification across uncorrelated assets

## Monitoring & kill criteria
- Drift between expected and realized turnover/cost
- Drawdown limit per strategy sleeve
- Regime classifier: if “range-bound” probability high → reduce allocation

## References (starting points)
- Moskowitz, Ooi & Pedersen (2012) *Time Series Momentum* (Journal of Financial Economics).
- Levy & Lopes (2021) *Trend-Following Strategies via Dynamic Momentum Learning* (arXiv).
