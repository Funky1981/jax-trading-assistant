# Current State and Gaps (based on repo inspection)

## Confirmed current state (from repo)

### Services
- `services/ib-bridge` exists (FastAPI + `ib_insync`) and is wired in `docker-compose.yml`.
- `services/jax-market` ingests market data into Postgres and serves:
  - `/health`, `/metrics`, `/metrics/prometheus`
  - **No `/tools` endpoint yet**
- `services/jax-api`:
  - Registers local UTCP tools for `risk` and `backtest`
  - Uses UTCP client for market/risk/storage/dexter/memory
  - Current event detection is **daily-candle gap** only
- `services/jax-trade-executor`:
  - Executes approved signals via IB bridge
  - Risk gates exist (max open positions, max daily loss based on stored risk amounts)
  - No concept of “flatten by close”
- `services/jax-signal-generator`:
  - Generates RSI/MACD/MA signals (not aligned to the same-day event strategies)

### Config
- `config/providers.json` points market provider to `http://market-service:8080/tools`
- Docker runs `jax-market:8095` → endpoint mismatch

### Backtest
- `libs/utcp/backtest_local_tools.go` generates **fake** stats; not a backtest.

## Critical gaps blocking research-grade strategy testing
1. No intraday candle tool path (`market.get_candles` via UTCP) served by `jax-market`.
2. Market provider endpoint mismatch in `providers.json`.
3. Backtest is not real.
4. Strategy configs are mostly descriptive text (e.g., `"entryRule": "..."`).
5. No run tracking / parameter sweeps / reproducible research project structure.
6. No automatic “flat by close” enforcement.

## Implication
Until the above is fixed, you can run pipelines, but you cannot trust research results for same-day strategies.
