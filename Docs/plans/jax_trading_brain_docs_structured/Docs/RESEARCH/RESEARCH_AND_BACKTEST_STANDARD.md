# Research and Backtest Standard

## Purpose

This document defines what counts as acceptable research evidence for Jax.

A backtest is not proof by itself. It is evidence that must be validated, challenged, and followed by paper trading.

## Research ladder

```text
IDEA
HYPOTHESIS
BACKTESTED_WEAK
BACKTESTED_PROMISING
PAPER_READY
PAPER_REJECTED
PAPER_PROVEN
```

No setup family can become `PAPER_READY` without a research evidence bundle.

Live trading is not part of the current roadmap.

## Research hypothesis template

```json
{
  "hypothesis_id": "hyp_swing_001",
  "setup_family": "post_earnings_drift_continuation",
  "claim": "Stocks with earnings beat, raised guidance, and volume confirmation outperform over 5-20 trading days.",
  "target_assets": ["large_cap_equities"],
  "holding_period": "5_to_20_trading_days",
  "entry_rule": "Enter after confirmation, not on first gap.",
  "exit_rule": "Exit on target, invalidation, or max hold period.",
  "risk_rule": "Risk/reward must be >= 2:1.",
  "expected_failure_modes": [
    "Overfitting",
    "Survivorship bias",
    "Ignoring slippage",
    "Macro regime change"
  ]
}
```

## Backtest evidence bundle

```json
{
  "hypothesis_id": "hyp_swing_001",
  "setup_family": "post_earnings_drift_continuation",
  "dataset_id": "earnings_daily_2018_2025_v1",
  "dataset_hash": "sha256:...",
  "date_range": {
    "start": "2018-01-01",
    "end": "2025-12-31"
  },
  "instrument_universe": ["US_large_cap", "UK_large_cap"],
  "benchmark": "SPY_or_relevant_index",
  "assumptions": {
    "execution": "next_bar_or_next_day_open",
    "position_sizing": "fixed_fractional",
    "max_risk_per_trade": 0.01
  },
  "slippage_model": {
    "type": "bps",
    "value": 10
  },
  "fees_model": {
    "commission": "broker_model_or_fixed",
    "spread_assumption": "included_or_documented"
  },
  "in_sample_period": {
    "start": "2018-01-01",
    "end": "2022-12-31"
  },
  "out_of_sample_period": {
    "start": "2023-01-01",
    "end": "2025-12-31"
  },
  "performance_metrics": {
    "total_return": 0,
    "annualised_return": 0,
    "sharpe": 0,
    "sortino": 0,
    "max_drawdown": 0,
    "profit_factor": 0,
    "expectancy": 0,
    "win_rate": 0,
    "average_win": 0,
    "average_loss": 0,
    "sample_size": 0
  },
  "failure_modes": [],
  "promotion_decision": "RESEARCH_ONLY"
}
```

## Minimum acceptance rules

A backtest is invalid if it lacks:

- dataset id
- dataset hash
- date range
- instrument universe
- benchmark
- slippage assumption
- cost/fees assumption
- sample size
- drawdown metrics
- failure modes
- out-of-sample validation or explicit reason why unavailable

## Promotion rules

| Evidence | Max Promotion |
|---|---|
| Idea only | IDEA |
| Hypothesis only | HYPOTHESIS |
| Backtest without costs/slippage | BACKTESTED_WEAK |
| Backtest with costs but no OOS | BACKTESTED_WEAK |
| Backtest with OOS and realistic costs | BACKTESTED_PROMISING |
| Backtest promising + risk rules defined | PAPER_READY |
| Paper results poor | PAPER_REJECTED |
| Paper results stable over agreed sample | PAPER_PROVEN |

## Anti-overfitting rules

Jax must flag:

- too many parameters
- small sample size
- no out-of-sample period
- no cost/slippage modelling
- strategy only works in one regime
- unrealistic entry/exit assumptions
- missing failed trades
- survivorship bias
- lookahead bias

## Current roadmap limitation

No strategy can become:

```text
LIVE_READY
```

in the current roadmap.
