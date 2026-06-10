# 12 — Beginner UX

## Goal

Make Jax understandable for users who have never traded before.

## Core UX Rule

If Jax cannot explain a candidate simply, it should not ask for approval.

## Required Screens

### 1. ETF Universe Screen

Show:

- approved ETFs
- what each ETF represents
- risk level
- examples of news that affects it

Example:

```text
SMH — Semiconductor ETF
Used when: AI/chip sector news
Avoids: guessing one winning chip stock
Risk: medium/high sector movement
```

### 2. Strategy Cards

Each strategy card shows:

```text
what it watches
which ETFs it can trade
when it trades
when it walks away
typical holding time
risk level
example trade
```

### 3. Candidate Trade Screen

Show:

```text
plain-English summary
ETF selected
why this ETF
news sources
price chart before/after news
priced-in verdict
other news/confounders
risk controls
entry/stop/target
approve/reject buttons
```

### 4. Research Timeline

Show:

```text
news timestamp
ETF movement
related/conflicting news
Jax analysis
approval decision
paper order
exit
post-trade reflection
```

### 5. Beginner Toggle

Add modes:

```text
Simple
Detailed
Technical
```

## Beginner Explanations

Avoid raw jargon where possible.

Instead of:

```text
abnormal return versus benchmark
```

Say:

```text
This ETF moved more than the wider market, so the news may have had a real effect.
```

## Acceptance Criteria

- Strategy cards are readable by a beginner.
- Every candidate explains why, risk, and stop-loss.
- User can see why Jax walked away.
- User can disable strategies easily.
