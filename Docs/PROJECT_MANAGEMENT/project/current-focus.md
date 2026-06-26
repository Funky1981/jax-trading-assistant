# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Prepare a narrow internal adapter around the Phase 15 review operations app service only if explicitly approved.

## Active Phase

Review Operations Internal Adapter

## Active Work Items

- Reuse Phase 15 internal application service methods from `internal/decisioning/app`.
- Decide whether existing backend conventions justify a narrow internal handler or CLI-style operator adapter.
- Preserve human approval before any strategy, scoring, confirmation, or risk rule change.
- Keep research gaps and setup-family follow-up operationally visible without changing rules automatically.
- Preserve `NO_TRADE` as the default outcome.
- Keep review operations separate from broker execution, live trading, and automatic rule changes.

## Do Not Work On

- Live trading, broker execution, auto execution, unattended trading, and day trading.
- Paper execution unless explicitly included in a later phase contract.
- Frontend UI work unless a later phase calls for it.

## Current Constraints

- Research and paper-trading direction only; no live capital paths.
- Event Intelligence and Swing Brain are deterministic and must not rely on LLM calls as source of truth.
- Decision Core remains the owner of final decisions.

## Latest Decisions

Summarise recent decisions. Full details go in `/project/decisions.md`.

- Phase 2 Event Intelligence is implemented and tested in `internal/decisioning/classify`.
- Phase 3 Swing Brain v1 is implemented and tested in `internal/decisioning/brains/swing`.
- Phase 4 Risk Veto is implemented and tested in `internal/decisioning/risk`.
- Phase 5 Research/Backtest Evidence is implemented and tested.
- Phase 6 Paper Approval Loop is implemented and tested.
- Phase 7 Post Decision Review is implemented and tested.
- Phase 8 Decision Pipeline Integration is implemented and tested.
- Phase 9 Persistence and Observability Hardening is implemented and tested.
- Phase 10 Replay and Memory Feedback Reporting is implemented and tested.
- Phase 11 Review Operations and Human Feedback Triage is implemented and tested.
- Phase 12 Review Operations Persistence and Reporting is implemented and tested.
- Phase 13 Review Operations Workflow and Export Integration is implemented and tested.
- Phase 14 Review Operations Operator Access and Read Models is implemented and tested.
- Phase 15 Review Operations Internal Access Wiring is implemented and tested.

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Internal adapters could accidentally expose review actions as execution or rule-change authority; keep Decision Core, Risk Veto, Research Evidence, Paper Approval, and Review boundaries explicit.
- Golden coverage must protect every deterministic decision gate from execution, live-order, and auto-approval drift.

## Next 3 Actions

1. Decide whether a later phase should add internal handlers or CLI-style access around the Phase 15 app service.
2. Keep read-model queries read-only and manual actions record-only.
3. Preserve tests that prove operator access does not approve trades, execute orders, or mutate rules.

## Last Updated

2026-06-26
