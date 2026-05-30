# 12 — Testing, UAT, and Readiness

## Unit Tests

```text
ETF allowlist
blocked symbols
event classification
ETF mapping
priced-in scoring
confounder lookup
evidence bundle validation
AI output validation
approval expiry
mobile approval token validation
```

## Integration Tests

```text
candle + event -> event window
event -> ETF mapping
event -> priced-in score
event -> evidence bundle
evidence bundle -> candidate
candidate -> approval
approval -> paper execution instruction
invalid guardrail path rejected
```

## Final GO Criteria

```text
migrations applied cleanly
ETF-only defaults verified
live trading disabled
IB paper mode verified
event ingestion works
historical analysis works
priced-in verdict generated
evidence bundle generated
mobile approval works
paper order can be entered/exited
audit trail complete
```
