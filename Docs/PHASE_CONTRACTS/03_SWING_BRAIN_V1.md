# Phase 3 Contract: Swing Brain v1

## Objective

Build the first specialist trading brain as a swing-trading vertical slice.

## Delivers

- Swing Brain v1
- Swing setup families
- Swing scoring
- Swing explanation
- Golden swing cases
- NO_TRADE/WATCH/TRADE_CANDIDATE outputs

## Explicitly does not deliver

- Day trading
- Tick-level logic
- Options flow
- Long-term valuation brain
- Live execution
- Autonomous trading

## User-facing capability made testable

Given a classified market event, Jax can decide whether there is a swing opportunity.

## Acceptance tests

- FTSE/oil/labour does not become TRADE_CANDIDATE.
- Missing invalidation returns NO_TRADE.
- Poor risk/reward returns NO_TRADE.
- Missing confirmation returns WATCH at most.
- Clear event/confirmation/risk-reward can return TRADE_CANDIDATE.

## Required evidence

- Golden cases.
- Unit tests.
- Capability matrix update.
- Example decisions.

## What Jax still cannot do afterwards

- Create paper ticket without risk gate.
- Prove setup family via backtest.
- Trade live.
