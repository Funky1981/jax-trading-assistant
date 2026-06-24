# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 7 Status

- **Current completed phase**: Phase 7 Post Decision Review.
- **Phase source**: `Docs/PHASE_CONTRACTS/07_POST_DECISION_REVIEW.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic decision log, review schedule, outcome review, lesson, and validation models under `internal/decisioning/review`.
- **Tested**: `NO_TRADE`, `WATCH`, `SETUP_FORMING`, `TRADE_CANDIDATE`, `REJECTED_BY_RISK`, and `APPROVED_FOR_PAPER` decisions schedule reviews; correct no-trades, missed opportunities, avoided losses, risk-veto strictness, and paper worked/failed outcomes can be recorded; lesson suggestions require human approval and `LIVE_READY` promotion is blocked.
- **Next focus**: integration and hardening of the decision pipeline.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 7 does not implement paper execution, broker execution, frontend UI, live trading, auto execution, options trading, day trading, live order creation, or automatic strategy/scoring/risk rule changes.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Next target**: integration and hardening of the decision pipeline.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`, `tests/golden/paper/`, `tests/golden/review/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
