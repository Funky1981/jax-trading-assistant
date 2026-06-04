# 02 — Database Event-Study Schema

## Goal

Extend existing Postgres schema for historical ETF/news research.

## Reuse Existing Tables

```text
event_sources
event_raw
event_normalized
event_symbol_map
quotes
candles
candidate_trades
candidate_events
memory_items
```

## Add Tables

```text
event_windows
event_confounders
event_priced_in_scores
etf_context_snapshots
research_summaries
```

## event_windows

Stores ETF price movement around news.

Fields:

```text
id
event_id
symbol
window_name
window_start
window_end
price_before
price_after
return_pct
benchmark_symbol
benchmark_return_pct
abnormal_return_pct
volume_before
volume_after
volume_change_pct
spread_before_bps
spread_after_bps
volatility_adjusted_move
data_quality
created_at
```

## event_confounders

Stores other events that may have affected the same move.

Fields:

```text
id
event_id
confounding_event_id
symbol
relationship_type
time_distance_minutes
relevance_score
reason
created_at
```

## event_priced_in_scores

Stores priced-in verdict.

Fields:

```text
id
event_id
symbol
pre_event_1h_return
pre_event_4h_return
pre_event_1d_return
post_event_15m_return
post_event_1h_return
benchmark_symbol
benchmark_return
abnormal_return
volume_confirmation_score
spread_quality_score
priced_in_score
verdict
reason
created_at
```

## Acceptance Criteria

- Migrations apply from empty DB.
- Migrations apply to existing DB.
- Unique constraints prevent duplicate event windows and priced-in scores.
- Integration tests verify schema.
