# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 12 Status

- **Current completed phase**: Phase 12 Review Operations Persistence and Reporting.
- **Phase source**: attached Phase 12 Review Operations Persistence and Reporting brief; Phase 11 source was the Review Operations and Human Feedback Triage brief.
- **Default decision**: `NO_TRADE`.
- **Implemented**: deterministic end-to-end non-execution decision pipeline under `internal/decisioning/pipeline`.
- **Implemented in Phase 9**: persistence models, repository interface, deterministic in-memory repository, audit records, trace records, and structured observability summaries under `internal/decisioning/persistence` and `internal/decisioning/observability`.
- **Tested**: persistence tests cover saving/retrieving no-trade, risk-rejected, and paper-review-ready pipeline records, decision logs, review schedules, outcome reviews, audit records, pending review listing, and non-upgradeability. Observability tests cover no-trade and paper-review-ready summaries, warning/error counts, trace fields, and hidden-reasoning exclusion.
- **Implemented in Phase 10**: deterministic replay and memory feedback reporting under `internal/decisioning/replay` and `internal/decisioning/feedback` over supplied persisted-style decision, review, risk, research, paper, and pipeline records.
- **Tested in Phase 10**: replay and feedback unit tests plus golden reporting cases cover no-trades, missed opportunities, avoided losses, risk veto helped/too strict, paper setup worked/failed, weak or missing research evidence, human-only suggestions, blocked live promotion, and preserved forbidden actions.
- **Implemented in Phase 11**: deterministic triage items, priority/status validation, human feedback decisions, review operation results, and follow-up action records under `internal/decisioning/triage` and `internal/decisioning/operations`.
- **Tested in Phase 11**: unit and golden triage/operations cases cover missed opportunities, risk-veto-too-strict findings, failed paper setups, research gaps, scoring reviews, rejected suggestions, more-evidence requests, critical priority no-auto-apply behavior, live-ready promotion blocking, and close-with-no-action decisions.
- **Implemented in Phase 12**: deterministic review operations repository, triage-item persistence, human feedback decision persistence, follow-up action persistence, operation audit records, open/high-priority/due queue listing, and review operations reporting under `internal/decisioning/operations`.
- **Tested in Phase 12**: repository/reporting unit tests plus golden persistence/reporting cases cover saved/retrieved triage items, feedback decisions, follow-up actions, audit records, open/high-priority/due listings, rejected and needs-more-evidence reporting, auto-apply blocking, live-ready blocking, forbidden action preservation, and human approval requirements.
- **Next focus**: integrate persisted review operations into operator-facing backend workflows and export contracts without adding frontend UI, broker execution, live trading, automatic rule changes, or paper execution.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 12 does not implement paper execution, broker execution, frontend UI, live trading, auto execution, options trading, day trading, live order creation, or automatic strategy/scoring/risk rule changes.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Next target**: backend workflow/export integration for persisted review-operation queues and reports.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`, `tests/golden/paper/`, `tests/golden/review/`, `tests/golden/pipeline/`, `tests/golden/replay/`, `tests/golden/feedback/`, `tests/golden/triage/`, `tests/golden/operations/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
