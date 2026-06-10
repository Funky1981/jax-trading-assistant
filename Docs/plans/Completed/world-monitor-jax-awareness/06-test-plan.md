# 06 — Test Plan

## Unit Tests

Add tests for:

- Valid research trigger payload.
- Missing timestamp rejection.
- Missing source URLs rejection.
- Low source count rejection.
- Stale event rejection.
- Trade-instruction language rejection.
- Unknown event type handling.
- Non-ETF mapping rejection.
- Duplicate event detection.

## Integration Tests

Add tests for:

- Adapter payload → Jax endpoint.
- Jax endpoint → raw event storage.
- Raw event → normalized event.
- Normalized event → ETF mapping.
- Event → research queue/status.
- Rejected trigger → audit record.

## End-to-End Smoke Test

Scenario:

```text
World Monitor-style macro rates headline arrives
  ↓
Adapter sends research trigger
  ↓
Jax stores event
  ↓
Jax maps QQQ/SPY/TLT
  ↓
Jax marks event as pending research
  ↓
No trade is created automatically
```

Expected result:

- Event stored.
- Status returned.
- Audit trail exists.
- No order exists.
- No execution instruction exists.
- No approved trade exists.

## No-Go Test Cases

The system must reject:

```json
{
  "headline": "Buy QQQ now",
  "reason": "World Monitor says this is a strong trade"
}
```

The system must also reject:

- Missing source URLs.
- Stale event.
- Unknown source with low confidence.
- Leveraged/inverse/volatility ETF mapping.
- Any payload attempting to set runtime mode.

## Manual UAT Checklist

- [ ] World Monitor runs locally.
- [ ] Jax runs locally in research/paper mode.
- [ ] Adapter runs locally.
- [ ] Test payload reaches Jax.
- [ ] Jax stores raw event.
- [ ] Jax stores normalised event.
- [ ] Jax maps ETFs.
- [ ] Jax does not create a trade automatically.
- [ ] Jax can reject bad trigger.
- [ ] Audit record is visible.

## Signoff Rule

This integration is not complete until a full local smoke test proves World Monitor can trigger Jax research without creating any trade/order/execution instruction.

## Separate-System Control Layer Tests

Add tests for the World Monitor → Jax boundary.

Required tests:

1. A valid high-severity World Monitor event enters the Jax research inbox.
2. A low-severity event is stored or ignored but does not trigger Jax research by default.
3. A duplicate headline cluster creates only one research trigger.
4. A single weak Tier 3 source is rejected or held as awareness-only.
5. A Tier 1 official-source event can pass with a lower source count if it is directly relevant.
6. A payload containing order/execution fields is rejected.
7. A payload without timestamp is rejected.
8. A payload without source URLs is rejected unless it is from a trusted direct source connector.
9. A confidence score without confidence reasons is rejected.
10. A candidate cannot be created until Jax evidence checks complete.
11. Mobile notification is sent only after Jax evidence passes, not when World Monitor first sees the headline.
12. Operator feedback is stored and linked back to the original World Monitor event.

