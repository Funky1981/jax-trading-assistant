# Add Sentiment Options to Research and Backtest Config

## Summary

- Priority: P1
- Phase: Phase 2
- Estimate: 3-5d
- Outcome: testable sentiment strategy

## Objective

Allow users to run backtests with sentiment disabled, sentiment as a filter, or sentiment as a boost from the guided Research flow.

## In-scope touchpoints

- Research APIs
- Research wizard
- Analysis result rendering

## Implementation notes

- Add `BacktestSentimentConfig` with enabled, mode, source scope, window, threshold, decay mode, weighting mode, and divergence enabled.
- Supported beginner modes are disabled, filter, and boost.
- Advanced controls may expose source weighting and decay modes.
- Result summary should show whether sentiment was enabled and how it contributed.
- Handle incomplete historical sentiment data explicitly instead of implying full coverage.

## Acceptance criteria

- Backtests can run with sentiment disabled.
- Backtests can run with sentiment as filter.
- Backtests can run with sentiment as boost.
- Result summary includes sentiment contribution when enabled.
- Missing historical sentiment coverage is disclosed in the run result.

## Suggested validation

- Run backend tests for backtest request parsing and sentiment feature handling.
- Run frontend wizard tests for request construction.
- Run analysis result rendering tests for enabled, disabled, and missing sentiment states.
- Run replay/golden checks if sentiment features affect deterministic strategy outputs.

## Dependencies

- Builds on `07-research-wizard-v1.md`.
- Depends on sentiment feature data from `10-sentiment-ingest-scoring-aggregates.md`.
