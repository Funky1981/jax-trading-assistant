---
title: "Lookahead Bias / Data Leakage"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "backtesting", "data"]
---
## Why it’s tempting
It happens accidentally: using end-of-day data to trade intraday, using revised fundamentals, survivorship-clean universes, etc.

## Why it’s dangerous / usually wrong
It manufactures an edge that never existed live. One of the fastest ways to build a ‘perfect’ strategy that instantly fails.

## How to detect it in Jax (guardrails)
- Enforce timestamp discipline on every feature.
- Use point-in-time datasets.
- Disallow using future bars (including close) unless trade occurs after close.
- Audit feature lineage.

## Safer alternatives
- Build a data validation layer.
- Use replay engines with strict clocking.
- Maintain point-in-time universes and corporate actions.

## References
