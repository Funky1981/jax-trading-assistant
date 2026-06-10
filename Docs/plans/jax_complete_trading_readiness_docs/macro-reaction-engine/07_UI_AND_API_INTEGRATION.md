# 07 — UI and API Integration

## Goal

Expose macro event analysis clearly in the Jax frontend/API.

The UI should help the user answer:

```text
What happened?
What did Jax check?
What does the chart show?
Why is this a candidate or no-trade?
What would invalidate the setup?
What do I approve/reject?
```

## API endpoints

Suggested endpoints:

```http
GET  /api/v1/macro/events
GET  /api/v1/macro/events/{id}
GET  /api/v1/macro/events/{id}/reactions
GET  /api/v1/macro/events/{id}/evidence
POST /api/v1/macro/events/{id}/research
POST /api/v1/macro/candidates/{id}/approve
POST /api/v1/macro/candidates/{id}/reject
```

## UI screens

### Macro Event Inbox

Shows:

```text
event time
event type
headline
actual vs expected
direction
mapped ETFs
status
confidence
```

### Event Detail

Shows:

```text
macro facts
ETF mappings
reaction snapshots
chart window
priced-in verdict
confounders
evidence bundle
candidate/no-trade result
```

### Candidate Review

Shows:

```text
ETF
side/bias
entry type
entry reference
stop
target
risk
why this trade
why not to take it
approval buttons
```

## Status values

```text
received
validated
researching
reaction_checked
evidence_ready
candidate_created
candidate_rejected
watch_only
rejected
archived
```

## Codex task

```text
Add API and frontend views for macro reaction analysis.

Use existing frontend style and existing protected API patterns.

Do not add execution buttons unless they already route through the existing manual approval/broker guard path.
```

## Tests

```text
macro event list loads
event detail loads
reaction snapshot renders
evidence bundle renders
candidate review shows human approval required
rejected candidate shows walk-away reasons
```

## Acceptance criteria

```text
user can inspect macro event
user can see chart reaction summary
user can see evidence bundle
user can approve/reject paper candidate only
UI never implies World Monitor created a trade
UI never hides missing evidence
```
