# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Implement Phase 7 Post Decision Review after Phase 6 Paper Approval Loop.

## Active Phase

Phase 7 - Post Decision Review

## Active Work Items

- Define deterministic outcome review requirements for decisions, no-trades, and paper approvals.
- Preserve `NO_TRADE` as the default outcome.
- Keep review and learning separate from broker execution and live trading.

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

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Post-decision review could accidentally imply strategy promotion without enough outcomes; keep review evidence-only until promotion rules are satisfied.
- Golden coverage must protect Research Evidence, Swing Brain, Risk Veto, and Paper Approval outputs from bypassing review requirements.

## Next 3 Actions

1. Read the Phase 7 Post Decision Review contract.
2. Define deterministic review inputs and outputs on top of Decision Core, Research Evidence, Risk Veto, and Paper Approval results.
3. Add golden review cases for no-trades, rejected candidates, approved paper tickets, and prohibited live-execution promotion.

## Last Updated

2026-06-24
