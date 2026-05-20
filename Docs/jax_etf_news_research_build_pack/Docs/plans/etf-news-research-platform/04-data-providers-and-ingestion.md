# 04 — Data Providers and Ingestion

## Goal

Use existing provider APIs where possible. Do not build custom scraping unless absolutely necessary.

## Development / Free-First Provider Stack

### Finnhub

Use for:

- company news
- earnings calendar
- economic context where available
- cheap development data

### Alpaca

Use for:

- historical bars
- real-time ETF data if account supports it
- paper trading integration if needed
- live market stream later

### Existing Calendar Store

Use for:

- high-impact macro events
- scheduled economic releases
- Fed/central-bank dates
- jobs/inflation release context

### NewsAPI

Use only for:

- prototype enrichment
- background research
- manual/dev support

Do not use as live trading truth.

## Paid / Production Provider Stack

### Polygon / Massive

Recommended as production primary provider.

Use for:

- historical candles
- real-time quotes/trades
- ETF price reaction studies
- ticker-linked news
- sentiment/enrichment if available
- production feed health monitoring

### Finnhub Secondary

Use as:

- secondary news source
- cross-check
- fallback enrichment

### Alpaca Backup

Use as:

- secondary market-data stream
- paper trading/broker integration
- redundancy option

## Provider Architecture

Create or extend provider abstraction:

```text
Provider
- id
- type: market_data | news | macro_calendar | broker
- priority
- supports_historical
- supports_live
- latency_class
- cost_tier
- enabled
```

## Ingestion Principles

- Store raw provider payloads.
- Normalize into common event model.
- De-duplicate by canonical key.
- Preserve provider timestamp.
- Preserve ingestion timestamp.
- Store source/provider quality.
- Never let AI invent missing fields.

## Acceptance Criteria

- Historical backfill can run provider by provider.
- Provider failures are visible.
- Duplicate events are not repeatedly inserted.
- Source/provider is stored on every event.
- Production can replay what Jax saw at decision time.
