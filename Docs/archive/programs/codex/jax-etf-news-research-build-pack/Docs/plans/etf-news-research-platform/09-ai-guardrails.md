# 09 — AI Guardrails

## Goal

Use AI for research summaries and ranking, not direct trading.

## AI Allowed

AI can:

- summarise evidence
- explain event relevance
- explain ETF selection
- compare similar historical events
- describe risks
- recommend trade/wait/reject
- produce beginner-friendly approval summary

## AI Forbidden

AI must not:

- submit broker orders
- approve trades
- bypass guardrails
- trade non-allowlisted symbols
- override priced-in rejection
- override stale quote/spread/session checks
- invent missing data
- change stop-loss after approval
- increase position size
- enable live trading

## AI Input Contract

AI receives only structured evidence:

```text
event summary
source list
ETF mapping
historical event windows
confounders
priced-in score
risk checks
strategy rules
position state
```

## AI Output Contract

```json
{
  "decision": "trade | wait | reject",
  "confidence": 0.0,
  "plain_english_summary": "...",
  "why_this_etf": "...",
  "priced_in_view": "...",
  "main_risks": ["..."],
  "walk_away_reason": null,
  "approval_message": "..."
}
```

## Validation

Reject AI output if:

- confidence missing
- decision invalid
- references unavailable data
- recommends blocked instrument
- recommends trade despite failed guardrail
- lacks plain-English explanation

## Acceptance Criteria

- AI cannot create execution instructions.
- AI recommendation is stored.
- AI decision is auditable.
- Failed guardrails always override AI.
