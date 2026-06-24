# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 5 Status

- **Current completed phase**: Phase 5 Research/Backtest Evidence.
- **Phase source**: `Docs/PHASE_CONTRACTS/05_RESEARCH_BACKTEST_EVIDENCE.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic research hypothesis, backtest evidence, dataset integrity, validation result, and promotion-cap models under `internal/decisioning/research`.
- **Tested**: missing dataset hash, missing slippage/costs, missing out-of-sample evidence, weak sample size, missing failure modes, promising evidence, paper-ready evidence, and attempted `LIVE_READY` promotion.
- **Next implementation phase**: Phase 6 Paper Approval Loop.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 5 does not implement paper ticket creation, broker execution, frontend UI, live trading, auto execution, options trading, day trading, or a full backtest engine.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Phase 6 target**: Paper Approval Loop.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
