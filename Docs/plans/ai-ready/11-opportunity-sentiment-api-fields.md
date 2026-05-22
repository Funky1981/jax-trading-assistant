# Add Sentiment Fields to Opportunity Summary and Detail APIs

## Summary

- Priority: P0
- Phase: Phase 2
- Estimate: 3-5d
- Outcome: explainable sentiment

## Objective

Expose sentiment evidence through Opportunity summary and detail read models so the UI can explain why sentiment affected an opportunity.

## In-scope touchpoints

- API handlers for opportunities or adapted signal/candidate detail
- `frontend/src/data/types.ts`
- Frontend adapter logic from `03-opportunity-adapter.md`

## Implementation notes

- Add `SentimentSummary` with score, label, confidence, time window, source count, source groups, price agreement, top drivers, and limitations.
- Add detail-level fields for sentiment breakdown, source items, invalidation conditions, policy context, risk, and limitations.
- Distinguish missing sentiment, disabled sentiment, sparse sources, and scoring failure.
- Do not expose raw provider payloads as the user-facing contract.
- Preserve compatibility for opportunities that do not have sentiment evidence.

## Acceptance criteria

- Opportunity summaries can include sentiment summary data.
- Opportunity detail includes sentiment score, time window, source count, top drivers, source breakdown, and limitations.
- Missing or unavailable sentiment is shown as a clear state, not as zero.
- TypeScript contracts match backend response fields.

## Suggested validation

- Run backend handler/read-model tests for opportunities with full, sparse, missing, and disabled sentiment.
- Run frontend adapter tests for the same states.
- Run affected UI tests consuming opportunity details.

## Dependencies

- Depends on `10-sentiment-ingest-scoring-aggregates.md`.
- Extends `03-opportunity-adapter.md`.
