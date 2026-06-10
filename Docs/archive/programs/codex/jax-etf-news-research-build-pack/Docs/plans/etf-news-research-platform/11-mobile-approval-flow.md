# 11 — Mobile Approval Flow

## Goal

Send fast, beginner-friendly ETF paper-trade suggestions to mobile for approval.

## Recommended Phase-One Channel

Start with:

```text
Telegram bot
```

Reasons:

- cheap/free
- easy to test
- fast mobile notifications
- approve/reject buttons possible
- no full mobile app required yet

Alternatives:

```text
Pushover
Discord private channel
email fallback
React Native / Expo later
Firebase Cloud Messaging later
```

## Notification Format

```text
ETF: SMH
Strategy: Sector News Momentum
Action: Paper Buy
Confidence: 72%

Why:
AI chip news is moving semiconductors. SMH and SOXX both confirmed.

Priced-in check:
Partially priced in, still acceptable.

Other news:
No major conflicting macro event found.

Entry:
100.20

Stop-loss:
98.50

Target:
103.60

Risk:
0.5% paper account

Expires:
15 minutes

Buttons:
Approve | Reject | Snooze | Ask Jax
```

## Approval Rules

- Approval must reference candidate id.
- Expired candidates cannot be approved.
- Rejected candidates store reason.
- Approval creates execution instruction.
- Execution instruction remains paper-only.
- Mobile approval action must be audited.

## Backend Requirements

Add or verify:

```text
notification_outbox
mobile_approval_tokens
candidate expiry
approval API
telegram webhook
audit records
```

## Security

- Use one-time approval token.
- Expire quickly.
- Require candidate still valid.
- Reject if guardrails changed since notification.
- Never allow arbitrary symbol/order from mobile message.

## Acceptance Criteria

- Mobile alert is received.
- Approve creates paper execution instruction.
- Reject stores reason.
- Expired approval is rejected.
- Tampered approval is rejected.
- No live order possible.
