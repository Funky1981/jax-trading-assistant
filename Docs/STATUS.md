# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 3 Status

- **Current completed phase**: Phase 3 Swing Brain v1.
- **Phase source**: `Docs/PHASE_CONTRACTS/03_SWING_BRAIN_V1.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic Swing Brain v1 on top of existing Decision Core and Event Intelligence types, including setup family selection, catalyst checks, confirmation gates, invalidation requirements, risk/reward minimums, unresolved event-risk downgrades, structured explanations, and non-execution allowed/forbidden actions.
- **Tested**: FTSE/oil/labour cannot become `TRADE_CANDIDATE`; missing invalidation and poor risk/reward return `NO_TRADE`; missing confirmation returns `WATCH`; a fully confirmed setup can return `TRADE_CANDIDATE` while still forbidding live execution.
- **Next implementation phase**: Phase 4 Risk Veto.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 3 does not implement paper trading, broker execution, frontend UI, LLM calls inside Swing Brain, full Risk Veto, paper ticket creation, live trading, auto execution, options trading, or day trading.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Phase 4 target**: Risk Veto.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
