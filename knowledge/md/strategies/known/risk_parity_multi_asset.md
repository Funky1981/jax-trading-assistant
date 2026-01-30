---
title: "Risk Parity (Diversified Multi-Asset Allocation)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["strategy", "allocation", "risk-parity", "multi-asset"]
---
## What it is
Allocates across asset classes so that each contributes roughly equally to total portfolio risk, often using leverage to reach a target risk level.

## Why it can work (edge hypothesis)
Diversification across uncorrelated assets can reduce portfolio drawdowns; risk parity emphasizes risk balance rather than capital balance.

## Best conditions
- When asset-class diversification holds
- When disciplined rebalancing is maintained
- When leverage and financing are controlled

## Worst conditions / known failure modes
- When correlations spike across assets (crises)
- When leverage/financing becomes constrained
- When bond-heavy allocations suffer from rising yields (depends on implementation)

## Signal definition (implementation-friendly)
Example:
- Choose asset buckets (equities, nominal bonds, inflation hedges, etc.)
- Estimate vol and correlations
- Weight to equalize risk contribution
- Rebalance on schedule; apply leverage caps

## Execution notes
- Requires robust vol/correlation estimates.
- Funding/financing matters if leverage is used.
- Rebalancing costs can be non-trivial; model them.

## Risk & sizing
- Conservative leverage caps
- Stress tests on correlation spikes
- Liquidity-aware rebalancing

## Monitoring & kill criteria
- Risk contribution drift
- Correlation regime shifts
- Financing/margin stress metrics

## References (starting points)
- Bridgewater’s public discussion of All Weather / risk parity concepts (conceptual reference).
