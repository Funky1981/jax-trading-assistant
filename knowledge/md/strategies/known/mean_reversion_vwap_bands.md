---
title: "Mean Reversion (VWAP / Bands)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "mean-reversion", "intraday", "equities", "crypto"]
---
## What it is
Buys temporary dislocations away from a reference price (VWAP, moving average, statistical bands) expecting a pullback toward the mean.

## Why it can work (edge hypothesis)
Liquidity provision, inventory pressure, and short-term overreaction can create temporary mispricings.

## Best conditions
- Range-bound markets
- Stable volatility
- High liquidity periods

## Worst conditions / known failure modes
- Strong trends and breakouts
- News-driven moves
- Volatility spikes (mean keeps moving)

## Signal definition (implementation-friendly)
Example:
- Reference = VWAP (intraday) or MA(20)
- Enter when price deviates beyond k standard deviations (z-score)
- Exit at mean reversion target or time stop

## Execution notes (the part that kills most backtests)
- Needs tight spread/low fees; otherwise edge dies.
- Must avoid catching falling knives: add trend filter or volatility spike filter.
- Use limit orders cautiously; beware adverse selection.

## Risk & sizing
- Small initial sizing + add-on rules only with evidence
- Hard stops + time stops
- Volatility filter: reduce/disable when vol spikes

## Monitoring & kill criteria
- Win rate may be high but tail losses dominate; watch skew.
- If average loss grows vs average win → disable and reassess.

## References (starting points)
- General market microstructure literature on liquidity and temporary price impact (conceptual).
