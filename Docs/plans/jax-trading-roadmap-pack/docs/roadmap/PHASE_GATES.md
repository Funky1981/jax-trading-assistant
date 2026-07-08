# Jax Phase Gates

## Phase 0 Gate — Safety Baseline

Pass only if:

- Live execution is disabled by default.
- Human approval is mandatory.
- Kill switch exists.
- All candidate decisions are logged.

## Phase 1 Gate — Candidate Completeness

Pass only if every candidate has:

- Instrument.
- Direction.
- Setup type.
- Catalyst.
- Entry.
- Stop.
- Target.
- Invalidation point.
- Risk plan.
- Evidence summary.

## Phase 2 Gate — Evidence Quality

Pass only if:

- Evidence is structured.
- Sources/timestamps are stored.
- Contradictory evidence is tracked.
- Data quality is scored.

## Phase 3 Gate — Risk Validity

Pass only if:

- Position size is calculated.
- Slippage-adjusted loss is calculated.
- Max loss is within risk budget.
- Leverage is disabled unless explicitly allowed.

## Phase 4 Gate — Paper Approval Loop

Pass only if:

- Human can approve/reject.
- Rejections are logged.
- Paper execution records fills.
- Trade review is created after close.

## Phase 5 Gate — Paper Trading Evidence

Pass only if:

- Several months of paper trades exist.
- Expectancy is calculated.
- Drawdown is measured.
- Jax beats simple baselines.
- Rule-following is good.

## Phase 6 Gate — Tiny Live Readiness

Pass only if:

- Broker reconciliation works.
- Slippage is measured.
- Kill switch tested.
- Risk limits are enforced.
- Human approval remains mandatory.

## Phase 7 Gate — Scale-Up

Pass only if:

- Tiny live trading shows process discipline.
- Strategy still has positive expectancy.
- Drawdown remains acceptable.
- Slippage does not break the model.
- No emotional overrides are causing damage.

## Phase 8 Gate — Leverage Review

Pass only if:

- Leverage has a specific purpose.
- Strategy has proven edge.
- Stop-loss and slippage behaviour are reliable.
- Broker reconciliation is stable.
- The risk engine calculates margin and liquidation exposure.
