# Build Sentiment Ingest, Scoring, and Aggregate Pipeline

## Summary

- Priority: P0
- Phase: Phase 2
- Estimate: 5-8d
- Outcome: sentiment feature layer

## Objective

Create the backend sentiment feature layer that turns source documents into per-symbol sentiment aggregates usable by scanner ranking, explainability, alerts, and research.

## In-scope touchpoints

- New backend worker/service modules
- Storage migrations
- Existing source/news ingestion paths where applicable

## Implementation notes

- Add storage for `news_items`, `sentiment_events`, `sentiment_aggregates`, and `opportunity_sentiment_snapshots`.
- Ingest source documents with metadata, source family, timestamps, symbol mapping, and provenance.
- Score sentiment per document with score, label, confidence, drivers, and limitations.
- Roll up per-symbol aggregates by configured time windows and source groups.
- Capture immutable sentiment snapshots when an Opportunity is created or materially rescored.
- Use a hybrid-ready internal interface so scoring can later call an external provider or local model.
- Keep approval policy separate from sentiment scoring.

## Acceptance criteria

- Source documents can be stored and scored.
- Aggregates can be queried by symbol and window.
- Sparse-source and low-confidence states are represented explicitly.
- Opportunity sentiment snapshots are auditable and not recomputed silently after review.
- The pipeline can be disabled without breaking existing trading workflows.

## Suggested validation

- Run focused Go tests for ingestion, scoring interface, aggregation, and snapshot persistence.
- Run migration validation against the repo's normal database migration workflow.
- Run golden/replay checks before using sentiment in behavior-sensitive scanner or trading decisions.

## Dependencies

- Supports `11-opportunity-sentiment-api-fields.md`, `13-sentiment-alert-rules-inbox-categories.md`, and `15-research-backtest-sentiment-options.md`.
