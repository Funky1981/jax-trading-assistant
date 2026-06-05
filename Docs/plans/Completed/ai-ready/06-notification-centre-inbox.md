# Build NotificationCentrePage V1 With In-App Inbox

## Summary

- Priority: P1
- Phase: Phase 1
- Estimate: 2-3d
- Outcome: durable alerts

## Objective

Add a durable in-app notification center so opportunities, approval events, and future sentiment alerts are recoverable after transient toasts disappear.

## In-scope touchpoints

- New frontend notification page
- Router and navigation shell
- Existing event aggregation endpoint or a minimal new read endpoint if required

## Implementation notes

- Add `NotificationCentrePage` as a top-level destination.
- Start with in-app inbox semantics only: unread/read state, event type, title, body, entity route, created time, and delivery channel labels.
- Include categories that can later support sentiment-triggered and sentiment-invalidated events.
- Do not request desktop notification permission in Phase 1.
- Route notification clicks to the relevant Opportunity, approval, analysis, or settings surface.

## Acceptance criteria

- Notifications have a durable in-app list.
- Users can identify unread items and mark items read if supported by the backing model.
- Notification entries include a clear destination or recovery path.
- Desktop/web-push permission is not requested prematurely.

## Suggested validation

- Run targeted page and routing tests using fixture events.
- Run frontend type checks for notification event contracts.
- Manually verify empty, unread, read, and stale event states.

## Dependencies

- Later expanded by `13-sentiment-alert-rules-inbox-categories.md` and `17-desktop-web-push-mobile-preferences.md`.
