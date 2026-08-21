# Phase 03 Reference Notes — Financial Evidence

## Fincept source catalogue value
Use Fincept to discover coverage, then prefer authoritative sources where practical.

High-value candidates:
- SEC/EDGAR for filings, submissions and XBRL facts;
- FRED/ALFRED for macro series and historical vintages;
- BLS/BEA/Treasury/EIA for first-party economic/energy data;
- CBOE/CFTC for volatility/options/futures positioning;
- validated market-data providers for prices/OHLCV.

## External source findings
SEC's official `data.sec.gov` APIs expose submissions and XBRL company facts without API keys and provide bulk archives. FRED/ALFRED expose revision/vintage semantics, critical for leakage-safe historical replay.

## Jax target
Start with a small high-trust source set. A source is not approved until storage rights, rate limits, cost, latency, historical depth, revision semantics and failure behaviour are recorded in the Source Registry.
