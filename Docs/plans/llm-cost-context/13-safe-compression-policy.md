# 12 — Safe Compression Policy

## Purpose

Define when Headroom or any compression layer may be used.

## Main Concern

Compression can save tokens, but it can also remove or distort details.

For Jax, losing the wrong detail is unacceptable.

## Core Rule

```text
Compression is allowed only on non-authoritative supporting context.
```

## Compression Zones

### Zone A — Never Compress

```text
symbol
asset_type
event_id
candidate_id
strategy_id
timestamp
source timestamp
quote timestamp
entry
stop_loss
take_profit
risk_amount
position_size
spread_bps
quote_freshness_seconds
priced_in_verdict
priced_in_score
guardrail_results
approval_token
approval_expiry
paper_mode
live_mode
broker_order_id
fill_status
```

### Zone B — Deterministic Compaction Only

These may be reduced by Jax code, but not by lossy AI compression:

```text
candles
quotes
price windows
event windows
confounder scores
provider health
backfill run stats
strategy eligibility checks
```

Example:

```text
500 candles -> return windows, volume change, volatility-adjusted move
```

### Zone C — Safe To Compress

```text
article bodies
duplicated news text
long logs
test failures
search results
documentation excerpts
historical research prose
post-trade narrative notes
developer discussion
```

## Required Envelope

Every compressed payload must be wrapped with metadata:

```json
{
  "compression_allowed": true,
  "compression_zone": "C",
  "source_ids": ["..."],
  "original_available": true,
  "retrieval_key": "...",
  "content_hash": "...",
  "compressed_text": "..."
}
```

## Retrieval Requirement

If compressed context is used, the original must remain retrievable.

Do not delete originals.

## Approval Flow Rule

Approval summaries may use compressed supporting context, but the final approval packet must include uncompressed trading truth.

## Forbidden

Do not compress:

```text
approval callback payloads
broker order state
risk controls
stop-loss
take-profit
position size
priced-in verdict
guardrail status
```

## Failure Behaviour

```text
historical/offline task -> continue without compression if budget allows
live candidate task -> fail closed or use deterministic summary only
approval task -> fail closed if required evidence is missing
```

## Acceptance Criteria

- Compression zones are enforced in code.
- Tests prove Zone A fields never pass through compression.
- Tests prove compressed content retains source IDs.
- Tests prove original content can be retrieved.
- Tests prove approval packet contains uncompressed trading truth.
