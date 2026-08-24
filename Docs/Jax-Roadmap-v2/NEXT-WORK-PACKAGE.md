# Next Work Package

Status: WP-02.03 implementation complete / technical-lead review pending

## Immediate next action - review WP-02.03

Path: `02-data-platform-provider-architecture/WP-02.03-normalization-and-validation-pipeline.md`

Phase 01 is complete with **GO PHASE 01**, and WP-02.01/WP-02.02 are accepted. WP-02.03 has now implemented the provider-owned deterministic normalization contract, exact capability/schema/target routing, typed stage failures, canonical and raw-provenance validation, loss/omission metadata, strict normalizer identity/version binding, storage-port-to-normalization proof, and repeatability gate without migrating runtime providers or adding network calls, persistence, migrations, freshness, fallback, retries, health, or source scoring.

The technical lead must review the implementation and `../evidence/WP-02.03-NORMALIZATION-VALIDATION-PIPELINE.md`, then issue `GO`, `CONDITIONAL GO`, `NO-GO`, or `ROADMAP CHANGE` under `governance/GO-NO-GO-PROCESS.md`.

The exact next roadmap package after an accepted WP-02.03 decision and separate authorization is `WP-02.04 - Freshness/TTL/last-known-good semantics`.

Do not begin WP-02.04 from this implementation record.
