# 14 — AI Eligibility Gate

## Purpose

Prevent unnecessary AI calls.

## Rule

Before any AI/model call, Jax must check whether AI is actually needed.

## Eligibility Input

```json
{
  "task_type": "approval_summary",
  "event_id": "...",
  "symbol": "QQQ",
  "strategy_id": "ETF_NEWS_002_SECTOR_MOMENTUM",
  "candidate_id": "...",
  "evidence_bundle_id": "..."
}
```

## Required Checks

### Event Checks

```text
event exists
event is not duplicate
event source quality acceptable
event is recent enough
event category is tradeable
```

### ETF Checks

```text
symbol allowlisted
asset type is ETF
not leveraged
not inverse
not volatility-linked
paper mode only
```

### Market Checks

```text
quote fresh
spread acceptable
market/session acceptable
no halt state
```

### Research Checks

```text
ETF mapping exists
priced-in verdict exists
priced-in verdict not hard reject
confounder analysis exists
evidence bundle exists
```

### Cost Checks

```text
task budget available
daily budget available
model route enabled
paid escalation allowed if needed
```

## Output

```json
{
  "eligible": true,
  "allowed_model_route": "local-small",
  "reason": "Evidence bundle complete and all guardrails passed.",
  "blocked_reason": null
}
```

or:

```json
{
  "eligible": false,
  "allowed_model_route": "disabled",
  "reason": null,
  "blocked_reason": "priced_in verdict blocks trade"
}
```

## Hard Blocks

No model call if:

```text
symbol not allowlisted
event duplicate
quote stale
spread too wide
priced_in
unclear
missing evidence bundle
live trading path
budget unavailable
```

## Acceptance Criteria

- Eligibility gate runs before every AI call.
- Blocked model calls are logged.
- Most rejected events produce no AI call.
- Tests cover all hard blocks.
