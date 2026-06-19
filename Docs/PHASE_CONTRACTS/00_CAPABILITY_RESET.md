# Phase 0 Contract: Capability Reset

## Objective

Stop scope drift and create one source of truth for what Jax can and cannot do.

## Delivers

- Product charter
- Capability matrix
- Required documentation spine
- Swing-first roadmap
- Current capability status
- Explicit exclusion of day trading and live trading
- Phase contract standard

## Explicitly does not deliver

- Trading logic
- Decision engine code
- Swing Brain code
- Broker execution
- Paper ticket creation
- Backtesting implementation
- Frontend changes
- Live trading

## User-facing capability made testable

None. This is a governance/reset phase.

## Acceptance tests

- `Docs/JAX_PRODUCT_CHARTER.md` exists.
- `Docs/CAPABILITY_MATRIX.md` exists.
- `Docs/PHASE_CONTRACTS/00_CAPABILITY_RESET.md` exists.
- Day Trading Brain is marked `NOT_PLANNED`.
- Live trading is marked `NOT_PLANNED`.
- Swing Brain is marked `PLANNED`.
- NO_TRADE is defined as default.

## Required evidence

- Updated docs committed.
- Capability matrix created.
- Next phase contract created before implementation starts.

## What Jax still cannot do afterwards

- Classify events.
- Decide NO_TRADE/WATCH/TRADE_CANDIDATE.
- Create paper tickets from decisions.
- Review no-trades.
