---
title: "Unhedged Short Volatility (Picking Up Nickels in Front of a Steamroller)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "tail-risk", "options"]
---
## Why it’s tempting
High win rate and steady gains in calm markets.

## Why it’s dangerous / usually wrong
Rare tail events can erase years of gains. Negative skew is the core problem.

## How to detect it in Jax (guardrails)
- Detect strategies with strongly negative skew / high short-gamma exposure.
- Require tail stress tests and explicit caps.
- Enforce max loss / scenario loss limits.

## Safer alternatives
- Use defined-risk structures.
- Pair with convex hedges.
- Keep exposure small and dynamically reduced in stress regimes.

## References
- Carry and short-vol strategies are known to embed crash/tail risk in many markets (conceptual).
