# 10 — Frontend, Mobile, and Approval

## Goal

Make swing research understandable and safe to review.

The UI should make it clear whether the item is:

```text
research only
swing thesis
swing paper candidate
intraday paper candidate
approved paper trade
revalidation warning
closed/reflected trade
```

## New/Updated Pages

```text
/research-inbox
/swing-theses
/swing-theses/:id
/candidates/:candidateId/evidence
/candidates/:candidateId/revalidation
/trading-modes
/uat-readiness
```

## Research Inbox

Columns:

```text
time
source
event type
headline
affected ETFs
severity
source quality
state
next action
```

Actions:

```text
Research now
Ignore
Archive
View raw sources
```

## Swing Thesis Page

Must show:

```text
plain English thesis
ETF
expected hold period
historical +1d/+3d/+5d/+10d results
priced-in verdict
confounders
calendar risk
risk summary
invalidators
candidate eligibility
```

## Candidate Approval Page

For swing candidates, show:

```text
This is a multi-day paper candidate.
Overnight risk is allowed.
Weekend hold is not allowed unless explicitly changed.
Jax will revalidate daily.
Max hold is X days.
```

Buttons:

```text
Approve paper candidate
Reject
Request more research
Mark as watch only
```

## Mobile Notification

Only notify on:

```text
high-severity research trigger accepted
swing thesis becomes candidate-eligible
candidate awaiting approval
revalidation warning/failure
paper trade needs close/review
```

Do not notify on every World Monitor event.

## Beginner Summary Requirements

Every candidate must include:

```text
What happened?
Why this ETF?
Why swing or intraday?
What history says?
Is it already priced in?
What else could be moving it?
What is the risk?
When does Jax walk away?
What happens after approval?
```

## Tests

- UI clearly labels swing vs intraday.
- Approval page blocks if evidence bundle missing.
- Approval page blocks if guardrails failed.
- Mobile token created only for candidate awaiting approval.
- Revalidation warning appears on open swing paper trade.
- User rejection captures reason.
