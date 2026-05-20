# 07 — Priced-In Engine

## Goal

Determine whether the ETF has already moved enough that Jax should avoid chasing the news.

## First Version

Use deterministic rules.

Do not use ML yet.

## Required Inputs

- event timestamp
- selected ETF
- candles around event
- benchmark ETF
- volume data
- bid/ask spread if available
- related/confounding events

## Calculations

For each event/symbol:

```text
pre_event_1h_return
pre_event_4h_return
pre_event_1d_return
post_event_5m_return
post_event_15m_return
post_event_1h_return
benchmark_return
abnormal_return
volume_spike
spread_quality
volatility_adjusted_move
```

## Verdicts

```text
not_priced_in
partially_priced_in
priced_in
overreaction
unclear
```

## Example Rules

### Priced In

```text
IF pre_event_4h_return already exceeds normal event reaction threshold
AND post_event_15m_return is weak
THEN verdict = priced_in
```

### Overreaction

```text
IF post_event_15m_return is extreme
AND spread widened
AND price reverses sharply
THEN verdict = overreaction
```

### Not Priced In

```text
IF pre_event drift is small
AND post_event confirmation is strong
AND no confounders dominate
THEN verdict = not_priced_in
```

### Unclear

```text
IF conflicting events exist
OR price reaction is mixed
OR data quality is poor
THEN verdict = unclear
```

## Hard Walk-Away Rules

Jax must reject trade candidates if:

- verdict is `priced_in`
- verdict is `unclear`
- event is too old
- spread is abnormal
- stop-loss is too wide
- confounder score is too high
- price already reached target-like move

## Acceptance Criteria

- Every candidate has a priced-in verdict.
- Hard rejection cannot be overridden by AI.
- Verdict reason is stored.
- Mobile approval summary includes priced-in view.
