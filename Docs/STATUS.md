# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 8 Status

- **Current completed phase**: Phase 8 Decision Pipeline Integration and Hardening.
- **Phase source**: attached Phase 8 implementation brief; Phase 7 source remains `Docs/PHASE_CONTRACTS/07_POST_DECISION_REVIEW.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic end-to-end non-execution decision pipeline under `internal/decisioning/pipeline`.
- **Tested**: pipeline golden cases cover FTSE/oil/labour no-trade, missing research evidence, promising research paper-review readiness, risk rejection, WATCH and NO_TRADE non-upgradeability, live account blocking, and missing portfolio warnings.
- **Next focus**: persistence and observability hardening for decision records, pipeline results, and review scheduling without adding broker execution or live trading.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 8 does not implement paper execution, broker execution, frontend UI, live trading, auto execution, options trading, day trading, live order creation, or automatic strategy/scoring/risk rule changes.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Next target**: persistence and observability hardening for pipeline and review records.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`, `tests/golden/paper/`, `tests/golden/review/`, `tests/golden/pipeline/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
