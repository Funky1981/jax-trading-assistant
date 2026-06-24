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
   - Status: implemented and tested.
   - Source: `Docs/PHASE_CONTRACTS/02_EVENT_INTELLIGENCE.md`.
   - Goal: classify events, extract drivers, detect conflicts, and map affected assets.
   - Constraint: use Decision Core outputs; do not add trading execution or Swing Brain logic.
4. **Phase 3: Swing Brain v1**
   - Status: implemented and tested.
   - Source: `Docs/STRATEGIES/SWING_TRADING/` and `Docs/PHASE_CONTRACTS/03_SWING_BRAIN_V1.md`.
   - Goal: evaluate swing setups only after Decision Core and Event Intelligence exist.
5. **Phase 4: Risk Veto**
   - Status: implemented and tested.
   - Goal: reject or downgrade swing candidates when risk constraints fail before any paper ticket or approval workflow exists.
6. **Phase 5: Research/Backtest Evidence**
   - Status: implemented and tested.
   - Goal: attach structured research and backtest evidence before any setup family can progress toward paper approval.
7. **Phase 6: Paper Approval Loop**
   - Status: implemented and tested.
   - Goal: require human approval before any paper-trading workflow can be prepared.
8. **Phase 7: Post Decision Review**
   - Status: next implementation phase.
   - Goal: record and review outcomes for decisions, including no-trades and paper approvals.

## Explicitly Not Planned

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.

## Supporting Work

- Post Decision Review is the next planned supporting phase after Paper Approval Loop.
- Historical plans and reports are preserved in `Docs/plans/` and `Docs/archive/`; they are not the active source of truth.
