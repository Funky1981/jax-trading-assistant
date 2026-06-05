# Add Save-to-Paper or Live Handoff Preserving Sentiment Flags

## Summary

- Priority: P1
- Phase: Phase 2
- Estimate: 2-3d
- Outcome: live/paper conversion

## Objective

Let users save a sentiment-enabled backtest result as a paper-ready or live-ready setup without losing the sentiment feature configuration.

## In-scope touchpoints

- Research result action
- New backend endpoint for save-to-paper/live setup

## Implementation notes

- Add `POST /api/v1/backtests/runs/{id}/save-paper-setup` or equivalent route if the repo already has a better pattern.
- Persist selected feature flags, thresholds, source scope, window, and sentiment mode.
- Default to paper-ready setup unless live readiness and permissions already exist.
- Carry provenance from the backtest run into the saved setup.
- Show a confirmation that explains which sentiment settings were saved.

## Acceptance criteria

- A sentiment-enabled result can be saved as a paper-ready setup.
- Saved setup preserves sentiment feature flags and thresholds.
- Live handoff remains gated by existing readiness and permission controls.
- Users can inspect what was saved before activation.

## Suggested validation

- Run backend tests for save endpoint, permission/readiness checks, and persisted config.
- Run frontend tests for result action and confirmation state.
- Run paper pilot smoke checks if this touches existing paper setup activation.

## Dependencies

- Depends on `15-research-backtest-sentiment-options.md`.
