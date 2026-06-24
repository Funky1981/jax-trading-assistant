# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 4 Status

- **Current completed phase**: Phase 4 Risk Veto.
- **Phase source**: `Docs/PHASE_CONTRACTS/04_RISK_VETO.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic Risk Veto layer for Swing Brain decisions, including structured risk assessment output, portfolio context, exposure thresholds, risk/reward and invalidation checks, paper-only/human-approval requirements, live account rejection, and mandatory forbidden execution actions.
- **Tested**: valid Swing candidate can pass risk; poor risk/reward, missing stop/invalidation, and live account mode are rejected; high sector and correlated exposure downgrade to `WATCH`; existing `WATCH` and `NO_TRADE` decisions cannot be upgraded.
- **Next implementation phase**: Phase 5 Research/Backtest Evidence.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 4 does not implement paper trading, broker execution, frontend UI, paper ticket creation, live trading, auto execution, options trading, day trading, or strategy promotion from backtest evidence.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Phase 5 target**: Research/Backtest Evidence.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
