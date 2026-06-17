# 00 — Implementation Order

## Existing plan dependency

Start only after the existing World Monitor/Jax awareness boundary is either complete or has a working local smoke test.

Existing plan should already cover:

```text
World Monitor = radar
Adapter = bridge
Jax = research/evidence engine
Human = approval
Broker = execution only after approval
```

This pack starts after Jax has accepted a valid research trigger.

## Implementation phases

```text
Phase 1 — Macro Event Model and Calendar Data
Phase 2 — Candle/Chart Reaction Engine
Phase 3 — ETF Mapping and Scenario Playbooks
Phase 4 — Priced-In and Confounder Checks
Phase 5 — Evidence Bundle Builder
Phase 6 — Candidate Trade Generator
Phase 7 — UI/API Integration
Phase 8 — Backtesting and UAT
```

## Do not skip phases

The dangerous mistake is jumping from "headline received" to "trade suggested".

The correct order is:

```text
headline
actual data
market reaction
ETF mapping
historical context
priced-in score
confounders
risk
evidence
candidate
approval
```

## Branch strategy

Recommended:

```text
branch: feature/macro-reaction-engine
```

## Global acceptance criteria

The work is complete only when:

```text
1. A jobs/Fed/CPI event can be ingested.
2. Jax can map it to allowed ETFs.
3. Jax can fetch pre/post event candles.
4. Jax can decide whether the chart confirms the event.
5. Jax can reject late/chased/noisy moves.
6. Jax can produce an evidence bundle.
7. Jax can create a paper-only candidate trade.
8. No live order can be created without human approval.
9. Tests prove rejection paths.
10. Full local UAT uses a fake macro event and no broker write occurs.
```
