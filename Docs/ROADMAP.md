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
   - Status: implemented and tested.
   - Goal: record and review outcomes for decisions, including no-trades and paper approvals.
9. **Phase 8: Decision Pipeline Integration and Hardening**
   - Status: implemented and tested.
   - Goal: wire the deterministic decision pipeline end-to-end while preserving risk veto, research evidence, human approval, paper-only, review scheduling, and live-trading exclusions.
10. **Phase 9: Persistence and Observability Hardening**
   - Status: implemented and tested.
   - Goal: persist pipeline results and review schedules, add operational visibility for deterministic decisions, and preserve all live-trading exclusions.
11. **Deterministic Replay and Memory Feedback Reporting**
   - Status: implemented and tested.
   - Goal: report on persisted decisions, due reviews, and replay outcomes so Jax can learn from no-trades and paper-review candidates without automatic rule changes or execution authority.
12. **Review Operations and Human Feedback Triage**
   - Status: implemented and tested.
   - Goal: help humans inspect due reviews and feedback suggestions, prioritize research actions, and decide whether to open rule-review work without changing rules automatically.
13. **Review Operations Persistence and Reporting**
   - Status: implemented and tested.
   - Goal: persist triage queues, human feedback decisions, and follow-up action outcomes for operational reporting while preserving human approval and all execution exclusions.
14. **Review Operations Workflow and Export Integration**
   - Status: implemented and tested.
   - Goal: expose persisted review-operation queues and deterministic reports to backend operator workflows and export contracts without frontend UI, broker execution, live trading, paper execution, or automatic rule changes.
15. **Review Operations Operator Access and Read Models**
   - Status: implemented and tested.
   - Goal: expose review workflow/export outputs through backend operator-facing read models and safe service methods while preserving read-only behavior, human approval, and all execution exclusions.
16. **Review Operations Internal Access Wiring**
   - Status: implemented and tested.
   - Goal: connect operator read models and record-only manual actions through a deterministic internal application service, without adding frontend UI, broker execution, live trading, paper execution, or automatic rule changes.
17. **Review Operations Internal Adapter**
   - Status: recommended next focus.
   - Goal: decide whether to add a narrow internal CLI or handler adapter around the Phase 15 app service, preserving read-only queries, record-only manual state transitions, human approval, and all execution exclusions.

## Explicitly Not Planned

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.

## Supporting Work

- A narrow internal adapter around the Phase 15 app service is the recommended next focus after Review Operations Internal Access Wiring. This is not a live-trading phase.
- Historical plans and reports are preserved in `Docs/plans/` and `Docs/archive/`; they are not the active source of truth.
