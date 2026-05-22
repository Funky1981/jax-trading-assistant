# Add Baseline Analytics Events

## Summary

- Priority: P1
- Phase: Phase 1
- Estimate: 1-2d
- Outcome: measurable rollout

## Objective

Instrument the redesigned workflow so adoption, comprehension, and drop-off can be measured.

## In-scope touchpoints

- Frontend analytics layer
- Page event hooks for Home, AI Trading, Notifications, Approvals, and Research

## Implementation notes

- Add events only through the repo's existing analytics/event abstraction if one exists.
- Instrument baseline events: `ai_scanner_enabled`, `sentiment_settings_opened`, `opportunity_sentiment_viewed`, `sentiment_alert_opened`, `approval_sentiment_evidence_viewed`, `backtest_sentiment_enabled`, and `teach_me_sentiment_opened`.
- Include stable entity metadata such as source surface, Opportunity ID, route type, and sentiment mode when available.
- Avoid logging raw source article text, private notes, credentials, or broker-sensitive details.

## Acceptance criteria

- Key user actions emit analytics through a single abstraction.
- Events are stable and documented in code or nearby documentation.
- No sensitive payloads are emitted.
- Missing analytics transport does not break user workflows.

## Suggested validation

- Run targeted tests or spies for analytics event emission.
- Run type checks for event payload definitions.
- Manually verify events in local debug logging if the repo provides it.

## Dependencies

- Best implemented after `02-ai-trading-opportunity-feed.md` and `04-scanner-sentiment-controls.md` define surfaces.
