---
title: "Strategy Lifecycle & Governance"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["meta", "governance"]
---

## States
- **Known / Approved**: allowed to trade live within limits.
- **Discovered / Candidate**: *never* trades live by default.
- **Shadow (Paper)**: trades on live data, simulated execution.
- **Pilot**: limited capital, strict kill-switches.
- **Retired**: no longer used; kept for learnings + regression tests.

## Promotion gates (minimum)

A candidate can only be promoted if:
- It survives **walk-forward** evaluation and out-of-sample tests
- It is profitable **after** realistic costs/slippage
- It is not a single-regime fluke (stress tests across regimes)
- It has clear **kill criteria** and monitoring metrics
- It passes **risk review** (drawdown, tail risk, concentration)

## Demotion / Kill-switch triggers

Examples (tune per venue):
- Breach of max drawdown or max daily loss
- Slippage exceeds modeled levels for N trades
- Fill rate collapses (execution becomes unrealistic)
- Regime classifier flags “out-of-domain”
- Live P&L deviates materially from expected distribution

## Auditability

Every trade must be explainable:
- Strategy ID + version
- Signal values at decision time
- Execution assumptions used (fees, slippage model)
- Risk checks passed/failed
