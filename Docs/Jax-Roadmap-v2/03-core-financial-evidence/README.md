# Phase 03 — Core Financial Evidence

**Status:** IN PROGRESS. WP-03.01, WP-03.02 and WP-03.03 are accepted
**COMPLETE / GO**, including their documented temporal/vintage and evidence
closures. WP-03.04 is implemented and awaits separate technical-lead review;
it is not yet accepted COMPLETE / GO.

## Purpose
    Populate the data platform with the smallest high-value set of market, corporate and macro sources required for useful US equity/ETF research.

## Prerequisites
- Phase 02 GO

## Reference systems to inspect at this phase
Fincept connector catalogue plus official SEC/FRED and selected market/economic sources.

## Work packages
- `03.01` — Market price/OHLCV provider hardening
- `03.02` — SEC/EDGAR filings and XBRL company facts
- `03.03` — FRED/ALFRED macro observations and vintages
- `03.04` — Economic release/calendar ingestion
- `03.05` — Treasury/EIA/CBOE/CFTC source evaluation and first approved integrations
- `03.06` — Evidence-quality cross-source checks

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
Jax can construct a source-linked evidence packet for a representative US equity/ETF using real market, company and macro evidence without relying on model memory.

See `GATE.md`. Every work package requires independent review before the next one starts.
