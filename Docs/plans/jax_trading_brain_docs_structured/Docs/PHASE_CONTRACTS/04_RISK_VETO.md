# Phase 4 Contract: Risk Veto

## Objective

Create a mandatory risk layer that can downgrade or reject any candidate.

## Delivers

- Risk assessment schema
- Risk veto rules
- Portfolio context model
- Risk tests

## Explicitly does not deliver

- Live trading
- Broker execution
- Full portfolio analytics
- Day trading risk model
- Options risk model

## User-facing capability made testable

Jax can reject a candidate even if the Swing Brain likes it.

## Acceptance tests

- Risk/reward below 2:1 rejects.
- Missing stop rejects.
- Event risk unresolved downgrades.
- Correlated exposure rejects or downgrades.
- Live execution remains forbidden.

## Required evidence

- Risk unit tests.
- Candidate downgrade examples.
- Capability matrix update.

## What Jax still cannot do afterwards

- Paper trade without approval.
- Validate strategy family through backtest.
- Execute live orders.
