# 01 — Macro Event Model and Calendar Data

## Goal

Create a structured macro event model so Jax does not rely on headlines alone.

World Monitor can detect that "jobs data matters", but Jax needs structured facts:

```text
actual value
expected value
previous value
surprise size
release timestamp
event category
affected themes
affected ETFs
```

## Events in scope

Phase 1 supports:

```text
US_NONFARM_PAYROLLS
US_UNEMPLOYMENT_RATE
US_AVERAGE_HOURLY_EARNINGS
US_CPI_HEADLINE
US_CPI_CORE
US_PPI
FOMC_RATE_DECISION
FOMC_STATEMENT
FOMC_DOT_PLOT
FED_CHAIR_PRESS_CONFERENCE
```

## Out of scope for phase 1

```text
UK CPI
ECB
BoE
earnings
single-stock news
options
futures
crypto
forex
```

## Data model

### macro_events

```sql
CREATE TABLE macro_events (
    id UUID PRIMARY KEY,
    source TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    region TEXT NOT NULL,
    event_time_utc TIMESTAMPTZ NOT NULL,
    headline TEXT NOT NULL,
    summary TEXT NULL,
    actual_value NUMERIC NULL,
    expected_value NUMERIC NULL,
    previous_value NUMERIC NULL,
    unit TEXT NULL,
    surprise_value NUMERIC NULL,
    surprise_percent NUMERIC NULL,
    direction TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(source, source_event_id)
);
```

### macro_event_etf_map

```sql
CREATE TABLE macro_event_etf_map (
    id UUID PRIMARY KEY,
    macro_event_id UUID NOT NULL REFERENCES macro_events(id),
    symbol TEXT NOT NULL,
    theme TEXT NOT NULL,
    mapping_reason TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(macro_event_id, symbol)
);
```

## Direction values

```text
hawkish_rates
dovish_rates
risk_on
risk_off
inflation_hot
inflation_cool
growth_strong
growth_weak
unclear
```

## Example: hot jobs event

```json
{
  "event_type": "US_NONFARM_PAYROLLS",
  "actual_value": 172000,
  "expected_value": 85000,
  "previous_value": 139000,
  "unit": "jobs",
  "direction": "hawkish_rates",
  "affected_etfs": ["QQQ", "SPY", "TLT", "IWM"],
  "reason": "Jobs beat forecast strongly, reducing rate-cut expectations."
}
```

## Validation rules

Reject or quarantine when:

```text
event_time_utc missing
event_type unsupported
actual/expected missing for numeric releases
source_event_id duplicate
source confidence too low
event is older than configured freshness window
event has no ETF mapping
```

## Codex task

```text
Add macro event domain models, storage, migrations, validation, and tests.

Do not create trade candidates in this phase.
Do not call broker/order APIs.
Do not add live execution paths.
```

## Tests

```text
valid NFP payload accepted
missing expected value rejected for NFP/CPI
duplicate source_event_id deduped
unsupported event_type quarantined
low-confidence event stored but not queued
ETF map only accepts allowlisted ETFs
```

## Acceptance criteria

```text
macro_events table exists
macro_event_etf_map table exists
valid macro payload persists
invalid payload rejected/quarantined
tests cover happy path and failure path
no trade/candidate/order created
```
