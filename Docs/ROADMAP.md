# Roadmap

## Product Direction

Jax is an event-driven trading research assistant. The active product truth is `Docs/JAX_PRODUCT_CHARTER.md`, and capability status is tracked in `Docs/CAPABILITY_MATRIX.md`.

The default decision is `NO_TRADE`. Jax upgrades from `NO_TRADE` only when structured evidence justifies the next decision state.

## Active Roadmap

1. **Phase 0: Capability Reset**
   - Status: complete.
   - Source: `Docs/PHASE_CONTRACTS/00_CAPABILITY_RESET.md`.
   - Outcome: product truth, capability matrix, active documentation spine, and explicit exclusions.
2. **Phase 1: Decision Core**
   - Status: implemented and tested.
   - Source: `Docs/PHASE_CONTRACTS/01_DECISION_CORE.md`.
   - Goal: define structured Event, Decision, EvidenceBundle, decision enum, deterministic evaluator v1, and golden decision runner.
   - Default: `NO_TRADE`.
3. **Phase 2: Event Intelligence**
   - Status: next implementation phase.
   - Source: `Docs/PHASE_CONTRACTS/02_EVENT_INTELLIGENCE.md`.
   - Goal: classify events, extract drivers, detect conflicts, and map affected assets.
   - Constraint: use Decision Core outputs; do not add trading execution or Swing Brain logic.
4. **Swing Brain v1**
   - Status: first active strategy brain after Decision Core.
   - Source: `Docs/STRATEGIES/SWING_TRADING/` and `Docs/PHASE_CONTRACTS/03_SWING_BRAIN_V1.md`.
   - Goal: evaluate swing setups only after Decision Core and Event Intelligence exist.

## Explicitly Not Planned

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.

## Supporting Work

- Risk veto, paper approval, research/backtest evidence, and memory/review remain planned supporting phases.
- Historical plans and reports are preserved in `Docs/plans/` and `Docs/archive/`; they are not the active source of truth.
