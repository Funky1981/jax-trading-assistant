---
title: "Risk Controls & Kill Switches"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["risk", "controls"]
---

## Portfolio-level controls
- Max gross exposure
- Max net exposure (for market-neutral systems)
- Volatility targeting / risk scaling
- Max position concentration (per asset, per sector)
- Correlation / crowding limits
- Max leverage and margin usage caps

## Strategy-level controls
- Hard stop: max loss per trade / per day
- Time stop: exit if thesis doesn’t play out within T
- Liquidity stop: exit if spread widens beyond threshold
- Execution stop: disable if fills drop below X%

## Tail risk awareness

Some strategies (carry, short volatility, mean reversion in crashes) have **negative skew**:
- They win often and lose rarely — but catastrophically.
Use explicit tail hedges or strict risk caps if you deploy these.

## Position sizing (practical)
- Start with **volatility scaling** and conservative caps.
- Consider fractional Kelly only when you have stable edge estimates (often you don’t).
