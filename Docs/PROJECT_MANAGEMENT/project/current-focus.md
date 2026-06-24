# Current Focus

This file tells humans and AI assistants what matters right now.

## Current Objective

Persist and observe deterministic decision pipeline results after Phase 8 integration.

## Active Phase

Persistence and Observability Hardening - Decision Pipeline Results

## Active Work Items

- Persist deterministic pipeline results, decision records, and review schedules.
- Add observability for pipeline status, validation warnings, and blocked gates.
- Preserve `NO_TRADE` as the default outcome.
- Keep persistence and observability separate from broker execution and live trading.

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

## Current Risks

Summarise current risks. Full details go in `/project/risks.md`.

- Persistence work could accidentally treat stored pipeline output as execution authority; keep Decision Core, Risk Veto, Research Evidence, Paper Approval, and Review boundaries explicit.
- Golden coverage must protect every deterministic decision gate from execution, live-order, and auto-approval drift.

## Next 3 Actions

1. Design persistence records for pipeline results and review schedules without adding execution authority.
2. Add tests that prove persisted records preserve no-trade, risk-veto, research, paper-approval, and review gates.
3. Add observability for blocked gates and validation warnings while preserving live-trading exclusions.

## Last Updated

2026-06-24
