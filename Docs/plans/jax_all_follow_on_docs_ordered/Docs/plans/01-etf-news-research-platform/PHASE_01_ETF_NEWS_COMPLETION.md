# ETF News Research Platform Completion

Date: 2026-06-01
Branch: `redesign`

## Scope

This checkpoint completes the `01-etf-news-research-platform` folder from the ordered follow-on docs. The pass verified the existing implementation against all 14 plan files and did not add live trading, options, futures, leveraged ETFs, inverse ETFs, volatility products, autonomous broker submission, n8n automation, or paid production feed work.

## Completion Matrix

| Plan | Status | Evidence |
| --- | --- | --- |
| `01-etf-only-hardening.md` | Complete | ETF source of truth exists in `config/etf-instruments.json`; policy tests pass in `internal/modules/instruments`; manual ETF broker-write reroute tests pass in `cmd/trader`. |
| `02-database-event-study-schema.md` | Complete | Migrations `000022_event_study_schema`, `000023_priced_in_engine`, and schema tests cover event windows, confounders, priced-in scores, ETF context snapshots, and research summaries. |
| `03-data-providers-and-ingestion.md` | Complete for development scope | `cmd/research/backfill.go` preserves provider/source metadata and writes provider-aware summaries; production paid feed expansion remains intentionally deferred. |
| `04-historical-backfill-pipeline.md` | Complete for initial pipeline | Research backfill tests cover allowlist enforcement, event-study writes, evidence bundles, priced-in scoring, and idempotent upsert behavior. |
| `05-etf-event-classification.md` | Complete | Rule-based classifier exists in `internal/modules/etfnews`; tests verify ETF mappings for AI/chips, oil, rates/inflation, banking stress, and unknown/no-trade cases. |
| `06-priced-in-engine.md` | Complete | Priced-in scoring and hard-reject behavior are implemented in `cmd/research/backfill.go`; tests cover `priced_in` hard rejection and `not_priced_in` acceptance behavior. |
| `07-research-evidence-bundles.md` | Complete | Research evidence bundle generation is implemented in `cmd/research/backfill.go`; schema and backfill tests cover persisted summary/evidence shape. |
| `08-ai-guardrails.md` | Complete | AI guardrail validation is implemented in `cmd/research/ai_guardrails.go`; tests verify schema validation and guardrails overriding AI trade decisions. |
| `09-strategy-integration.md` | Complete | ETF news strategy types and trading mode catalog exist for market panic, sector momentum, and rates/bonds rotation; strategy and catalog tests pass. |
| `10-mobile-approval-flow.md` | Complete | Telegram webhook route, one-time token storage, expiry handling, and approval service tests exist in `internal/modules/approvals` and `cmd/trader`. |
| `11-beginner-ux.md` | Complete | ETF universe, strategy cards, candidate evidence, research timeline, and beginner mode toggle are wired in frontend routes and targeted tests pass. |
| `12-testing-uat-readiness.md` | Complete for repository validation | Targeted backend, schema, approvals, and frontend tests pass; paper/live production proof remains a later operational phase. |
| `13-production-live-feed-roadmap.md` | Complete as roadmap | Production provider requirements are documented; paid production feeds remain deferred until paper-trading evidence justifies them. |
| `14-codex-prompts.md` | Complete | Prompt scope has been executed or verified by the implementation and evidence above. |

## Verification Run

```text
go test ./internal/modules/instruments ./cmd/trader ./db/postgres/migrations -run 'ETF|Event|Priced|Evidence|Mobile|Approval|Schema|AI|Backfill' -count=1
go test ./cmd/research ./internal/modules/etfnews ./libs/strategytypes ./internal/modules/tradingmodes ./internal/modules/approvals -count=1
npm test -- src/pages/Step9BeginnerPages.test.tsx src/pages/ApprovalsPage.test.tsx src/pages/MobileApprovalHarnessPage.test.tsx src/pages/TestingPage.test.tsx src/components/trading/ScannerSettingsCard.test.tsx --run
```

## Result

All targeted verification passed. The screenshot folder is complete on `redesign`.

## Remaining Non-Blockers

- Production live feed hardening, paid Polygon/Massive primary feeds, and long paper-trading proof are intentionally later phases.
- n8n automation is intentionally later and was not added here.
- Live trading remains disabled and out of scope.
