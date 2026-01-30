---
title: "Strategy Sprawl (No Governance, No Versioning)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["anti-pattern", "governance"]
---
## Why it’s tempting
It’s easy to keep adding variants when something underperforms.

## Why it’s dangerous / usually wrong
You end up with an un-auditable mess: unknown live exposure, unknown assumptions, and ‘why did we do that?’ incidents.

## How to detect it in Jax (guardrails)
- Enforce strategy IDs and semantic versioning.
- Require a promotion gate.
- Keep a registry of active strategies and allocations.

## Safer alternatives
- Use the lifecycle gates in `meta/strategy_lifecycle.md`.
- Keep strategies as immutable versions; deploy by reference.

## References
