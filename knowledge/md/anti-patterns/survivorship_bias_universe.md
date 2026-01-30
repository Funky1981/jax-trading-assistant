---
title: "Survivorship Bias (Clean Universes That Never Existed)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "backtesting", "data"]
---
## Why it’s tempting
Backtests run faster on ‘current constituents’ lists and tidy datasets.

## Why it’s dangerous / usually wrong
You accidentally exclude delisted/bankrupt names, inflating performance — especially in equity strategies.

## How to detect it in Jax (guardrails)
- Require point-in-time membership (index constituents by date).
- Include delistings.
- Validate universe snapshots.

## Safer alternatives
- Use point-in-time index membership data.
- Track data coverage and missingness explicitly.

## References
