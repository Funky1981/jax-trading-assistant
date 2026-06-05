# Introduce Opportunity Adapter Over Signals, Candidates, and Approvals

## Summary

- Priority: P0
- Phase: Phase 1
- Estimate: 2-4d
- Outcome: one user-facing object

## Objective

Create a frontend read-model adapter that maps current signals, candidate trades, and approval queue items into one user-facing `Opportunity` model.

## In-scope touchpoints

- `frontend/src/data/types.ts`
- New frontend adapter module near existing data services
- Existing API handlers only if needed for stable read fields

## Implementation notes

- Define a user-facing `OpportunitySummary` type with `id`, `symbol`, `signalType`, `confidenceBand`, `summary`, `detectedAt`, `expiresAt`, `route`, `routeReason`, `sentimentSummary`, and `status`.
- Support route values for manual-allowed, approval-required, and blocked outcomes.
- Keep existing backend resources intact; this first pass should adapt existing shapes rather than force a backend rewrite.
- Preserve enough source metadata for debugging, such as original entity type and original ID.
- Use plain-language labels: Opportunity, Proposed trade, Saved setup, Opportunity queue, Run deeper analysis.

## Acceptance criteria

- Existing signal, candidate, and approval data can be normalized into one Opportunity summary list.
- The adapter is deterministic and covered by fixture tests.
- The adapter does not silently drop policy, expiry, or route information.
- Unknown or partial source records produce a safe degraded Opportunity state rather than crashing the UI.

## Suggested validation

- Run targeted adapter unit tests using representative signal, candidate, approval, partial, and blocked fixtures.
- Run TypeScript type checks for `frontend/src/data`.
- Run affected UI tests that consume the adapter.

## Dependencies

- Enables `02-ai-trading-opportunity-feed.md`.
- Later extended by `11-opportunity-sentiment-api-fields.md`.
