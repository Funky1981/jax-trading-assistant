# 09 — World Monitor Adapter Compatibility

## Goal

Keep World Monitor completely separate while making its outputs useful to Jax.

World Monitor remains:

```text
separate dashboard / awareness radar
```

Jax remains:

```text
research + evidence + candidate engine
```

Adapter remains:

```text
thin bridge with strict validation
```

## Flow

```text
World Monitor alert/headline
  -> world-monitor-jax-adapter
  -> Jax research trigger inbox
  -> Jax source verification
  -> Jax event classification
  -> Jax swing research
  -> optional intraday paper candidate
```

## Adapter Payload

```json
{
  "source": "world-monitor",
  "source_event_id": "wm-20260614-001",
  "trigger_type": "research_trigger",
  "headline": "Fed signals slower rate cuts after inflation surprise",
  "summary": "...",
  "source_urls": ["https://..."],
  "source_count": 3,
  "source_quality": "tier_1_plus_tier_2",
  "timestamp_utc": "2026-06-14T14:30:00Z",
  "region": "US",
  "themes": ["rates", "inflation"],
  "possible_affected_etfs": ["TLT", "QQQ", "SPY", "GLD"],
  "severity": "high",
  "confidence": 0.74,
  "reason": "Multiple independent sources show a rates-sensitive surprise.",
  "raw_payload": {}
}
```

## Research Trigger Rules

Adapter can send only:

```text
research_trigger
```

Adapter cannot send:

```text
candidate_trade
execution_instruction
approval
broker_order
risk_override
```

## Jax Ingestion Rules

Jax must validate:

```text
timestamp present
source urls present
source count threshold
allowed theme
ETF suggestions allowlisted
no trade instruction language
not duplicate
not stale
```

Accepted trigger states:

```text
received
validated
rejected
researching
thesis_created
candidate_created
candidate_rejected
archived
```

## Swing-Specific Adapter Benefit

World Monitor is especially useful for swing research because it provides broader context:

```text
geopolitical escalation
energy shocks
macro stress
infrastructure/cyber events
regional instability
central bank and government sources
```

This context helps Jax detect confounders and multi-day themes.

## Tests

- Adapter payload creates research trigger only.
- Missing URLs reject.
- Duplicate source event id idempotent.
- Non-allowlisted ETF suggestions are dropped, not trusted.
- Trigger cannot create candidate without research engine output.
- Trigger cannot create order under any condition.
