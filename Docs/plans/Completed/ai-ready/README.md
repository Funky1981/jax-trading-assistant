# AI-Ready Trading Assistant Plan

This plan set turns the deep research report into implementation-ready tickets for a beginner-friendly AI Trading experience with sentiment as a first-class evidence layer. The original research source is copied here as `deep-research-report.md`.

## Source

- Research report: `deep-research-report.md`
- Archived original source path: `Docs/archive/reports/deep-research-report.md`

## Target state

Jax should expose one coherent AI Trading workflow where scanner state is visible, opportunities are explained in plain language, sentiment contributes transparent evidence, approvals stay policy-safe, notifications are durable, and research/backtests support sentiment without forcing raw JSON.

## Plan structure

| File | Priority | Phase | Estimate | Outcome |
|---|---|---|---:|---|
| `00-design-system-foundation.md` | P0 | Phase 0 | 2-4d | consistent redesign foundation |
| `01-homepage-simplified-nav.md` | P0 | Phase 1 | 2-3d | clear first-run IA |
| `02-ai-trading-opportunity-feed.md` | P0 | Phase 1 | 3-4d | dedicated AI home |
| `03-opportunity-adapter.md` | P0 | Phase 1 | 2-4d | one user-facing object |
| `04-scanner-sentiment-controls.md` | P0 | Phase 1 | 2-3d | visible scanner and sentiment config |
| `05-manual-etf-policy-reroute.md` | P0 | Phase 1 | 1-2d | fewer dead ends |
| `06-notification-centre-inbox.md` | P1 | Phase 1 | 2-3d | durable alerts |
| `07-research-wizard-v1.md` | P1 | Phase 1 | 3-5d | beginner backtests |
| `08-baseline-analytics-events.md` | P1 | Phase 1 | 1-2d | measurable rollout |
| `09-ai-overview-scanner-api.md` | P0 | Phase 2 | 3-5d | AI home data model |
| `10-sentiment-ingest-scoring-aggregates.md` | P0 | Phase 2 | 5-8d | sentiment feature layer |
| `11-opportunity-sentiment-api-fields.md` | P0 | Phase 2 | 3-5d | explainable sentiment |
| `12-opportunity-drawer-sentiment-explanation.md` | P0 | Phase 2 | 3-4d | unified evidence UX |
| `13-sentiment-alert-rules-inbox-categories.md` | P1 | Phase 2 | 2-4d | better alerts |
| `14-approval-sentiment-evidence-pack.md` | P1 | Phase 2 | 2-3d | policy-safe evidence |
| `15-research-backtest-sentiment-options.md` | P1 | Phase 2 | 3-5d | testable sentiment strategy |
| `16-save-paper-live-sentiment-handoff.md` | P1 | Phase 2 | 2-3d | live/paper conversion |
| `17-desktop-web-push-mobile-preferences.md` | P2 | Phase 3 | 4-6d | multi-channel alerts |
| `18-model-card-evidence-limitations.md` | P2 | Phase 3 | 2-4d | transparency maturity |
| `19-override-reason-calibration-reporting.md` | P2 | Phase 3 | 3-5d | trust and governance |
| `20-hybrid-sentiment-provider-abstraction.md` | P2 | Phase 3 | 3-5d | infra flexibility |

## Recommended work order

1. Land `00-design-system-foundation.md` first so redesign screens share one token system, component base, and Storybook workflow.
2. Land Phase 1 UX and IA tickets so users have a coherent entry point before backend sentiment is complete.
3. Land Phase 2 backend/read-model tickets, then connect sentiment evidence into the drawer, approvals, alerts, and research.
4. Land Phase 3 maturity tickets after the core workflow has usage data and validated evidence semantics.

## Shared metrics

- Time to first AI scan enabled
- Time to first opportunity reviewed
- Time to first sentiment-enabled scan configured
- Opportunity detail open rate
- Sentiment evidence viewed rate
- Sentiment-triggered alert open rate
- Approval decision time for sentiment-enriched opportunities
- Manual ETF reroute completion rate
- Backtest start and completion rates
- Backtest-to-paper conversion for sentiment-enabled strategies
- Opportunity comprehension score
- Sentiment comprehension score
- Override reason capture rate
- Rate of opportunities missing sentiment evidence due to sparse sources
- Rate of sentiment-triggered alerts dismissed without review

## Shared analytics events

- `ai_scanner_enabled`
- `sentiment_settings_opened`
- `sentiment_threshold_changed`
- `sentiment_source_scope_changed`
- `sentiment_window_changed`
- `opportunity_sentiment_viewed`
- `opportunity_sentiment_source_clicked`
- `sentiment_alert_received`
- `sentiment_alert_opened`
- `sentiment_flip_reviewed`
- `approval_sentiment_evidence_viewed`
- `approval_override_reason_selected`
- `backtest_sentiment_enabled`
- `backtest_sentiment_mode_changed`
- `backtest_result_sentiment_summary_viewed`
- `paper_setup_saved_with_sentiment`
- `teach_me_sentiment_opened`

## Phase acceptance criteria

### Phase 1

- Beginner-visible nav is reduced to the new top-level structure.
- Home explains the product in one sentence and shows three clear starting actions.
- AI Trading exists as a dedicated route.
- Scanner settings visibly include sentiment scope, time window, and threshold placeholders.
- No manual ETF route ends without a next-step CTA.
- Notifications have a durable in-app inbox.
- Research has a no-JSON guided path.

### Phase 2

- AI opportunities include sentiment summary and sentiment evidence.
- Opportunity detail shows sentiment score, time window, source count, top drivers, and limitations.
- Sentiment-triggered alerts are delivered only when configured rules are met.
- Approval-required opportunities show sentiment evidence but still require human action.
- Backtests can run with sentiment disabled, sentiment as filter, or sentiment as boost.
- Users can save a sentiment-enabled result as a paper-ready setup.

### Phase 3

- Desktop/mobile channel preferences can be managed in-product.
- Every opportunity detail includes model-card-style limitation and intended-use cues.
- Users can understand why sentiment affected an opportunity without reading raw scores alone.
- At least 80% of novice test users can answer both "what does sentiment mean here?" and "what do I do next?"
- The system records why humans overruled sentiment-enriched opportunities.

## Assumptions

- `Docs/plans` is the canonical documentation plan path; the requested `Doc/plans` path is treated as a typo.
- The original research file remains in place and is copied into this bundle.
- Sentiment is an evidence family, not an autonomous trading trigger.
- Sentiment must never bypass approval policy or replace human approval in approval-gated flows.
- Beginner defaults are conservative: trusted news, short-to-medium time windows, and sentiment as boost/filter rather than sole trigger.
- Hybrid sentiment is the preferred implementation direction: provider abstraction plus internal aggregation, routing, and explainability.
