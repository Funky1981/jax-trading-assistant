# 04 — Priced-In and Confounder Checks

## Goal

Avoid trading news that the market already expected or where another event is driving the chart.

This is the difference between:

```text
Headline chasing
```

and:

```text
Evidence-based event trading
```

## Priced-in verdicts

```text
not_priced_in
partially_priced_in
priced_in
overreaction
unclear
```

Hard rule:

```text
priced_in = no candidate trade
unclear = no candidate trade
```

## Confounders

A confounder is another market-moving factor that may explain the move.

Examples:

```text
Fed speaker at same time
CPI and jobs released close together
major geopolitical shock
earnings mega-cap move
Treasury auction
oil shock
market-wide liquidity crash
unexpected central-bank statement
```

## Data model

### macro_priced_in_scores

```sql
CREATE TABLE macro_priced_in_scores (
    id UUID PRIMARY KEY,
    macro_event_id UUID NOT NULL REFERENCES macro_events(id),
    symbol TEXT NOT NULL,
    verdict TEXT NOT NULL,
    score NUMERIC NOT NULL,
    reasons TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(macro_event_id, symbol)
);
```

### macro_confounders

```sql
CREATE TABLE macro_confounders (
    id UUID PRIMARY KEY,
    macro_event_id UUID NOT NULL REFERENCES macro_events(id),
    confounder_type TEXT NOT NULL,
    headline TEXT NOT NULL,
    source TEXT NULL,
    severity TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Priced-in scoring signals

Use:

```text
pre-event ETF drift
consensus vs actual surprise size
pre-event yield move
news saturation before release
volatility already elevated
analyst expectation clustering
gap before release
```

Phase 1 can use simple deterministic scoring.

Example:

```text
actual surprise > 50% and pre-event drift small = not_priced_in
actual near expected and pre-event move large = priced_in
conflicting ETF/yield reaction = unclear
huge immediate move = possible overreaction
```

## Codex task

```text
Add priced-in scoring and confounder checks.

Inputs:
- macro_event
- scenario result
- reaction snapshots
- source metadata
- optional event calendar overlap

Outputs:
- priced-in verdict per ETF
- confounder list
- trade eligibility decision
```

## Tests

```text
large surprise + small pre-move = not_priced_in
small surprise + big pre-move = priced_in
missing data = unclear
same-time Fed speech creates confounder
major unrelated market news blocks candidate
overextended move can become overreaction/watchlist not immediate candidate
```

## Acceptance criteria

```text
priced-in score stored per ETF
confounders stored
priced_in blocks candidate
unclear blocks candidate
high-severity confounder blocks candidate
all reasons are human-readable
```
