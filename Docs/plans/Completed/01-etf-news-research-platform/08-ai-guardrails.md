# 08 — AI Guardrails

## AI Can

```text
summarise evidence
explain event relevance
explain ETF selection
compare similar historical events
describe risks
recommend trade/wait/reject
produce beginner approval summary
```

## AI Cannot

```text
submit broker orders
approve trades
bypass guardrails
trade non-allowlisted symbols
override priced-in rejection
override stale quote/spread/session checks
invent missing data
change stop-loss after approval
increase position size
enable live trading
```

## Output Contract

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

## Acceptance Criteria

- AI cannot create execution instruction.
- AI output schema validated.
- Failed guardrails override AI.
- Decision is auditable.
