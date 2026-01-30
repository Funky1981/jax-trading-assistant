---
title: "Momentum (Cross-Sectional / Relative Strength)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "momentum", "equities", "cross-sectional"]
---
## What it is
Ranks assets by past performance and goes **long winners / short losers** (market-neutral or beta-managed).

## Why it can work (edge hypothesis)
Relative performance persistence has been widely documented in equities and other assets, though it is sensitive to costs and crashes.

## Best conditions
- Stable growth regimes
- Moderate volatility
- High dispersion between winners and losers

## Worst conditions / known failure modes
- Sharp reversals / bear market rebounds
- High transaction cost environments
- Crowding and forced deleveraging

## Signal definition (implementation-friendly)
Example:
- Monthly rank assets by 12-1 month returns (12 months excluding most recent 1 month)
- Long top decile, short bottom decile
- Beta-neutralize or risk-parity weight positions

## Execution notes (the part that kills most backtests)
- Turnover can be high; costs can erase returns.
- Shorting can be expensive or impossible in some venues.
- Must model borrow fees and constraints.

## Risk & sizing
- Volatility scaling
- Crash protection: reduce exposure when market volatility spikes
- Concentration limits; avoid illiquid tails

## Monitoring & kill criteria
- Turnover vs forecast
- Exposure to common factors (beta, size, value)
- “Momentum crash” detector (rapid factor reversal)

## References (starting points)
- Jegadeesh & Titman (1993) seminal work on momentum (referenced widely).
- Korajczyk & Sadka (2004) *Are Momentum Profits Robust to Trading Costs?*
- Kelly (2021) review on momentum literature (JFE).
