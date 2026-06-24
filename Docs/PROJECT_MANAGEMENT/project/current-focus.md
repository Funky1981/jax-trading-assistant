# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Implement Phase 3 Swing Brain v1 after Phase 2 Event Intelligence.

## Active Phase

Phase 3 - Swing Brain v1

## Active Work Items

- Build swing setup evaluation on top of enriched Event Intelligence and Decision Core outputs.
- Preserve `NO_TRADE` as the default outcome.
- Add deterministic tests and golden cases before any paper approval workflow.

## Do Not Work On

- Live trading, broker execution, auto execution, unattended trading, and day trading.
- Paper ticket creation and risk veto implementation unless explicitly phased.
- Frontend UI work unless a later phase calls for it.

## Current Constraints

- Research and paper-trading direction only; no live capital paths.
- Event Intelligence is deterministic and must not rely on LLM calls as source of truth.
- Decision Core remains the owner of final decisions.

## Latest Decisions

Summarise recent decisions. Full details go in `/project/decisions.md`.

- Phase 2 Event Intelligence is implemented and tested in `internal/decisioning/classify`.
- Phase 3 Swing Brain v1 is the next implementation phase.

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Swing Brain could accidentally duplicate Event Intelligence or Decision Core ownership; keep boundaries explicit.
- Golden coverage must expand before promoting any setup family.

## Next 3 Actions

1. Read `Docs/PHASE_CONTRACTS/03_SWING_BRAIN_V1.md`.
2. Define deterministic swing setup inputs from enriched events and existing Decision Core types.
3. Add golden swing cases that preserve common `NO_TRADE` outcomes.

## Last Updated

2026-06-19
