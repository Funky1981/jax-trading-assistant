# 11 — Monte Carlo and Stress Testing

## Goal

Understand risk of ruin before live trading.

A strategy can look profitable but still have unacceptable drawdown.

## Inputs

```text
historical paper trade results
R multiples
win/loss distribution
strategy type
event type
regime
ETF
```

## Monte Carlo outputs

```text
expected equity curve range
max drawdown distribution
loss streak distribution
risk of ruin
probability of ending negative
best/worst percentile outcomes
```

## Stress tests

```text
5 losses in a row
10 losses in a row
slippage doubled
win rate drops 20%
average loss increases 25%
major gap through stop
data outage during event
broker unavailable
```

## Data model

### risk_simulation_runs

```sql
CREATE TABLE risk_simulation_runs (
    id UUID PRIMARY KEY,
    strategy_key TEXT NULL,
    input_summary JSONB NOT NULL,
    simulation_count INT NOT NULL,
    result JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Hard rules from simulation

```text
risk_of_ruin too high = strategy disabled
max drawdown too high = reduce risk
loss streak tolerance exceeded = reduce risk/stop strategy
insufficient sample size = paper-only
```

## Codex task

```text
Create Monte Carlo and stress testing module.

Use stored trade_reviews as input.
Produce risk simulation output.
Do not connect to broker.
```

## Tests

```text
empty trade history returns insufficient_sample
known R series produces expected stats
high drawdown triggers warning
stress scenario doubles slippage
```

## Acceptance criteria

```text
strategy cannot be promoted without simulation
simulation results persisted
risk warnings visible
