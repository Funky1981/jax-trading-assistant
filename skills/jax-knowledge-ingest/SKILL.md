---
name: jax-knowledge-ingest
description: Knowledge database ingestion workflow for `knowledge/md`, `tools/sql/schema.sql`, and `tools/cmd/ingest`. Use when adding or modifying markdown knowledge sources, applying schema updates, running dry-run ingestion, or executing full ingest into local Postgres.
---

# Jax Knowledge Ingest

Ingest knowledge content safely and reproducibly with dry-run-first discipline.

## Workflow

1. Start local knowledge Postgres.
2. Apply schema before ingest if schema changed.
3. Run dry-run ingest first.
4. Run full ingest only after dry-run is clean.
5. Verify row counts or expected ingest output.
6. Stop services when done.

Use `scripts/knowledge-cycle.ps1` to run the sequence.

## Guardrails

- Keep source content in `knowledge/md/`.
- Treat `tools/sql/schema.sql` as schema source of truth for knowledge DB.
- Prefer deterministic ingest options and explicit DSN settings.

## Commands

- `scripts/knowledge-cycle.ps1 -Mode all`
- `scripts/knowledge-cycle.ps1 -Mode ingest -DryRun`
- `scripts/knowledge-cycle.ps1 -Mode down`

Use `references/ingest-notes.md` for failure triage.
