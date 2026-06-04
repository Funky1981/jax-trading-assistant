# 03 — Data Providers and Ingestion

## Free/Cheap Development Providers

### Finnhub

Use for:

```text
company news
earnings calendar
cheap development data
```

### Alpaca

Use for:

```text
historical bars
real-time ETF data if available
paper trading integration if needed
```

### Existing Calendar Store

Use for:

```text
macro releases
central-bank dates
high-impact scheduled events
```

### NewsAPI

Use only as helper/prototype data.

Do not rely on it for live trading.

## Paid Production Providers Later

### Polygon / Massive

Recommended production primary.

Use for:

```text
historical candles
real-time quotes
ticker-linked news
ETF event studies
production feed health
```

### Finnhub Secondary

Use as secondary cross-check/enrichment.

### Alpaca Backup

Use as market-data/broker backup where appropriate.

## Provider Rules

- Store raw payload.
- Normalise into event model.
- Deduplicate.
- Preserve source timestamp.
- Preserve ingestion timestamp.
- Store source quality.
- Never let AI invent missing fields.
