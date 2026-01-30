---
title: "Carry (Risk-Aware)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "carry", "fx", "rates", "risk"]
---
## What it is
Earns yield differentials (e.g., long high-rate, short low-rate currencies) or analogous carry in other markets.

## Why it can work (edge hypothesis)
Carry can compensate for providing liquidity/funding in normal times, but embeds crash risk.

## Best conditions
- Stable risk appetite
- Abundant funding liquidity
- Low crash risk environment

## Worst conditions / known failure modes
- Deleveraging events
- Funding stress / flight to safety
- Sudden regime shifts (crash risk)

## Signal definition (implementation-friendly)
Example (FX):
- Rank currencies by interest rate differential
- Long high differential basket, short low differential basket
- Overlay: cut exposure when funding stress indicators spike

## Execution notes (the part that kills most backtests)
- Crash risk is the main story. Use conservative leverage.
- Costs include funding and roll.
- Requires robust risk-off detector.

## Risk & sizing
- Strict exposure caps
- Tail hedges (optional) or hard de-risking when stress spikes
- Stop trading in extreme volatility regimes

## Monitoring & kill criteria
- Skewness and tail loss metrics
- Funding stress proxies
- Sudden correlation shifts

## References (starting points)
- Brunnermeier, Nagel & Pedersen (2008) *Carry Trades and Currency Crashes*.
