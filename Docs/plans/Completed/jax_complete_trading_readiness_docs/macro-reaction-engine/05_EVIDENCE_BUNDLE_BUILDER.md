# 05 — Evidence Bundle Builder

## Goal

Every candidate must have an evidence bundle.

No evidence bundle means no approval.

## Evidence bundle contents

Required sections:

```text
1. What happened
2. Structured macro data
3. Why this ETF
4. Expected market reaction
5. Actual chart reaction
6. Priced-in verdict
7. Confounders
8. Historical comparison
9. Risk guardrail result
10. Entry/stop/target proposal
11. Walk-away reasons
12. Beginner summary
```

## Data model

### macro_evidence_bundles

```sql
CREATE TABLE macro_evidence_bundles (
    id UUID PRIMARY KEY,
    macro_event_id UUID NOT NULL REFERENCES macro_events(id),
    symbol TEXT NOT NULL,
    status TEXT NOT NULL,
    verdict TEXT NOT NULL,
    summary TEXT NOT NULL,
    evidence JSONB NOT NULL,
    missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    walkaway_reasons TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(macro_event_id, symbol)
);
```

## Bundle verdicts

```text
candidate_allowed
candidate_blocked
watch_only
insufficient_evidence
```

## Required evidence gates

Candidate allowed only when:

```text
macro event valid
scenario mapped
ETF allowlisted
chart reaction confirms
not too extended
priced-in verdict is not_priced_in or partially_priced_in
no high-severity confounders
risk calculation succeeds
paper mode enabled
human approval required
```

## Codex task

```text
Build evidence bundle generation for macro event ETF candidates.

The builder must collect:
- macro event data
- scenario result
- reaction snapshots
- priced-in scores
- confounders
- risk check
- historical comparison placeholder if full historical engine not complete

It must output candidate_allowed, candidate_blocked, watch_only, or insufficient_evidence.
```

## Tests

```text
full evidence creates candidate_allowed bundle
missing chart reaction creates insufficient_evidence
priced_in creates candidate_blocked
high-severity confounder creates candidate_blocked
too_extended creates watch_only
missing historical comparison is allowed only as explicit limitation in phase 1
```

## Acceptance criteria

```text
evidence bundle persisted
bundle is readable by UI/API
missing evidence is explicit
candidate cannot be created without candidate_allowed bundle
walk-away reasons visible
```
