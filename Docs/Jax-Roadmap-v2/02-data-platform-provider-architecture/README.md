# Phase 02 — Data Platform & Provider Architecture

## Purpose
    Make external data ingestion routine, normalized, health-aware and replaceable.

## Prerequisites
- Phase 01 GO

## Reference systems to inspect at this phase
Fincept DataHub, provider and normalization patterns.

## Work packages
- `02.01` — Provider registry/capability contract
- `02.02` — Raw payload persistence/reference policy
- `02.03` — Normalization and validation pipeline
- `02.04` — Freshness/TTL/last-known-good semantics
- `02.05` — Rate limit/retry/backoff/health instrumentation
- `02.06` — Data source qualification registry

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
A new provider can be added behind a stable adapter, produces canonical validated data with raw provenance, and exposes freshness/health without changing research logic.

See `GATE.md`. Every work package requires independent review before the next one starts.
