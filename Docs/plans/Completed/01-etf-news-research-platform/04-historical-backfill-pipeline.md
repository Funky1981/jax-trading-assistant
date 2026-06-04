# 04 — Historical Backfill Pipeline

## Goal

Fill Jax database with ETF candles, news, macro events, event windows, priced-in scores, and confounders.

## Initial Scope

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

## Jobs

```text
candle backfill
news backfill
macro calendar backfill
event study backfill
priced-in score generation
confounder detection
```

## Suggested Endpoints

```text
POST /research/backfill/events
POST /research/backfill/candles
POST /research/backfill/event-study
GET  /research/backfill/runs/{id}
```

## Idempotency

Backfills must be safe to rerun.

Use canonical keys and upserts where appropriate.

## Acceptance Criteria

- Backfill all 11 ETFs.
- Rerun without duplicates.
- Show provider/data gaps.
- Generate event windows.
- Produce run summaries.
