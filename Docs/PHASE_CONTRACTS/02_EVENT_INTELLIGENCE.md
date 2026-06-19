# Phase 2 Contract: Event Intelligence

## Objective

Classify events, extract drivers, detect conflicts, and map affected assets.

## Delivers

- Event classifier
- Driver extractor
- Conflict detector
- Asset mapper
- Event taxonomy tests
- Additional golden cases

## Explicitly does not deliver

- Trade decisions beyond Decision Core rules
- Swing-specific setup scoring
- Risk sizing
- Paper ticket creation
- Backtesting
- Live execution

## User-facing capability made testable

Given an event/article package, Jax explains what happened and which assets may be affected.

## Acceptance tests

- Macro event classification works.
- Commodity shock classification works.
- Earnings/guidance classification works.
- FTSE/oil/labour identifies conflicting drivers.
- Oil maps to BP/SHEL/energy exposure.
- UK labour maps to GBP/gilts/UK sectors.

## Required evidence

- Unit tests.
- Golden event updates.
- Capability matrix update.

## What Jax still cannot do afterwards

- Properly evaluate swing candidates.
- Produce paper tickets.
- Prove strategies through backtests.
