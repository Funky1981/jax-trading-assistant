# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 6 Status

- **Current completed phase**: Phase 6 Paper Approval Loop.
- **Phase source**: `Docs/PHASE_CONTRACTS/06_PAPER_APPROVAL_LOOP.md`.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic paper ticket, approval status, lifecycle, ticket validation, and explicit human approval models under `internal/decisioning/paper`.
- **Tested**: valid risk-approved trade candidate creates a pending paper ticket; `NO_TRADE`, `WATCH`, risk-rejected candidates, missing invalidation, and poor risk/reward cannot create tickets; expired, rejected, deferred, and auto approvals cannot approve; valid pending tickets can be approved for paper.
- **Next implementation phase**: Phase 7 Post Decision Review.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 6 does not implement paper execution, broker execution, frontend UI, live trading, auto execution, options trading, day trading, or live order creation.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Phase 7 target**: Post Decision Review.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`, `tests/golden/paper/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
