# Post Decision Review Standard

## Purpose

Jax must learn from every decision, including decisions not to trade.

Most events should be rejected. Those rejections must still be reviewed.

## Review windows

Every decision must schedule reviews after:

```text
1 day
1 week
1 month
```

## Review schema

```json
{
  "review_id": "rev_001",
  "decision_id": "dec_001",
  "event_id": "evt_001",
  "original_decision": "NO_TRADE",
  "review_window": "1_week",
  "asset_outcome": {},
  "market_outcome": {},
  "was_decision_correct": true,
  "missed_opportunity": false,
  "avoided_loss": true,
  "lesson": "Conflicting macro drivers produced no clean swing setup.",
  "memory_tags": [
    "no_trade",
    "macro_conflict",
    "oil",
    "ftse"
  ],
  "scoring_adjustment_suggestion": null
}
```

## NO_TRADE review questions

For every no-trade:

1. Did the event later produce a clean move?
2. Was the rejection reason valid?
3. Was there a missed opportunity?
4. Did the conflicting signal resolve?
5. Did required confirmations appear later?
6. Should the scoring thresholds change?
7. Should a new golden case be added?
8. Did Jax avoid a poor trade?

## WATCH review questions

For every watch:

1. Did confirmation appear?
2. Did invalidation occur?
3. Did Jax fail to upgrade in time?
4. Was watch status too cautious or correct?
5. Should the setup family be improved?

## PAPER TRADE review questions

For every paper trade:

1. Did the setup behave as expected?
2. Did invalidation trigger?
3. Was entry realistic?
4. Was stop realistic?
5. Was target realistic?
6. Was risk/reward realistic?
7. Did slippage/spread matter?
8. Should the strategy be promoted, held, or rejected?

## Memory write requirements

Each review must write:

- original event
- original decision
- review outcome
- lessons
- tags
- score adjustment suggestion
- whether golden tests need updating

## Promotion impact

| Review Result | Effect |
|---|---|
| Correct NO_TRADE | strengthens rejection rule |
| Missed opportunity | creates investigation item |
| Bad paper trade | weakens setup family |
| Good paper trade | adds evidence, no live promotion |
| Repeated good paper evidence | may move to PAPER_PROVEN |

## Forbidden

Jax must not:

- only review winning trades
- ignore no-trades
- promote strategies from one good outcome
- rewrite history after review
- delete failed paper trades
