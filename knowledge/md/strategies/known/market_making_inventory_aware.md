---
title: "Market Making (Inventory-Aware) — Advanced / Venue Dependent"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "market-making", "hft", "microstructure"]
---
## What it is
Provides bid/ask quotes to capture spread while managing inventory risk. This is **execution + microstructure heavy**.

## Why it can work (edge hypothesis)
Earns spread/fees (or rebates) for providing liquidity, but requires excellent execution and risk control.

## Best conditions
- Stable microstructure
- Predictable order flow
- Tight spreads + reliable fills

## Worst conditions / known failure modes
- Toxic flow / adverse selection
- News events
- Latency disadvantage

## Signal definition (implementation-friendly)
Example (conceptual):
- Quote around midprice with spread adjusted by inventory
- Widen quotes as inventory becomes imbalanced
- Pull quotes during volatility spikes

## Execution notes (the part that kills most backtests)
- Requires fast, reliable market data and order management.
- Latency and queue position matter.
- Backtests often lie if they don’t simulate fills realistically.

## Risk & sizing
- Strict inventory limits
- Volatility and news filters
- Kill switch on abnormal adverse selection

## Monitoring & kill criteria
- Fill rate, adverse selection, inventory P&L decomposition
- Queue position/fill simulation accuracy

## References (starting points)
- Academic lineage: Ho & Stoll (1981) and Avellaneda & Stoikov (2008) are commonly cited in market making literature (see survey-style PDFs).
