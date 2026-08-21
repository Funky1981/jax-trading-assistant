# Next Work Package

Status: identified by roadmap; awaiting separate technical-lead authorization; not started

## WP-01.01 - Inventory existing Jax domain contracts before adding new ones

### Purpose

Establish an evidence-backed map of Jax's existing domain vocabulary, ownership boundaries, persistence representations, API/DTO projections, provenance fields, version identities, and audit mechanisms before proposing new canonical contracts. The package prevents duplicate models and makes later Phase 01 contract work an explicit evolution of what exists.

### Major deliverables

- An inventory of existing Instrument, Issuer, Event, Evidence, Observation, ResearchRun, QuantResult, Recommendation, provenance, and audit concepts, including equivalent names and missing concepts.
- An ownership/dependency map across shared contracts, internal domain/decisioning packages, feature modules, runtime DTOs, and persistence schemas.
- A current/target/gap classification identifying contracts to retain, evolve, adapt, deprecate later, or leave feature-local.
- A collision/duplication and compatibility-risk register for WP-01.02 through WP-01.04.
- A recommended canonical home and sequencing proposal only; no broad contract implementation or migration in WP-01.01.

### Acceptance criteria

- The inventory covers the representative domain concepts and all material duplicate representations discovered in code and schema.
- Every finding cites concrete source files/types/tables and identifies current ownership/consumers.
- Existing modular-monolith, artifact-trust, Postgres, paper/live, and execution boundaries are preserved.
- Gaps and alternatives are explicit; no generic event-sourcing platform, duplicate subsystem, or new event bus is introduced.
- Verification is reproducible and the standard review handover is returned before WP-01.02 begins.

### Dependencies

- Phase 00 GO: satisfied 2026-08-21.
- Phase 00 compact evidence/decision record: satisfied by the close-out evidence and decision log.
- Governance: `governance/CODEX-OPERATING-RULES.md`, `governance/DEFINITION-OF-DONE.md`, and `governance/GO-NO-GO-PROCESS.md`.
- Phase documents: `01-canonical-contracts-provenance-audit/README.md`, `GATE.md`, and `REFERENCE-NOTES.md`.

### Likely files/components inspected or documented

- `libs/contracts/domain/` and `libs/contracts/services/`
- `internal/decisioning/core/`, `internal/decisioning/research/`, and `internal/decisioning/persistence/`
- `internal/domain/artifacts/`
- `internal/modules/instruments/`, `assetresolution/`, `macroevents/`, `evidencequality/`, `eventdecisions/`, `candidates/`, and `audit/`
- `cmd/research/` and `cmd/trader/` request/response, store, and API projection types
- `db/postgres/` and `db/postgres/migrations/` tables for events, provenance, artifacts, research, candidates, recommendations, and audit
- Existing architecture/ADR/evidence documentation relevant to ownership and compatibility

The exact affected files will be narrowed by the inventory. No Phase 01 code, schema, API, or migration change is authorized by this close-out.
