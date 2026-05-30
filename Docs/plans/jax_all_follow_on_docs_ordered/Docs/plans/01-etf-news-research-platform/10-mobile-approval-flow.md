# 10 — Mobile Approval Flow

## Start With

```text
Telegram
```

Alternatives later:

```text
Pushover
Discord private channel
email fallback
React Native / Expo
Firebase Cloud Messaging
```

## Notification Format

```text
ETF: SMH
Strategy: Sector News Momentum
Action: Paper Buy
Confidence: 72%

Why:
AI chip news is moving semiconductors. SMH and SOXX both confirmed.

Priced-in:
Partially priced in, still acceptable.

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

Approve | Reject | Snooze
```

## Rules

- One-time approval token.
- Expiry required.
- Candidate revalidated on approval.
- Guardrails rechecked.
- Approval creates paper execution instruction only.
- No live order possible.
