# 05 — Liquidity and Execution Quality

## Goal

Prevent trades that are theoretically correct but practically bad.

A good thesis can still lose if execution quality is poor.

## Checks

```text
spread
volume
slippage estimate
market open/close danger
event candle volatility
halt risk
data freshness
broker availability
order book quality if available
ETF liquidity
```

## Phase 1 simple rules

```text
spread too wide = no trade
volume too low = no trade
slippage estimate too high = no trade
first 1-3 minutes after event = no trade
market data stale = no trade
broker unavailable = no trade
```

## Data model

### execution_quality_snapshots

```sql
CREATE TABLE execution_quality_snapshots (
    id UUID PRIMARY KEY,
    symbol TEXT NOT NULL,
    as_of_utc TIMESTAMPTZ NOT NULL,
    spread_percent NUMERIC NULL,
    volume_ok BOOLEAN NOT NULL,
    slippage_estimate_percent NUMERIC NULL,
    market_data_fresh BOOLEAN NOT NULL,
    broker_available BOOLEAN NOT NULL,
    event_volatility_state TEXT NOT NULL,
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL,
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Verdicts

```text
good
acceptable
poor
blocked
insufficient_data
```

## Defaults

```text
max_spread_percent = 0.15
max_slippage_estimate_percent = 0.25
min_volume_ratio = 0.50
event_no_trade_delay_seconds = 180
```

Tune per ETF later.

## Codex task

```text
Create Execution Quality service.

Inputs:
- quote/candle/broker status
- symbol
- event time
- current time

Outputs:
- execution quality snapshot
- verdict
- reasons
```

## Tests

```text
wide spread blocks
stale data blocks
broker unavailable blocks
first minute after CPI blocks
normal liquid ETF passes
missing quote creates insufficient_data
```

## Acceptance criteria

```text
execution quality is part of evidence bundle
blocked execution quality prevents candidate
reasons visible to user
