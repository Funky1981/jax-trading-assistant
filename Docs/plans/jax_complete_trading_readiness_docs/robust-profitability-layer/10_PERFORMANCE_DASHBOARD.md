# 10 — Performance Dashboard

## Goal

Show whether Jax is improving or fooling itself.

The dashboard must focus on expectancy and risk, not exciting screenshots.

## Metrics

```text
number of events analysed
number of candidates created
candidate rate
approval rate
win rate
average R
median R
expectancy
max drawdown
false positive rate
false rejection rate
walk-away correctness
strategy performance
event-type performance
ETF performance
regime performance
score bucket performance
```

## Key dashboard sections

### Event Funnel

```text
events detected
events validated
events researched
events rejected
watch-only
candidates created
approved
paper trades completed
```

### Strategy Performance

```text
strategy
trades
win rate
avg R
expectancy
max drawdown
best regime
worst regime
```

### Score Calibration

```text
candidate score bucket
number of trades
avg R
win rate
```

This proves whether high scores actually perform better.

### Walk-Away Review

Track whether Jax was right to say no.

```text
walk-away reason
later outcome
was rejection correct?
```

## Data model

Can use views first:

```sql
CREATE VIEW strategy_performance_summary AS
SELECT
  strategy_key,
  COUNT(*) AS trades,
  AVG(final_r) AS avg_r
FROM trade_reviews
GROUP BY strategy_key;
```

## Codex task

```text
Create performance dashboard data layer and simple UI/API.

Start with read-only metrics.
No trading actions in dashboard.
```

## Tests

```text
empty dashboard handles no data
strategy summary calculates avg R
event funnel counts statuses correctly
score bucket summary groups candidates
```

## Acceptance criteria

```text
operator can see if Jax has edge
metrics are read-only
dashboard includes rejection quality
