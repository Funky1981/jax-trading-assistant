# Project Status

## Snapshot

- **Active runtime layout**: `cmd/trader` serves deterministic runtime plus frontend API surface; `cmd/research` serves orchestration/research and memory tools.
- **External service boundaries retained**: `services/ib-bridge` and `services/agent0-service`.
- **Core stack target**: `jax-trader` (8081/8100), `jax-research` (8091), `ib-bridge` (8092), `agent0-service` (8093), frontend dev server (5173).
- **Product truth**: Jax is an event-driven trading research assistant. See `Docs/JAX_PRODUCT_CHARTER.md`.
- **Capability truth**: current capability status lives in `Docs/CAPABILITY_MATRIX.md`.

## Phase 15 Status

- **Current completed phase**: Phase 15 Review Operations Internal Access Wiring.
- **Phase source**: attached Phase 15 Review Operations Internal Access Wiring brief.
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
- **Implemented in Phase 13**: deterministic review workflow selection, review batch generation, review packet generation, JSON export, Markdown export, and read-only follow-up action export under `internal/decisioning/workflow` and `internal/decisioning/export`.
- **Tested in Phase 13**: workflow/export unit tests plus golden workflow/export cases cover active item filtering, critical/high/due/overdue ordering, packet safety fields, deterministic JSON and Markdown exports, source-record non-mutation, hidden-reasoning exclusion, live-ready blocking warnings, forbidden action preservation, human approval requirements, and no-auto-apply behavior.
- **Implemented in Phase 14**: deterministic operator-facing service methods and read models under `internal/decisioning/operator` and `internal/decisioning/readmodel`, with review-operation integration kept record-only under `internal/decisioning/operations`.
- **Tested in Phase 14**: operator/read-model unit tests plus golden operator/read-model cases cover queue summaries, triage item details, follow-up action details, batch/export summaries, accept/reject/defer/request-more-evidence/close workflows, no-auto-apply behavior, live-ready/live-order blocking, record-only mutation, forbidden action preservation, and human approval requirements.
- **Implemented in Phase 15**: internal review operations application service wiring under `internal/decisioning/app`, connecting the operations repository, operator service, workflow batches, JSON/Markdown exporters, read models, and safe human feedback actions behind one internal entry point.
- **Tested in Phase 15**: app service unit tests plus golden app/access cases cover construction with in-memory dependencies, queue/detail reads, deterministic batch building, JSON/Markdown exports, accept/reject/defer/request-more-evidence/close workflows, read-only query/export behavior, record-only action mutation, forbidden action preservation, no-auto-apply behavior, human approval requirements, and live/order/execution/broker blocking.
- **Next focus**: decide whether a later phase should add a narrow internal CLI or handler adapter around the Phase 15 app service; do not add frontend UI, broker execution, live trading, automatic rule changes, or paper execution.

## Explicit Exclusions

- Day trading is `NOT_PLANNED`.
- Live trading is `NOT_PLANNED`.
- Auto execution, broker order placement, unattended trading, and live capital are outside the current roadmap.
- Phase 15 does not implement paper execution, broker execution, frontend UI, live trading, auto execution, options trading, day trading, live order creation, or automatic strategy/scoring/risk rule changes.

## Next Focus

- **Roadmap**: `Docs/ROADMAP.md`.
- **Next target**: narrow internal CLI or handler adapter for review operations only if explicitly approved, preserving read-only queries and record-only manual state transitions.
- **Golden fixtures**: `tests/golden/events/`, `tests/golden/swing/`, `tests/golden/risk/`, `tests/golden/research/`, `tests/golden/paper/`, `tests/golden/review/`, `tests/golden/pipeline/`, `tests/golden/replay/`, `tests/golden/feedback/`, `tests/golden/triage/`, `tests/golden/operations/`, `tests/golden/workflow/`, `tests/golden/export/`, `tests/golden/operator/`, `tests/golden/readmodel/`, `tests/golden/app/`.
- **Project-management process**: `Docs/PROJECT_MANAGEMENT/`.
