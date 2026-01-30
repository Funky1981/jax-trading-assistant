---
title: "Order Flow / Order Book Imbalance"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["pattern", "microstructure"]
---
Measures whether aggressive buys vs sells dominate (venue dependent).

## Typical uses
- Micro-confirmation for breakouts
- Market making inventory management
- Detect liquidity vacuums

## Failure modes
- Highly sensitive to venue microstructure and data quality.
- Spoofing/quote stuffing can distort signals.
