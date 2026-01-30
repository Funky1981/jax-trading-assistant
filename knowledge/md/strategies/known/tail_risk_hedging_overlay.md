---
title: "Tail Risk Hedging (Convexity Overlay)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "risk", "tail-hedge", "options"]
---
## What it is
Adds convex payoff structures intended to offset large crash losses (e.g., defined-risk options structures).

## Why it can work (edge hypothesis)
Some strategy mixes have negative skew; convex hedges can reduce blow-up risk at the cost of carry/drag.

## Best conditions
- High uncertainty / crash-prone regimes
- When portfolio has embedded short-vol exposure

## Worst conditions / known failure modes
- Long calm regimes (hedge carry cost)
- Poorly designed hedges that bleed excessively

## Signal definition (implementation-friendly)
Example:
- Allocate a small budget to convex hedges (defined-risk)
- Increase hedge allocation when stress probability rises
- Keep hedges diversified and sized as ‘insurance’

## Execution notes
- Options liquidity and pricing are critical.
- Avoid excessive theta bleed; treat as insurance premium.
- Must model execution + implied vol surface changes.

## Risk & sizing
- Budget the hedge cost explicitly
- Defined-risk structures only
- Avoid over-hedging and creating new fragility

## Monitoring & kill criteria
- Hedge P&L attribution (carry vs payoff)
- Crash correlation effectiveness
- Cost budget adherence

## References (starting points)
