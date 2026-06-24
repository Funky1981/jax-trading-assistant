# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Implement Phase 6 Paper Approval Loop after Phase 5 Research/Backtest Evidence.

## Active Phase

Phase 6 - Paper Approval Loop

## Active Work Items

- Define deterministic human approval requirements before paper tickets are created.
- Preserve `NO_TRADE` as the default outcome.
- Keep paper approval separate from broker execution and live trading.

## Do Not Work On

- Live trading, broker execution, auto execution, unattended trading, and day trading.
- Paper ticket creation unless explicitly included in the Phase 6 contract.
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

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Paper approval could accidentally imply execution authority; keep it approval-only until a later phase.
- Golden coverage must protect Research Evidence, Swing Brain, and Risk Veto outputs from bypassing human approval.

## Next 3 Actions

1. Read the Phase 6 Paper Approval Loop contract.
2. Define deterministic approval inputs and outputs on top of Research Evidence, Swing Brain, and Risk Veto results.
3. Add golden approval cases for missing evidence, rejected approval, approved paper-only candidate, and prohibited live-execution promotion.

## Last Updated

2026-06-24
