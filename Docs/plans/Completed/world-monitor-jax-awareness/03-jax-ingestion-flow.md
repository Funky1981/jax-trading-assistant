# 03 — Jax Ingestion Flow

## Goal

Jax should treat World Monitor events as external research inputs.

## Target Flow

```text
World Monitor event/headline
  ↓
world-monitor-jax-adapter
  ↓
Jax research trigger endpoint
  ↓
event_raw
  ↓
event_normalized
  ↓
event_symbol_map
  ↓
research pipeline
```

## Suggested Endpoint

Add or reuse a research endpoint similar to:

```http
POST /research/events/world-monitor
```

Alternative generic endpoint:

```http
POST /research/events/ingest
```

## Storage Mapping

| Payload Field | Jax Storage Target |
|---|---|
| `source` | `event_sources` / raw metadata |
| `source_event_id` | `event_raw.source_event_id` |
| `headline` | `event_raw.title` / `event_normalized.title` |
| `summary` | `event_raw.summary` / `event_normalized.summary` |
| `timestamp_utc` | `event_normalized.event_time` |
| `event_type` | `event_normalized.attributes.event_type` |
| `possible_affected_etfs` | `event_symbol_map` |
| `source_urls` | raw payload / evidence metadata |
| `confidence` | attributes / research metadata |

## Processing Steps

1. Validate request schema.
2. Reject trade-execution language.
3. Deduplicate using `source + source_event_id + headline hash`.
4. Persist raw event.
5. Normalise event type and timestamp.
6. Map candidate ETFs.
7. Store audit metadata.
8. Queue or trigger Jax research review.

## Jax Research Review

For each accepted trigger, Jax must run:

1. Source verification.
2. ETF relevance mapping.
3. Historical event comparison.
4. Priced-in scoring.
5. Confounder lookup.
6. Risk guardrail check.
7. Candidate trade creation only if all required checks pass.

## Output States

```text
received
validated
rejected
researching
candidate_created
candidate_rejected
archived
```

## Important Boundary

No payload from World Monitor should directly create:

- order
- execution instruction
- broker request
- approved trade
- live trade

Only Jax may create a candidate, and only after evidence checks.
