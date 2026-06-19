# Decision Core Specification

## Purpose

The Decision Core is the central product layer of Jax.

It receives a structured market event and returns a structured decision.

It does not execute trades.

It does not call the broker.

It does not rely on free-form LLM text as the source of truth.

## Core principle

```text
NO_TRADE unless evidence upgrades the decision.
```

## Decision enum

```text
NO_TRADE
WATCH
SETUP_FORMING
TRADE_CANDIDATE
PAPER_APPROVAL_REQUIRED
REJECTED_BY_RISK
```

## Event schema

```json
{
  "event_id": "evt_2026_06_18_ftse_oil_labour",
  "source_type": "news_url",
  "source_url": "https://example.com/article",
  "received_at": "2026-06-18T10:30:00Z",
  "headline": "FTSE falls as oil slump outweighs strong UK labour data",
  "summary": "FTSE weakness appears driven by oil weakness while labour data is stronger than expected.",
  "event_type": "MACRO_COMMODITY_INDEX_MOVE",
  "primary_drivers": ["oil_price_drop"],
  "conflicting_drivers": ["strong_uk_labour_data", "boe_policy_uncertainty"],
  "affected_assets": ["FTSE100", "BP", "SHEL", "GBP", "UK_GILTS"],
  "asset_classes": ["equity_index", "single_stock", "fx", "rates"],
  "geography": ["UK"],
  "time_sensitivity": "medium",
  "uncertainty_notes": [
    "BoE decision pending",
    "Index move may be composition-driven"
  ]
}
```

## Decision schema

```json
{
  "decision_id": "dec_2026_06_18_001",
  "event_id": "evt_2026_06_18_ftse_oil_labour",
  "brain": "DECISION_CORE",
  "decision": "NO_TRADE",
  "confidence_score": 0.72,
  "clarity_score": 0.38,
  "edge_score": 0.22,
  "conflict_score": 0.79,
  "risk_score": 0.66,
  "primary_reason": "Conflicting macro drivers and no clean trade edge.",
  "supporting_reasons": [
    "Oil weakness is dragging energy-heavy FTSE constituents.",
    "UK labour data points in a conflicting policy direction.",
    "BoE decision uncertainty remains unresolved."
  ],
  "required_confirmations": [
    "Oil stabilises",
    "Energy stocks stop underperforming",
    "BoE decision removes policy uncertainty"
  ],
  "invalidation_conditions": [
    "Oil continues falling",
    "FTSE breaks lower on strong volume",
    "GBP/rates reaction worsens"
  ],
  "allowed_actions": [
    "store_event",
    "monitor",
    "review_later"
  ],
  "forbidden_actions": [
    "execute_trade",
    "create_live_order",
    "auto_approve"
  ],
  "review_after": [
    "1_day",
    "1_week",
    "1_month"
  ]
}
```

## Evidence bundle schema

```json
{
  "evidence_id": "evb_001",
  "input_event": {},
  "market_context": {},
  "reasoning_trace_summary": "Short summary only. No hidden chain-of-thought.",
  "scores": {
    "clarity_score": 0.38,
    "edge_score": 0.22,
    "conflict_score": 0.79,
    "risk_score": 0.66
  },
  "final_decision": {},
  "generated_at": "2026-06-18T10:35:00Z",
  "version": "decision-core-v1"
}
```

## Deterministic rules v1

| Condition | Decision |
|---|---|
| conflict_score >= 0.70 and edge_score < 0.60 | NO_TRADE |
| clarity_score < 0.50 | NO_TRADE |
| risk_score > 0.70 | NO_TRADE |
| edge_score between 0.50 and 0.70 with missing confirmation | WATCH |
| edge_score >= 0.75 and risk_score <= 0.60 | TRADE_CANDIDATE |
| risk veto fails | REJECTED_BY_RISK |

## Candidate requirements

A `TRADE_CANDIDATE` must include:

- asset
- setup family
- catalyst
- edge reason
- risk reason
- proposed invalidation
- required confirmation
- review horizon
- no live execution action

If any are missing, downgrade to:

```text
WATCH
```

or:

```text
NO_TRADE
```

## Forbidden shortcuts

The Decision Core must not:

- return free-form decisions only
- call broker execution
- auto-create live orders
- treat NO_TRADE as failure
- promote an LLM suggestion without structured evidence
- ignore conflicting drivers
