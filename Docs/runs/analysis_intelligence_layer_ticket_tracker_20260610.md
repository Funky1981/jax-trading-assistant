# Analysis Intelligence Layer Ticket Tracker

Date: 2026-06-10
Plan: Docs/plans/analysis-intelligence-layer/09_PHASED_CODEX_TICKETS.md
Branch: redesign

## Status Legend

- not_started
- in_progress
- done
- blocked

## Ticket Board

| Ticket | Title | Status | Owner | Acceptance Gate | Evidence |
| --- | --- | --- | --- | --- | --- |
| 01 | Technical Analysis Snapshot Model | done | unassigned | storage + model + validation + tests | migration 000037 + macroevents technical model/service/tests |
| 02 | Technical Analysis Engine | done | unassigned | deterministic TA checks implemented | technical service BuildAndSave + macroevents tests |
| 03 | Fundamental Analysis Snapshot Model | done | unassigned | storage + model + validation + tests | fundamental migration + model/service/tests + store persistence |
| 04 | Fundamental Analysis Engine | not_started | unassigned | deterministic FA checks implemented | pending |
| 05 | Event Playbook Library | done | unassigned | event-to-playbook mapping + allowlist gates | playbook catalog + scenario enrichment + tests |
| 06 | Analyst Scoring Service | done | unassigned | combined score + hard veto overrides | analyst scoring model + service + persistence + tests |
| 07 | Multi-Analyst Review | done | unassigned | role outputs + linked final decision | deterministic review orchestration + persistence + tests |
| 08 | Analyst Memory | done | unassigned | case study persistence + retrieval | case study + feedback persistence + similarity retrieval + tests |
| 09 | Evidence Bundle Integration | done | unassigned | TA/FA sections included in evidence | TA/FA + scoring/veto + similar-case sections integrated in evidence bundle |
| 10 | Backtesting and UAT Fixtures | done | unassigned | deterministic fixture coverage + no broker writes | deterministic TA/FA fixture pipeline + confounder/missing-data/no-broker-write assertions |

## Completed in this run

- Created canonical migration files for Ticket 01:
  - db/postgres/migrations/000037_technical_analysis_snapshots.up.sql
  - db/postgres/migrations/000037_technical_analysis_snapshots.down.sql
- Added migration schema test:
  - db/postgres/migrations/technical_analysis_snapshots_schema_test.go
- Added deterministic technical snapshot model, scoring, and service:
  - internal/modules/macroevents/technical.go
  - internal/modules/macroevents/technical_service.go
- Added tests for scoring and fail-safe behavior:
  - internal/modules/macroevents/technical_test.go
  - internal/modules/macroevents/technical_service_test.go
- Added persistence wiring in macroevents store:
  - internal/modules/macroevents/store.go
- Added deterministic technical engine path from candles:
  - internal/modules/macroevents/technical_service.go (BuildAndSave)
- Added Ticket 02 acceptance-focused engine tests:
  - internal/modules/macroevents/technical_service_test.go
- Added Ticket 03 fundamental snapshot model, service, storage, and tests:
  - db/postgres/migrations/000038_fundamental_analysis_snapshots.up.sql
  - db/postgres/migrations/000038_fundamental_analysis_snapshots.down.sql
  - db/postgres/migrations/fundamental_analysis_snapshots_schema_test.go
  - internal/modules/macroevents/fundamental.go
  - internal/modules/macroevents/fundamental_service.go
  - internal/modules/macroevents/fundamental_test.go
  - internal/modules/macroevents/fundamental_service_test.go
- Exposed technical analysis snapshots in the macro event detail API:
  - cmd/trader/macro_api.go
  - cmd/trader/macro_api_test.go
- Added deterministic event playbook catalog and scenario enrichment:
  - internal/modules/macroevents/playbook.go
  - internal/modules/macroevents/playbook_test.go
  - internal/modules/macroevents/scenario.go
  - internal/modules/macroevents/scenario_test.go
- Added analyst scoring service and persistence:
  - db/postgres/migrations/000039_analyst_decisions.up.sql
  - db/postgres/migrations/000039_analyst_decisions.down.sql
  - db/postgres/migrations/analyst_decisions_schema_test.go
  - internal/modules/macroevents/analyst.go
  - internal/modules/macroevents/analyst_test.go
  - internal/modules/macroevents/analyst_service_test.go
- Added multi-analyst review orchestration and persistence:
  - db/postgres/migrations/000040_multi_analyst_reviews.up.sql
  - db/postgres/migrations/000040_multi_analyst_reviews.down.sql
  - db/postgres/migrations/multi_analyst_reviews_schema_test.go
  - internal/modules/macroevents/review.go
  - internal/modules/macroevents/review_test.go
  - internal/modules/macroevents/review_service_test.go
- Added analyst memory and feedback loop:
  - db/postgres/migrations/000041_analyst_memory_case_studies.up.sql
  - db/postgres/migrations/000041_analyst_memory_case_studies.down.sql
  - db/postgres/migrations/analyst_memory_case_studies_schema_test.go
  - internal/modules/macroevents/memory.go
  - internal/modules/macroevents/memory_test.go
  - internal/modules/macroevents/memory_service_test.go
- Added Ticket 09 evidence bundle integration for TA/FA and analyst outputs:
  - internal/modules/macroevents/evidence.go
  - internal/modules/macroevents/evidence_test.go
- Added Ticket 10 deterministic TA/FA backtesting fixtures:
  - internal/modules/macroevents/uat_test.go

## Verification Commands

- go test ./db/postgres/migrations
- go test ./internal/modules/macroevents
- go test ./cmd/trader

## Notes

- Ticket 01 currently covers deterministic scoring, verdict/reason generation, and missing-candle fail-safe behavior.
- API/UI exposure for technical snapshots is intentionally deferred to Ticket 09 integration scope.
