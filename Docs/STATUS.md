# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 2 Status

- **Current completed phase**: Phase 2 Event Intelligence.
- **Phase source**: `Docs/PHASE_CONTRACTS/02_EVENT_INTELLIGENCE.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic event classification, driver extraction, conflict detection, affected-asset mapping, and enriched `core.Event` output for Decision Core.
- **Tested**: FTSE/oil/labour enriches to `MACRO_COMMODITY_INDEX_MOVE`, normalises oil/labour/rates/central-bank drivers, detects conflicts, maps FTSE100/BP/SHEL/GBP/UK_GILTS, and still returns `NO_TRADE` through Decision Core.
- **Next implementation phase**: Phase 3 Swing Brain v1.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 2 does not implement Swing Brain, paper trading, broker execution, frontend UI, LLM calls inside Event Intelligence, risk veto, paper ticket creation, live trading, auto execution, or day trading.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Phase 3 contract**: `Docs/PHASE_CONTRACTS/03_SWING_BRAIN_V1.md`.
- **Golden fixtures**: `tests/golden/events/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
