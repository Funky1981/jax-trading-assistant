# 15 — Event Clustering and Dedupe

## Purpose

Avoid paying AI to analyse duplicate news.

## Problem

One real event may appear as many headlines.

Example:

```text
Reuters CPI headline
CNBC CPI article
Yahoo Finance recap
MarketWatch analysis
analyst commentary
```

Jax should treat these as one canonical event.

## Canonical Event Key

Suggested key:

```text
event_type
normalized_topic
event_timestamp_bucket
primary_region
affected_theme
```

Example:

```text
inflation|us_cpi_hotter_than_expected|2026-05-13T13:30Z|US|macro_rates
```

## Dedupe Checks

Compare:

```text
headline similarity
source timestamp
provider event id
mentioned entities
event type
affected ETFs
macro release id if known
```

## Output

```json
{
  "canonical_event_id": "...",
  "is_duplicate": false,
  "cluster_size": 4,
  "sources": ["Reuters", "CNBC", "Yahoo", "MarketWatch"],
  "summary": "US CPI came in hotter than expected.",
  "affected_etfs": ["QQQ", "SPY", "TLT"]
}
```

## AI Call Rule

Do not call AI for each duplicate.

Call AI only after clustering, and only if deterministic gates pass.

## Acceptance Criteria

- Duplicate headlines map to one canonical event.
- Event clusters store all sources.
- AI is called once per canonical event at most.
- Dedupe decisions are auditable.
