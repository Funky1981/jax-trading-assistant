# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Implement Phase 4 Risk Veto after Phase 3 Swing Brain v1.

## Active Phase

Phase 4 - Risk Veto

## Active Work Items

- Define deterministic risk veto inputs and downgrade/rejection outputs for Swing Brain candidates.
- Preserve `NO_TRADE` as the default outcome.
- Add deterministic risk veto tests before any paper approval workflow.

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
- Phase 4 Risk Veto is the next implementation phase.

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Risk Veto could accidentally create paper tickets or execution-shaped outputs; keep it as rejection/downgrade logic only.
- Golden coverage must protect Swing Brain candidates from bypassing risk rejection.

## Next 3 Actions

1. Write the Phase 4 Risk Veto contract.
2. Define deterministic risk veto inputs and outputs on top of Swing Brain candidates.
3. Add golden risk cases for poor risk/reward, missing stop, unresolved event risk, concentrated exposure, and live execution requests.

## Last Updated

2026-06-24
