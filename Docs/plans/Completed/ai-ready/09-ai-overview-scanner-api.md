# Add AI Overview and Scanner State API Endpoints

## Summary

- Priority: P0
- Phase: Phase 2
- Estimate: 3-5d
- Outcome: AI home data model

## Objective

Expose backend read models for AI Trading Home and scanner settings so the frontend no longer relies only on local placeholders and scattered existing endpoints.

## In-scope touchpoints

- `cmd/trader/frontend_api.go`
- New backend handlers/services as needed

## Implementation notes

- Add `GET /api/v1/ai/overview` with scanner summary, opportunity counts, policy summary, and channel summary.
- Add `GET /api/v1/ai/scanner` to fetch scanner state.
- Add `PUT /api/v1/ai/scanner` to save scanner state.
- Scanner state should include `enabled`, `assetScope`, `symbols`, `universePreset`, `intervalSeconds`, `minimumConfidence`, `sentiment`, `status`, `lastScanCompletedAt`, `nextScanAt`, `channels`, and `policy`.
- Sentiment settings should include enabled state, source scope, window, threshold, minimum source count, source trust weighting mode, and mode.
- Preserve existing signal, candidate, and approval endpoints for compatibility.

## Acceptance criteria

- AI Trading Home can load one coherent overview read model.
- Scanner settings can be read and saved.
- Sentiment settings are represented even if the scoring pipeline is still disabled.
- Invalid scanner settings return clear validation errors.
- Existing frontend consumers of signals and approvals continue to work.

## Suggested validation

- Run focused Go tests for the new handlers and validation.
- Run frontend data-client tests if a TypeScript service is added.
- Run API smoke checks for `GET /api/v1/ai/overview`, `GET /api/v1/ai/scanner`, and `PUT /api/v1/ai/scanner`.

## Dependencies

- Enables persistence for `04-scanner-sentiment-controls.md`.
- Supports `02-ai-trading-opportunity-feed.md`.
