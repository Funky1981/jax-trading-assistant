# Phase 1 Contract: Decision Core

## Objective

Create the first-class event and decision model for Jax.

## Delivers

- Event schema
- Decision schema
- Evidence bundle schema
- Decision enum
- Deterministic evaluator v1
- FTSE/oil/labour golden case
- Golden decision runner

## Explicitly does not deliver

- Event scraping
- Full news ingestion
- Swing Brain
- Risk veto
- Broker integration
- Paper trading
- Backtesting
- Frontend UI
- Live trading

## User-facing capability made testable

Given a structured market event, Jax returns a structured decision.

## Acceptance tests

- FTSE/oil/labour conflict returns `NO_TRADE`.
- `NO_TRADE` includes primary reason.
- Forbidden actions include `execute_trade`.
- Allowed actions include `store_event`, `monitor`, `review_later`.
- Decision enum includes all required values.

## Required evidence

- Unit tests.
- Golden test fixtures.
- Capability matrix update.

## What Jax still cannot do afterwards

- Parse unstructured articles fully.
- Evaluate swing setups.
- Create trade tickets.
- Backtest setup families.
