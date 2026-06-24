# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Integrate and harden the deterministic decision pipeline after Phase 7 Post Decision Review.

## Active Phase

Integration and Hardening - Decision Pipeline

## Active Work Items

- Wire deterministic decision outputs, research evidence, paper approval, and post-decision review into a cohesive pipeline.
- Preserve `NO_TRADE` as the default outcome.
- Keep integration and hardening separate from broker execution and live trading.

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

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Integration work could accidentally bypass phase gates; keep Decision Core, Risk Veto, Research Evidence, Paper Approval, and Review boundaries explicit.
- Golden coverage must protect every deterministic decision gate from execution, live-order, and auto-approval drift.

## Next 3 Actions

1. Map the deterministic pipeline from Decision Core through Review without adding execution paths.
2. Add integration tests that preserve no-trade, risk-veto, research, paper-approval, and review gates.
3. Harden persistence/observability for decision records and reviews while preserving live-trading exclusions.

## Last Updated

2026-06-24
