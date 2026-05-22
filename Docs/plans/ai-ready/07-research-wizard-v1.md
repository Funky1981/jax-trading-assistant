# Wrap Research in Guided Wizard V1

## Summary

- Priority: P1
- Phase: Phase 1
- Estimate: 3-5d
- Outcome: beginner backtests

## Objective

Create a no-JSON beginner path for backtesting while preserving expert access to advanced research settings.

## In-scope touchpoints

- `frontend/src/pages/ResearchPage.tsx`
- New wizard components near existing research UI

## Implementation notes

- Add a guided Research entry path with strategy template, market, period, and optional sentiment feature step.
- Keep raw JSON, artifact IDs, project-grid JSON, dataset internals, and feature tuning under Advanced.
- If data is missing, explain the missing data in product language and link to the appropriate operator path instead of exposing repo-structure instructions as the primary message.
- Preserve the existing Analysis page depth for metrics, trades, events, timeline, and export.
- Sentiment options can be visible but disabled or marked unavailable until Phase 2 support lands.

## Acceptance criteria

- Research offers a guided path that can be used without raw JSON.
- Advanced controls remain available without dominating the first-run flow.
- Missing-data states are understandable to non-engineers.
- Wizard output can launch the existing backtest flow or produce the same request shape currently accepted by the backend.

## Suggested validation

- Run targeted frontend tests for wizard steps and request construction.
- Run existing research hook/page tests if present.
- Manually test missing dataset, valid run, cancelled run, and advanced settings visibility.

## Dependencies

- Later extended by `15-research-backtest-sentiment-options.md`.
