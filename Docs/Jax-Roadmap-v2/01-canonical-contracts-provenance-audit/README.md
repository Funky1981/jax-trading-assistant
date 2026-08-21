# Phase 01 — Canonical Contracts, Provenance & Audit

**Status:** eligible after Phase 00 GO; not started. WP-01.01 is the first incomplete package and awaits technical-lead authorization.

## Purpose
    Create the stable vocabulary and audit model required by all later data, research, quant and decision capabilities.

## Prerequisites
- Phase 00 GO

## Reference systems to inspect at this phase
Fincept bounded-context architecture and Alpha Arena replay/audit.

## Work packages
- `01.01` — Inventory existing Jax domain contracts before adding new ones
- `01.02` — Define canonical Instrument/Issuer/Event/Evidence/Observation/ResearchRun/QuantResult/Recommendation contracts
- `01.03` — Define provenance/version identity and immutable evidence references
- `01.04` — Define replay/audit event model and compatibility strategy

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
A reviewer can trace a representative output to immutable/identifiable inputs, source/provider, version and timestamp without depending on transient logs.

See `GATE.md`. Every work package requires independent review before the next one starts.
