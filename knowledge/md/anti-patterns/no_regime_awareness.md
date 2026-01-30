---
title: "No Regime Awareness (One Strategy To Rule Them All)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "regime"]
---
## Why it’s tempting
Simplicity: ‘Just run it everywhere all the time.’

## Why it’s dangerous / usually wrong
Strategies have regimes they thrive in and regimes they die in (trend vs range, calm vs crisis). Ignoring this increases drawdowns and failure probability.

## How to detect it in Jax (guardrails)
- Require each strategy to declare best/worst conditions.
- Run regime classifier and allocation rules.
- Force de-risking when out-of-domain.

## Safer alternatives
- Use a strategy ensemble with regime gating.
- Keep a ‘risk overlay’ sleeve (vol scaling, drawdown control).

## References
