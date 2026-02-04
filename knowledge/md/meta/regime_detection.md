---
title: "Regime Detection (Practical)"
version: "1.0"
status: "approved"
created_utc: "2026-01-30"
tags: ["meta", "regime"]
---

## Goal

Estimate which market ‘mode’ we are in so Jax can adjust allocation:
- **Trend**
- **Range / Mean Reversion**
- **High Vol / Stress**
- **Low Vol / Calm**
- **Liquidity stress** (venue dependent)

## Simple, robust features (start here)
- Realized volatility level and change (volatility spike pattern)
- Trend strength (e.g., moving-average slope, ADX-style measures)
- Autocorrelation sign (momentum vs reversal tendency)
- Dispersion across assets (cross-sectional spread)
- Spread / depth metrics (liquidity)

## Output

A probability vector, e.g.:
- P(trend), P(range), P(stress), P(calm), P(liquidity_stress)

## Use
- Gate strategies (enable/disable)
- Allocate weights (more to trend in trend regimes)
- Adjust cost/slippage assumptions (higher in stress)
