# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Prepare backend operator access/read-model integration for review workflow and export outputs after Phase 13.

## Active Phase

Review Operations Operator Access and Read Models

## Active Work Items

- Define backend operator-facing read models or API contracts for review workflow batches, packets, exports, and follow-up action lists.
- Preserve deterministic export contracts for review operations reports.
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

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Operator access/read-model work could accidentally treat feedback suggestions as execution or rule-change authority; keep Decision Core, Risk Veto, Research Evidence, Paper Approval, and Review boundaries explicit.
- Golden coverage must protect every deterministic decision gate from execution, live-order, and auto-approval drift.

## Next 3 Actions

1. Design backend operator-facing read models or API contracts for review workflow/export outputs without adding execution authority.
2. Add tests that prove operator access preserves no-trade, risk-veto, research, paper-approval, and review gates.
3. Keep any operator workflow read-only or manual-action-only until a later phase explicitly changes scope.

## Last Updated

2026-06-24
