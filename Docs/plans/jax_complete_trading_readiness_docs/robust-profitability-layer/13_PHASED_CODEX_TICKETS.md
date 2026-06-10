# 13 — Phased Codex Tickets

## Ticket 01 — Market Regime Engine

```text
Add market_regime_snapshots and deterministic regime classification.

Acceptance:
- classifies risk_on/risk_off/rates_dominant/high_volatility/unclear
- missing inputs explicit
- regime included in evidence
- unclear/conflicting regime can veto candidate
```

## Ticket 02 — Cross-Asset Confirmation

```text
Add cross_asset_confirmations.

Acceptance:
- confirms expected ETF basket movement
- detects conflicts
- missing assets explicit
- conflicted verdict blocks candidate
```

## Ticket 03 — Economic Calendar Integration

```text
Add economic_calendar_events and matching to macro events.

Acceptance:
- stores actual/forecast/previous
- calculates surprise
- links calendar event to macro event
- invalid/stale data quarantined
```

## Ticket 04 — Confounder Engine

```text
Add confounder_events and event_confounder_links.

Acceptance:
- detects same-time events
- classifies impact
- blocking confounders prevent candidates
- confounders visible in evidence
```

## Ticket 05 — Execution Quality

```text
Add execution_quality_snapshots and service.

Acceptance:
- wide spread blocks
- stale data blocks
- broker unavailable blocks
- event no-trade delay enforced
```

## Ticket 06 — Position Sizing and Portfolio Risk

```text
Add position_size_recommendations.

Acceptance:
- calculates size from risk and stop
- no stop blocks
- daily/weekly limits enforced
- same-theme exposure enforced
```

## Ticket 07 — Strategy Playbook Engine

```text
Add named strategy playbooks and results.

Acceptance:
- candidates require strategy match
- failed rules visible
- no strategy match blocks candidate
```

## Ticket 08 — Walk-Away Engine

```text
Add walkaway_decisions.

Acceptance:
- blocker reasons prevent candidates
- no-trade summaries generated
- UI/API can show why Jax walked away
```

## Ticket 09 — Post-Trade Review

```text
Add trade_reviews and case-study update.

Acceptance:
- review stores R result, MFE, MAE
- operator feedback stored
- original decision immutable
```

## Ticket 10 — Performance Dashboard

```text
Add read-only performance metrics.

Acceptance:
- event funnel
- strategy performance
- score calibration
- walk-away correctness
```

## Ticket 11 — Monte Carlo and Stress Testing

```text
Add risk_simulation_runs.

Acceptance:
- insufficient sample warning
- drawdown/loss streak simulation
- risk warnings produced
- no strategy promotion without simulation
```
