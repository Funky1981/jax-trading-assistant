# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Implement Phase 5 Research/Backtest Evidence after Phase 4 Risk Veto.

## Active Phase

Phase 5 - Research/Backtest Evidence

## Active Work Items

- Define deterministic research/backtest evidence inputs and outputs for setup families.
- Preserve `NO_TRADE` as the default outcome.
- Add deterministic evidence tests before any paper approval workflow.

## Do Not Work On

- Live trading, broker execution, auto execution, unattended trading, and day trading.
- Paper ticket creation unless explicitly phased.
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
- Phase 5 Research/Backtest Evidence is the next implementation phase.

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Research/backtest evidence could accidentally imply strategy promotion without paper proof; keep it evidence-only.
- Golden coverage must protect Swing Brain and Risk Veto outputs from bypassing evidence requirements.

## Next 3 Actions

1. Read the Phase 5 Research/Backtest Evidence contract.
2. Define deterministic evidence bundle inputs and outputs on top of Swing Brain and Risk Veto results.
3. Add golden evidence cases for missing backtest evidence, weak research support, acceptable paper-only candidate evidence, and prohibited live-execution promotion.

## Last Updated

2026-06-24
