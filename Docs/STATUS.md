# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 9 Status

- **Current completed phase**: Phase 9 Persistence and Observability Hardening.
- **Phase source**: attached Phase 9 implementation brief; Phase 8 source was the Decision Pipeline Integration and Hardening brief.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic end-to-end non-execution decision pipeline under `internal/decisioning/pipeline`.
- **Implemented in Phase 9**: persistence models, repository interface, deterministic in-memory repository, audit records, trace records, and structured observability summaries under `internal/decisioning/persistence` and `internal/decisioning/observability`.
- **Tested**: persistence tests cover saving/retrieving no-trade, risk-rejected, and paper-review-ready pipeline records, decision logs, review schedules, outcome reviews, audit records, pending review listing, and non-upgradeability. Observability tests cover no-trade and paper-review-ready summaries, warning/error counts, trace fields, and hidden-reasoning exclusion.
- **Next focus**: deterministic replay and memory feedback reporting for persisted decision/review records without adding broker execution or live trading.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 9 does not implement paper execution, broker execution, frontend UI, live trading, auto execution, options trading, day trading, live order creation, or automatic strategy/scoring/risk rule changes.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Next target**: deterministic replay and memory feedback reporting over persisted pipeline and review records.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`, `tests/golden/paper/`, `tests/golden/review/`, `tests/golden/pipeline/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
