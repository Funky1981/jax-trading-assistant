# Real Market Data Ingestion

Currently, the Jax trading assistant uses stubbed or limited market data through a `MarketData` interface. To trade on live markets, you must ingest real‑time or near‑real‑time prices and historical candles from a reliable provider.

## Why it matters

Event detection (e.g. gap identification), strategy generation and risk sizing all depend on accurate, timely market data. Using stale or fake data in production leads to false signals and poor risk calculations.

## Tasks

1. **Select a market data provider**
   - Evaluate APIs such as **Polygon.io**, **Alpaca Market Data**, **Alpha Vantage**, **Finnhub** or **IEX Cloud** for equity and ETF pricing. Consider cost, rate limits, and coverage.
   - Ensure the provider offers both historical candles and current quotes with at least daily resolution (more granular if intraday strategies are desired).

2. **Implement an ingest service**
   - Flesh out the `jax-ingest` service under `services/jax-ingest` (currently a skeleton).
   - Create a client library for the chosen market data provider, encapsulating API keys and request logic.
   - Schedule regular jobs (cron or a simple loop) to fetch daily candles for all tracked symbols and store them in UTCP or another time‑series database.

3. **Integrate with the existing `MarketData` interface**
   - Ensure that `MarketData.GetDailyCandles` uses the ingested data rather than hitting the provider on each request.
   - Add caching or a read‑through mechanism to reduce redundant API calls.
   - Validate and normalise incoming data (e.g. handle splits, missing days, timezone differences).

4. **Observability and error handling**
   - Log failed fetches and rate‑limit responses. Implement exponential backoff and alerting if data ingestion repeatedly fails.
   - Expose metrics on latency, data freshness and error counts.

5. **Testing**
   - Mock the external API to test error scenarios and edge cases (e.g. missing data, partial days).
   - Write integration tests that hit a sandbox or free tier of the provider to ensure your client works as expected.

6. **Documentation and run book**
   - Document the configuration needed (API keys, symbol lists, schedules) in `Docs/`.
   - Provide troubleshooting steps for common ingestion failures and how to rotate keys or switch providers.