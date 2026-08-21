# Phase 04 — World Monitor Intelligence

## Purpose
    Turn World Monitor from article collection into corroborated, deduplicated, market-relevant event intelligence.

## Prerequisites
- Phase 02 GO
- Phase 03 core source contracts stable

## Reference systems to inspect at this phase
Fincept NewsMonitor/NewsNLP/NewsCluster/NewsCorrelation, geopolitics, maritime and prediction-market concepts.

## Work packages
- `04.01` — Continuous durable collection and cursor semantics
- `04.02` — Deduplication/event clustering
- `04.03` — Entity/issuer/geography extraction
- `04.04` — Source triangulation and confidence
- `04.05` — Velocity/baseline-deviation signals
- `04.06` — Market-reaction correlation
- `04.07` — Evaluate ACLED/AIS/prediction-market evidence adapters

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
A replayed multi-source event is clustered, corroborated, linked to plausible instruments and accompanied by confidence/unknowns without creating a trade candidate automatically.

See `GATE.md`. Every work package requires independent review before the next one starts.
