# Phase 04 Reference Notes — World Monitor

## Fincept concepts
- NewsMonitorService: long-lived monitoring.
- NewsNlpService: structured NLP enrichment.
- NewsClusterService: deduplication/clustering.
- NewsCorrelationService: velocity spikes, triangulation, focal points, baseline deviation and prediction-market correlation.
- GeopoliticsService / MaritimeService: separate alternative-evidence domains.
- Polymarket service/scripts: prediction markets as an evidence source.

## Candidate external evidence
- ACLED: structured conflict/event data.
- AISStream: maritime WebSocket data; documentation currently states beta/no SLA, so use as corroboration rather than sole truth.
- Polymarket: public market/event/orderbook data as read-only market-implied evidence.

## Jax target
World Monitor should emit an **event evidence object**, not a trade:
event cluster -> entities/geography -> corroboration -> abnormality -> plausible exposures -> observed market reaction -> uncertainty.
