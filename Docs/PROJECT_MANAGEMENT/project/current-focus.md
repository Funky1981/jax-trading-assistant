# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Build deterministic replay and memory feedback reporting over persisted decision pipeline results after Phase 9.

## Active Phase

Deterministic Replay and Memory Feedback Reporting

## Active Work Items

- Report on persisted pipeline results, decision records, and review schedules.
- Add deterministic replay/reporting views for due reviews, no-trades, and paper-review candidates.
- Preserve `NO_TRADE` as the default outcome.
- Keep replay and memory feedback separate from broker execution, live trading, and automatic rule changes.

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

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Replay/reporting work could accidentally treat stored pipeline output as execution authority; keep Decision Core, Risk Veto, Research Evidence, Paper Approval, and Review boundaries explicit.
- Golden coverage must protect every deterministic decision gate from execution, live-order, and auto-approval drift.

## Next 3 Actions

1. Design deterministic reports for due reviews and persisted decision outcomes without adding execution authority.
2. Add tests that prove replay/reporting preserves no-trade, risk-veto, research, paper-approval, and review gates.
3. Add memory feedback summaries for lessons and missed opportunities without automatic strategy/scoring/risk rule changes.

## Last Updated

2026-06-24
