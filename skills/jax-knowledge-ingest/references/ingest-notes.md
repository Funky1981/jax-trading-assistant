# Knowledge Ingest Notes

## Source Paths

- content: `knowledge/md/`
- schema: `tools/sql/schema.sql`
- ingestor: `tools/cmd/ingest`

## Makefile Shortcuts

- `make knowledge-up`
- `make knowledge-schema`
- `make knowledge-ingest-dry`
- `make knowledge-ingest`
- `make knowledge-down`

## Common Failures

- DB unavailable: start compose service and retry.
- schema mismatch: run schema apply before ingest.
- malformed markdown metadata: fix source file and re-run dry-run.
