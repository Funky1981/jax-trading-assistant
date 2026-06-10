# 02 — Research Trigger Contract

## Important Naming Rule

Do not call World Monitor output a trade signal inside Jax.

Use this naming:

```text
World Monitor output = research_trigger
Jax output after evidence checks = candidate_trade
Broker output after approval = execution_instruction
```

## Payload Contract

```json
{
  "source": "world-monitor",
  "source_event_id": "wm-20260529-001",
  "event_type": "macro_rates",
  "headline": "Fed signals possible rate cuts after weaker inflation print",
  "summary": "Multiple sources report softer inflation data and falling yields.",
  "source_urls": [
    "https://example.com/source-1",
    "https://example.com/source-2"
  ],
  "source_count": 2,
  "timestamp_utc": "2026-05-29T10:15:00Z",
  "region": "US",
  "possible_affected_etfs": ["QQQ", "SPY", "TLT"],
  "asset_themes": ["rates", "growth_equities", "bonds"],
  "confidence": 0.72,
  "reason": "Rates-sensitive headline likely relevant to QQQ, SPY and TLT.",
  "raw_payload": {}
}
```

## Required Fields

- `source`
- `source_event_id`
- `event_type`
- `headline`
- `source_urls`
- `source_count`
- `timestamp_utc`
- `possible_affected_etfs`
- `reason`

## Allowed Event Types For Phase 1

```text
macro_rates
inflation
central_bank
geopolitical
energy_oil
semiconductor_ai
financial_credit
commodity_shock
cyber_outage
market_panic
supply_chain
unknown
```

## Rejection Rules

Jax must reject or quarantine the trigger if:

- `timestamp_utc` is missing.
- `source_urls` is empty.
- `source_count` is below the configured threshold.
- The event is older than the configured freshness window.
- The payload contains direct trade language such as `buy`, `sell`, `short`, `go long`, or `execute` as an instruction.
- No ETF mapping can be justified.
- The event type is unknown and confidence is low.

## Adapter Responsibility

The adapter may classify and enrich, but it cannot approve or execute.

The adapter output is only a hypothesis:

```text
"This may be worth Jax researching."
```
