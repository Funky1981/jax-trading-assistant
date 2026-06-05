# Add Sentiment Alert Rules and Inbox Categories

## Summary

- Priority: P1
- Phase: Phase 2
- Estimate: 2-4d
- Outcome: better alerts

## Objective

Generate sentiment-aware notification events while keeping delivery restrained and user-configurable.

## In-scope touchpoints

- Notifications API
- Frontend notification inbox
- Sentiment aggregate and Opportunity scoring paths

## Implementation notes

- Add event types for sentiment conviction boost, sentiment flip, watched idea invalidation, price/sentiment divergence, and stale-source warning.
- Respect scanner settings and alert preferences when generating sentiment alerts.
- Store events durably in `notification_events`.
- Include `sentimentTriggerType`, entity type, entity ID, route, delivery channels, created time, and read time.
- Avoid desktop/web-push delivery until `17-desktop-web-push-mobile-preferences.md`.

## Acceptance criteria

- Sentiment-triggered alerts are delivered only when configured rules are met.
- In-app inbox can filter or label sentiment-related events.
- Notification click routes to the relevant Opportunity or watched idea.
- Duplicate or noisy alerts are suppressed by event identity or cooldown logic.

## Suggested validation

- Run backend tests for rule matching, suppression, and event persistence.
- Run frontend tests for inbox categories and routing.
- Run replay/golden checks if sentiment alerts are tied to behavior-sensitive scanner scoring.

## Dependencies

- Depends on `10-sentiment-ingest-scoring-aggregates.md`.
- Builds on `06-notification-centre-inbox.md`.
