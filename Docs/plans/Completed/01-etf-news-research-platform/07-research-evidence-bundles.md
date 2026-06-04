# 07 — Research Evidence Bundles

## Goal

Create one compact object containing everything needed before approval.

## Bundle Contains

```text
event_id
symbol
strategy_id
event_type
headline
source
event_time
why_this_etf
price_reaction
priced_in verdict
confounders
risk
guardrails
beginner summary
```

## Beginner Questions

Each bundle must answer:

```text
What happened?
Why this ETF?
Was the move already priced in?
What else may be affecting it?
What is the risk?
Where is the stop-loss?
When does the idea expire?
What would make Jax walk away?
```

## Acceptance Criteria

- Bundle generated for historical event.
- Bundle generated for live event.
- Candidate cannot be approved without bundle.
- AI receives bundle, not raw chaos.
- UI renders simple/detailed view.
