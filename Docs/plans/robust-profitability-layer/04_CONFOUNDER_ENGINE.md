# 04 — Confounder Engine

## Goal

Detect other events that could explain or distort the move.

This prevents Jax from making the classic mistake:

```text
Headline happened.
ETF moved.
Therefore headline caused ETF move.
```

That is often wrong.

## Confounder categories

```text
same_time_macro_release
fed_speaker
treasury_auction
mega_cap_earnings
sector_specific_news
geopolitical_shock
oil_shock
credit_event
bank_stress
legal_regulatory_news
options_expiry
index_rebalance
market_liquidity_event
broker/data_issue
```

## Data sources

Phase 1:

```text
World Monitor events
economic calendar
Jax research triggers
known earnings calendar if available
manual fixtures
source timestamps
```

Later:

```text
earnings calendar provider
Treasury auction schedule
Fed speaker calendar
options expiry calendar
credit spread feeds
```

## Time windows

Check confounders around:

```text
same minute
±5 minutes
±15 minutes
±60 minutes
same session
previous session
```

## Data model

### confounder_events

```sql
CREATE TABLE confounder_events (
    id UUID PRIMARY KEY,
    related_event_id UUID NULL,
    confounder_type TEXT NOT NULL,
    affected_symbols TEXT[] NOT NULL DEFAULT '{}',
    headline TEXT NOT NULL,
    event_time_utc TIMESTAMPTZ NOT NULL,
    severity TEXT NOT NULL,
    confidence NUMERIC NOT NULL,
    reason TEXT NOT NULL,
    source TEXT NULL,
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### event_confounder_links

```sql
CREATE TABLE event_confounder_links (
    id UUID PRIMARY KEY,
    event_id UUID NOT NULL,
    confounder_event_id UUID NOT NULL REFERENCES confounder_events(id),
    impact TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(event_id, confounder_event_id)
);
```

## Impact values

```text
blocks_trade
reduces_confidence
requires_manual_review
reassigns_cause
informational_only
```

## Examples

### CPI + Nvidia shock

```text
CPI neutral.
QQQ drops.
Nvidia has a same-time negative headline.
Result: reassigns_cause or blocks CPI trade.
```

### NFP + wages contradiction

```text
Payrolls beat.
Wages miss badly.
Unemployment rises.
Result: reduces_confidence or conflicted.
```

### Fed statement + Powell reversal

```text
Statement hawkish.
Powell sounds dovish 30 minutes later.
Result: blocks initial candidate.
```

## Codex task

```text
Create Confounder Engine.

Inputs:
- primary event
- event calendar
- nearby research triggers
- known scheduled events
- source timestamps

Outputs:
- confounder list
- impact
- reason
- trade eligibility effect
```

## Tests

```text
same-time mega-cap headline blocks QQQ macro attribution
Fed speaker inside window creates manual_review
Treasury auction near rates move reduces confidence
no nearby events returns clean
low-severity unrelated event informational_only
```

## Acceptance criteria

```text
confounders persisted
links to primary event persisted
candidate evidence includes confounders
blocking confounders prevent candidate creation
