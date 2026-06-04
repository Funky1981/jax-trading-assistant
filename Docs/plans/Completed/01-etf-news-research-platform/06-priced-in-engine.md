# 06 — Priced-In Engine

## Goal

Detect whether the ETF move already happened before Jax acts.

## Inputs

```text
event timestamp
ETF candles
benchmark ETF
volume
spread
confounders
```

## Calculations

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

## Hard Rules

```text
priced_in = no trade
unclear = no trade
```

## Acceptance Criteria

- Every candidate has verdict.
- Verdict reason stored.
- AI cannot override hard rejection.
- UI can show simple explanation.
