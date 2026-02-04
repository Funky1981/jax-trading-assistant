---
title: "Latency Illusions (Backtests That Assume You’re Fast)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "hft", "microstructure"]
---

## Why it’s tempting

Tick-level signals can look predictive if you assume instant reaction.

## Why it’s dangerous / usually wrong

If you’re slower than the market, the edge vanishes or flips sign. Queue position matters; fills aren’t guaranteed.

## How to detect it in Jax (guardrails)
- Label strategies needing sub-second reaction as ‘HFT’.
- Require realistic order book simulation or reject.
- Enforce minimum latency assumptions based on your infra.

## Safer alternatives
- Use slower timeframes (minutes/hours/days).
- Use signals robust to latency (trend/mean reversion with wider thresholds).

## References
- Market making literature emphasizes execution realism; naive backtests frequently overstate performance.
