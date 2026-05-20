# 13 — Testing, UAT, and Readiness

## Goal

Prove ETF news research and paper trading are safe before use.

## Unit Tests

Add tests for:

- ETF allowlist
- blocked symbols
- event classification
- ETF mapping
- priced-in scoring
- confounder lookup
- evidence bundle validation
- AI output validation
- approval expiry
- mobile approval token validation

## Integration Tests

Add tests for:

- candle + event → event window
- event → ETF mapping
- event → priced-in score
- event → evidence bundle
- evidence bundle → candidate
- candidate → approval
- approval → paper execution instruction
- reject invalid guardrail path

## Live Paper UAT

Run on local paper stack:

```text
SPY quote fetch
QQQ quote fetch
SPY candle fetch
QQQ candle fetch
event ingestion
event study generation
candidate creation
approval
paper order
position readback
exit/flatten
memory retain
reflection
```

## Readiness Gates

GO only if:

- migrations applied cleanly
- ETF-only defaults verified
- live trading disabled
- IB paper mode verified
- event ingestion works
- historical analysis works
- priced-in verdict generated
- evidence bundle generated
- mobile approval works
- paper order can be entered/exited
- audit trail complete

## NO-GO Conditions

- non-ETF candidate generated
- live trading enabled
- missing stop-loss
- missing approval
- stale quote accepted
- priced-in rejection ignored
- AI can bypass guardrails
- migrations not applied
- provider health unknown
