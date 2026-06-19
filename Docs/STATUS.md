# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 1 Status

- **Current completed phase**: Phase 1 Decision Core.
- **Phase source**: `Docs/PHASE_CONTRACTS/01_DECISION_CORE.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: structured Event, Decision, Scores, EvidenceBundle, deterministic evaluator v1, and golden decision runner.
- **Tested**: FTSE/oil/labour conflict returns `NO_TRADE`; `NO_TRADE` is valid and not an error.
- **Next implementation phase**: Phase 2 Event Intelligence.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 1 does not implement Swing Brain, paper trading, broker execution, frontend UI, or LLM calls inside Decision Core.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Phase 2 contract**: `Docs/PHASE_CONTRACTS/02_EVENT_INTELLIGENCE.md`.
- **Golden fixtures**: `tests/golden/events/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
