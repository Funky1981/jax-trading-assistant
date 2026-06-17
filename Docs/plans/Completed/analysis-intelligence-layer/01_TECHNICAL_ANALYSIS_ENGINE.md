# 01 — Technical Analysis Engine

## Goal

Give Jax a deterministic technical analysis engine so it knows what to look for in charts.

This is not a vague AI prompt. It is a structured checklist and scoring system.

## What Jax must analyse

For each candidate ETF, Jax must inspect:

```text
trend
market structure
support/resistance
event candle range
breakout/breakdown
VWAP behaviour
moving average context
volume expansion
volatility/ATR expansion
relative strength
gap behaviour
false breakout risk
entry quality
stop quality
reward:risk
chase risk
```

## Supported instruments in phase 1

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

## Required chart windows

```text
daily: 20, 50, 200 session context
intraday pre-event: 60m, 30m, 15m, 5m
post-event: 5m, 15m, 30m, 60m
session-to-now
```

## Technical checks

### 1. Trend context

Jax should determine:

```text
uptrend
downtrend
range
transition
high-volatility chop
unknown
```

Rules can use:

```text
price vs 20/50/200 moving averages
higher highs / higher lows
lower highs / lower lows
slope of moving averages
ATR regime
```

### 2. Key levels

Jax should identify:

```text
pre-event high
pre-event low
previous day high
previous day low
session open
VWAP
major daily support/resistance
recent swing high/low
gap levels
```

### 3. Event candle behaviour

For macro/news events, the event candle matters.

Check:

```text
first reaction direction
event candle body size
upper/lower wick size
close position inside event range
break of pre-event range
reclaim/fail of pre-event range
```

### 4. Confirmation candle

Never rely on the first candle alone for Fed/CPI/jobs.

Jax should check whether the next confirmation window supports the first move:

```text
5m confirmation
15m confirmation
30m confirmation for Fed pressers
```

### 5. VWAP behaviour

Useful intraday checks:

```text
price below VWAP and rejecting = bearish confirmation
price above VWAP and holding = bullish confirmation
reclaim VWAP after selloff = bearish thesis weakened
lose VWAP after rally = bullish thesis weakened
```

### 6. Volume and volatility

Check:

```text
volume_ratio = post_event_volume / baseline_volume
atr_ratio = post_event_range / recent_atr
```

Interpretation:

```text
high volume + clean direction = confirmation
high volume + whipsaw = danger
low volume = weak confirmation
ATR too stretched = chase risk
```

### 7. Relative strength

Compare ETF against SPY.

Examples:

```text
QQQ weaker than SPY during hawkish rates = tech pressure confirmed
TLT weaker during hot CPI = rates pressure confirmed
XLF stronger during rising rates = possible sector rotation
GLD stronger during geopolitical panic = safe-haven confirmation
```

### 8. Multi-ETF confirmation

For macro events, one ETF alone is not enough.

Example: hot CPI / strong jobs.

Expected confirmation:

```text
QQQ down
SPY down
TLT down
IWM down or weak
```

If QQQ falls but TLT rallies, Jax must flag conflict.

## Technical result values

```text
confirmed_bullish
confirmed_bearish
watch_only
no_confirmation
conflicting
too_extended
whipsaw
insufficient_data
```

## Data model

### technical_analysis_snapshots

```sql
CREATE TABLE technical_analysis_snapshots (
    id UUID PRIMARY KEY,
    macro_event_id UUID NULL,
    symbol TEXT NOT NULL,
    analysis_time_utc TIMESTAMPTZ NOT NULL,
    timeframe TEXT NOT NULL,
    trend_state TEXT NOT NULL,
    structure_state TEXT NOT NULL,
    key_levels JSONB NOT NULL,
    event_reaction JSONB NOT NULL,
    volume_volatility JSONB NOT NULL,
    relative_strength JSONB NOT NULL,
    technical_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL,
    invalidation_rules TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Technical score

```text
Trend alignment:             0–20
Level break/hold quality:    0–20
Event reaction quality:      0–20
Volume/ATR confirmation:     0–15
Relative strength:           0–15
Entry/stop quality:          0–10
```

Total:

```text
0–39   no trade
40–59  watch only
60–74  possible candidate
75+    strong candidate
```

## Hard technical blocks

Regardless of score:

```text
no candles = no trade
no stop level = no trade
reward:risk below threshold = no trade
price too extended = no trade
whipsaw = no trade
confirmation missing = no trade
conflicting ETF basket = no trade
```

## Codex task

```text
Create a deterministic Technical Analysis Engine.

Inputs:
- symbol
- candles
- event_time_utc
- event scenario
- benchmark symbol, usually SPY

Outputs:
- technical snapshot
- technical score
- verdict
- reasons
- invalidation rules
```

## Tests

```text
break below pre-event low + VWAP rejection = bearish confirmation
break above pre-event high + VWAP hold = bullish confirmation
first candle down then reclaim VWAP = no bearish candidate
huge move beyond chase threshold = too_extended
missing candles = insufficient_data
QQQ down but TLT up on hot CPI = conflicting
```

## Acceptance criteria

```text
technical snapshot persisted
score calculated deterministically
verdict is explainable
hard blocks override score
candidate generator cannot bypass technical verdict
```
