# 05 — Historical Backfill Pipeline

## Goal

Fill the Jax database with ETF candles, news, macro events, and derived analytics.

## Backfill Scope

Initial target:

- 11 ETF allowlist symbols
- 1 to 2 years of data
- daily and intraday candles where available
- news and macro events
- ETF event-window analytics

ETF symbols:

```text
SPY
QQQ
DIA
IWM
XLK
XLF
XLE
SMH
SOXX
TLT
GLD
```

## Backfill Job Types

### 1. Candle Backfill

Input:

```text
symbols
timeframe
from
to
provider
```

Output:

- rows in `candles`
- data quality summary

### 2. News Backfill

Input:

```text
symbols
themes
from
to
provider
```

Output:

- `event_raw`
- `event_normalized`
- `event_symbol_map`

### 3. Macro Calendar Backfill

Input:

```text
country/region
impact
from
to
```

Output:

- macro event records in event tables

### 4. Event Study Backfill

Input:

```text
event ids
ETF symbols
windows
```

Windows:

```text
-1d to event
-4h to event
-1h to event
event to +5m
event to +15m
event to +1h
event to +4h
event to +1d
```

Output:

- `event_windows`
- `event_priced_in_scores`
- `event_confounders`

## Suggested Command/API

Add either CLI or research endpoint:

```text
POST /research/backfill/events
POST /research/backfill/candles
POST /research/backfill/event-study
GET  /research/backfill/runs/{id}
```

or equivalent internal runner tasks.

## Idempotency

Backfills must be safe to re-run.

Rules:

- use canonical event keys
- use `ON CONFLICT DO UPDATE` where appropriate
- log provider failures
- store run summaries
- avoid duplicate event windows

## Acceptance Criteria

- Can backfill candles for all phase-one ETFs.
- Can backfill provider news.
- Can generate event windows for historical events.
- Can rerun the same backfill without duplicate rows.
- Backfill failures are visible in run output.
