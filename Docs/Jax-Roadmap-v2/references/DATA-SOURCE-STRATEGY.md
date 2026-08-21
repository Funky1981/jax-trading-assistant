# Jax Data Source Strategy

Do not optimize for connector count. Optimize for evidence quality, historical replay and failure transparency.

## Tier 1 — Core first
- Existing validated market-data provider(s) for real OHLCV/quotes.
- SEC EDGAR for filings/XBRL.
- FRED/ALFRED for macro observations and historical vintages.
- Economic release/calendar data.
- Treasury/EIA/CBOE/CFTC where a concrete research use case is defined.

## Tier 2 — Event corroboration
- Multiple reputable news feeds.
- ACLED for structured conflict events where terms permit.
- Prediction-market public data as a secondary evidence class.
- Maritime/AIS only for event types where shipping behaviour is materially relevant.

## Tier 3 — Specialist/alternative
- Satellite/environmental/agricultural/climate/trade sources only when a research hypothesis demonstrates value.

## Every source record must capture
- owner/provider and source type;
- official vs aggregator;
- licence/terms;
- auth/key requirements;
- cost;
- rate limits;
- update latency/frequency;
- historical depth;
- revision/vintage behaviour;
- reliability/SLA;
- schema stability;
- allowed storage/redistribution;
- Jax use cases;
- trust tier;
- fallback;
- status: candidate / trial / approved / deprecated / rejected.

## Rule
No AI research prompt should receive source data whose age, origin and validation state Jax cannot identify.
