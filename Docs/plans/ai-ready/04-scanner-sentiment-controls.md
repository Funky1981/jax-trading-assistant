# Add Scanner State UI With Sentiment Control Placeholders

## Summary

- Priority: P0
- Phase: Phase 1
- Estimate: 2-3d
- Outcome: visible scanner and sentiment config

## Objective

Expose scanner state and sentiment controls in AI Trading so users understand what the AI is watching and how sentiment will contribute.

## In-scope touchpoints

- `frontend/src/pages/AiTradingPage.tsx`
- New scanner settings component
- `frontend/src/data/types.ts`

## Implementation notes

- Add a shared scanner settings card for enabled state, asset scope, symbols/universe, interval, and minimum confidence.
- Include sentiment controls for source scope, time window, minimum sentiment threshold, minimum source count, source trust weighting mode, and sentiment mode.
- Sentiment mode options should be filter, rank boost, and required feature where supported; default beginner copy should favor boost/filter.
- Show "placeholder" or disabled states honestly until Phase 2 backend endpoints exist.
- Use plain-language labels instead of raw-only scores.

## Acceptance criteria

- Scanner controls are visible on AI Trading.
- Sentiment scope, time window, and threshold are visible in Phase 1.
- Disabled or unsupported controls explain their status without blocking the main workflow.
- Settings changes are either persisted through existing state or clearly marked as not yet connected.

## Suggested validation

- Run targeted component tests for default, enabled, disabled, and unsupported states.
- Run accessibility checks for form labels and controls.
- Manually verify mobile wrapping and that text does not overflow compact controls.

## Dependencies

- Complements `02-ai-trading-opportunity-feed.md`.
- Phase 2 persistence depends on `09-ai-overview-scanner-api.md`.
