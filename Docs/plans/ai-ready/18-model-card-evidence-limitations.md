# Add Model-Card-Style Evidence and Limitation Blocks

## Summary

- Priority: P2
- Phase: Phase 3
- Estimate: 2-4d
- Outcome: transparency maturity

## Objective

Make Opportunity and research explanations more transparent by showing intended use, limitations, sparse evidence, and confidence caveats.

## In-scope touchpoints

- Opportunity drawer
- Research result summary
- Audit/evidence data where available

## Implementation notes

- Add reusable evidence and limitation blocks for Opportunity detail and backtest results.
- Include intended use, not-for-use warnings, source coverage, sparse data caveats, confidence caveats, and invalidation conditions.
- Keep copy short and contextual; do not turn every detail screen into a policy document.
- Use the same limitation vocabulary across AI Trading, Approvals, Notifications, and Research.

## Acceptance criteria

- Every Opportunity detail includes limitation and intended-use cues.
- Sentiment limitations are visible when evidence is sparse, stale, low confidence, or unavailable.
- Research result summaries disclose sentiment data coverage and assumptions.
- Users can understand why sentiment affected an Opportunity without raw scores alone.

## Suggested validation

- Run component tests for limitation block variants.
- Run UX/content review with representative full, sparse, stale, and unavailable evidence.
- Include novice comprehension testing when available.

## Dependencies

- Builds on `12-opportunity-drawer-sentiment-explanation.md` and `15-research-backtest-sentiment-options.md`.
