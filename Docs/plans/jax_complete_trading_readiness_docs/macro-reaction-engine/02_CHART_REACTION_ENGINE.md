# 02 — Candle / Chart Reaction Engine

## Goal

Jax must check the market reaction after news before suggesting a trade.

A headline is not enough. Jax needs to answer:

```text
Did the market actually react?
Which ETFs reacted?
Was the reaction in the expected direction?
Is the move already too extended?
Is the move noisy/fake/choppy?
```

## Inputs

```text
macro_event_id
event_time_utc
candidate ETF symbols
timeframes
candle source
```

Recommended ETFs for phase 1:

```text
SPY
QQQ
IWM
TLT
DIA
XLK
XLF
XLE
SMH
SOXX
GLD
```

## Candle windows

For each event and ETF, capture:

```text
pre_event_30m
pre_event_5m
post_event_5m
post_event_15m
post_event_30m
post_event_60m
session_to_now
```

## Data model

### macro_reaction_snapshots

```sql
CREATE TABLE macro_reaction_snapshots (
    id UUID PRIMARY KEY,
    macro_event_id UUID NOT NULL REFERENCES macro_events(id),
    symbol TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    pre_price NUMERIC NOT NULL,
    post_price NUMERIC NOT NULL,
    change_abs NUMERIC NOT NULL,
    change_percent NUMERIC NOT NULL,
    high_after NUMERIC NULL,
    low_after NUMERIC NULL,
    volume_ratio NUMERIC NULL,
    atr_ratio NUMERIC NULL,
    direction TEXT NOT NULL,
    confirms_event BOOLEAN NOT NULL,
    too_extended BOOLEAN NOT NULL,
    noisy BOOLEAN NOT NULL,
    reason TEXT NOT NULL,
    raw_candles JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(macro_event_id, symbol, timeframe)
);
```

## Direction values

```text
up
down
flat
whipsaw
unknown
```

## Confirmation examples

### Hot CPI / strong jobs

Expected:

```text
QQQ down
SPY down
TLT down
yields proxy up if available
DXY up if available
```

### Cool CPI / dovish Fed

Expected:

```text
QQQ up
SPY up
TLT up
yields proxy down if available
DXY down if available
```

## Extension rules

Do not chase if:

```text
QQQ already moved more than configured max_event_move_percent
price is far beyond event candle range
spread/volatility too high
ATR expansion extreme
no clean stop level exists
```

Suggested defaults:

```text
max_event_move_percent:
  QQQ: 2.5
  SPY: 1.8
  TLT: 1.5
  IWM: 2.2

minimum_confirmation_move_percent:
  QQQ: 0.35
  SPY: 0.25
  TLT: 0.25
```

## Codex task

```text
Build a chart reaction engine that uses the existing market candles API/provider path.

For a macro event and ETF list:
1. Fetch candles around event_time_utc.
2. Calculate pre/post change.
3. Mark confirms_event, too_extended, noisy.
4. Persist reaction snapshots.
5. Return a structured reaction summary.

Do not create candidate trades in this phase.
```

## Tests

```text
hot CPI + QQQ down = confirms_event true
hot CPI + QQQ up = confirms_event false
tiny move = confirms_event false
huge move beyond max threshold = too_extended true
missing candles = reaction status unavailable
whipsaw candles = noisy true
```

## Acceptance criteria

```text
reaction snapshots persisted
candle failure does not crash research
no candles = no trade candidate
no confirmation = no trade candidate
too_extended = no trade candidate unless strategy explicitly allows pullback watch
```
