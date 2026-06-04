# ETF News Research Platform

Status: complete on `redesign`. Evidence: `PHASE_01_ETF_NEWS_COMPLETION.md`.

## Purpose

Turn Jax into an ETF-only, news-driven research and paper-trading assistant.

Target flow:

```text
news event
  ↓
normalise/store
  ↓
map to ETF
  ↓
compare historic ETF moves
  ↓
priced-in check
  ↓
confounder check
  ↓
evidence bundle
  ↓
AI summary
  ↓
human approval
  ↓
paper trade only
```

## Docs

- `01-etf-only-hardening.md`
- `02-database-event-study-schema.md`
- `03-data-providers-and-ingestion.md`
- `04-historical-backfill-pipeline.md`
- `05-etf-event-classification.md`
- `06-priced-in-engine.md`
- `07-research-evidence-bundles.md`
- `08-ai-guardrails.md`
- `09-strategy-integration.md`
- `10-mobile-approval-flow.md`
- `11-beginner-ux.md`
- `12-testing-uat-readiness.md`
- `13-production-live-feed-roadmap.md`
- `14-codex-prompts.md`
