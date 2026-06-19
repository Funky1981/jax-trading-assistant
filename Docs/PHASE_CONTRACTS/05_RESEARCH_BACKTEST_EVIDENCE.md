# Phase 5 Contract: Research and Backtest Evidence

## Objective

Define and validate the evidence required before a setup family can become paper-ready.

## Delivers

- Research hypothesis schema
- Backtest evidence schema
- Dataset integrity requirements
- Slippage/cost requirements
- Out-of-sample requirements
- Promotion rules

## Explicitly does not deliver

- New backtest engine rewrite
- Live trading
- Auto-promotion to live
- Guaranteed profitability
- Day-trading evidence model

## User-facing capability made testable

Jax can explain why a setup family is or is not ready for paper trading.

## Acceptance tests

- Backtest evidence without dataset hash fails.
- Backtest evidence without slippage fails.
- Backtest evidence without OOS notes fails.
- Weak evidence cannot become PAPER_READY.
- Promising evidence can become BACKTESTED_PROMISING.

## Required evidence

- Evidence validation tests.
- Example evidence bundle.
- Capability matrix update.

## What Jax still cannot do afterwards

- Automatically paper trade.
- Live trade.
- Learn from outcomes unless review phase is complete.
