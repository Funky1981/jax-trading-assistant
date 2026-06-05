# Add Desktop, Web-Push, and Mobile Channel Preferences

## Summary

- Priority: P2
- Phase: Phase 3
- Estimate: 4-6d
- Outcome: multi-channel alerts

## Objective

Extend notifications beyond the in-app inbox with user-controlled channel preferences and permission-safe opt-in flows.

## In-scope touchpoints

- Notification preferences
- Service worker or web-push integration
- Mobile channel routing

## Implementation notes

- Add preferences for in-app, desktop/web-push, and mobile/Telegram-style channels.
- Request browser notification permission only after clear user intent and explanation.
- Provide recovery paths if permission is denied.
- Store channel preferences in `notification_preferences`.
- Support sentiment-specific alert rules without making them mandatory.

## Acceptance criteria

- Users can manage desktop/mobile channel preferences in-product.
- Browser permission prompts are user-initiated, not shown on first page load.
- Denied permissions do not block in-app notifications.
- Sentiment alert preferences can be adjusted by channel.

## Suggested validation

- Run frontend tests for preference states and permission prompt flow.
- Run backend tests for preference persistence and channel routing.
- Manually verify browser permission behavior in supported browsers.

## Dependencies

- Builds on `06-notification-centre-inbox.md` and `13-sentiment-alert-rules-inbox-categories.md`.
