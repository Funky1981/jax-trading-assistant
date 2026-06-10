# 08 — Research Evidence Bundles

## Goal

Create one structured object that contains everything Jax and the user need before approving a paper trade.

## Evidence Bundle Shape

```json
{
  "event_id": "...",
  "symbol": "SMH",
  "strategy_id": "ETF_NEWS_002_SECTOR_MOMENTUM",
  "event_type": "semiconductor_ai",
  "headline": "...",
  "source": "...",
  "event_time": "...",
  "why_this_etf": "...",
  "price_reaction": {
    "pre_1h": 0.2,
    "pre_4h": 0.8,
    "post_15m": 1.1,
    "post_1h": 1.7,
    "benchmark": "QQQ",
    "abnormal_return": 0.9
  },
  "priced_in": {
    "verdict": "partially_priced_in",
    "score": 0.42,
    "reason": "Some pre-event drift existed but ETF confirmed after event."
  },
  "confounders": [
    {
      "event_id": "...",
      "type": "macro",
      "relevance_score": 0.3,
      "reason": "Fed speech overlapped but did not dominate semiconductor reaction."
    }
  ],
  "risk": {
    "entry": 100.0,
    "stop_loss": 98.5,
    "take_profit": 103.0,
    "risk_reward": 2.0,
    "flatten_by_close": true
  },
  "guardrails": {
    "allowlist_pass": true,
    "spread_pass": true,
    "stale_quote_pass": true,
    "paper_mode_pass": true,
    "approval_required": true
  }
}
```

## Storage

Store generated bundles in:

- `research_summaries.evidence`
- candidate trade evidence bundle field if available
- memory bank `signals` summary if candidate is created
- memory bank `trades` after approval/execution

## Beginner Summary

Every evidence bundle should produce:

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

- Evidence bundle can be generated for a historical event.
- Evidence bundle can be generated for a live event.
- Candidate cannot be approved without evidence bundle.
- AI output references evidence fields.
- UI can render simple and detailed views.
