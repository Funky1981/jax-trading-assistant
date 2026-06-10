# 03 — Economic Calendar Integration

## Goal

Jax needs structured economic calendar data.

World Monitor detects that something matters. The calendar tells Jax the actual facts.

## Required fields

```text
event_id
event_type
country/region
release_time_utc
actual
forecast
previous
revised_previous
unit
importance
source
source_url
```

## Supported event types

Phase 1:

```text
US_CPI_HEADLINE_MOM
US_CPI_HEADLINE_YOY
US_CPI_CORE_MOM
US_CPI_CORE_YOY
US_NONFARM_PAYROLLS
US_UNEMPLOYMENT_RATE
US_AVERAGE_HOURLY_EARNINGS
US_PPI
US_RETAIL_SALES
US_GDP
US_PMI
FOMC_RATE_DECISION
FOMC_STATEMENT
FOMC_DOT_PLOT
FED_CHAIR_PRESS_CONFERENCE
US_TREASURY_AUCTION
```

## Data model

### economic_calendar_events

```sql
CREATE TABLE economic_calendar_events (
    id UUID PRIMARY KEY,
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    country TEXT NOT NULL,
    release_time_utc TIMESTAMPTZ NOT NULL,
    actual NUMERIC NULL,
    forecast NUMERIC NULL,
    previous NUMERIC NULL,
    revised_previous NUMERIC NULL,
    unit TEXT NULL,
    importance TEXT NOT NULL,
    source_url TEXT NULL,
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_event_id)
);
```

## Calendar matching

When a research trigger arrives, Jax should match it to calendar data by:

```text
event_type
country/region
release_time window
headline similarity
provider id if supplied
```

## Surprise calculation

```text
surprise_value = actual - forecast
surprise_percent = (actual - forecast) / abs(forecast)
```

For some values, interpretation is not simple.

Example:

```text
higher payrolls = growth strong / hawkish rates
higher unemployment = growth weak / dovish rates
higher CPI = inflation hot / hawkish rates
```

So each event type needs a direction mapping.

## Calendar freshness

Reject or quarantine:

```text
missing release time
future event pretending to have actual
actual missing after release
forecast missing for surprise-driven events
stale provider payload
duplicate provider_event_id
```

## Codex task

```text
Create Economic Calendar Integration.

Start with fixture/provider abstraction.
Do not require paid APIs in tests.
Allow manual JSON import.
Persist calendar events and match to macro research triggers.
```

## Tests

```text
valid CPI event persists
valid NFP event persists
duplicate dedupes
actual missing after release quarantines
trigger matches calendar event by time/type
surprise calculation works
direction mapping works
```

## Acceptance criteria

```text
calendar data persisted
macro events can link to calendar events
surprise values calculated
missing/invalid data fails safely
