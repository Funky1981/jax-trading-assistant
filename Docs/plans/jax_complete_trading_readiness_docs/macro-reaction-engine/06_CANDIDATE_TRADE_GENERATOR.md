# 06 — Candidate Trade Generator

## Goal

Create a paper-only candidate trade after all evidence gates pass.

This is not execution.

## Candidate trade rules

Phase 1 permits only:

```text
ETF long
ETF avoid-long / short-bias recommendation
paper-only candidate
human approval required
```

If your platform does not support short ETF paper trades safely, represent bearish trades as:

```text
bearish_candidate
no execution instruction
manual review required
```

Blocked:

```text
live orders
options
leveraged ETF workaround
inverse ETF workaround
single-stock proxies
auto-approval
broker write
```

## Candidate fields

```text
id
macro_event_id
evidence_bundle_id
symbol
side
bias
entry_type
entry_reference_price
stop_reference_price
target_reference_price
risk_percent
time_limit
status
created_reason
rejection_reason
created_at
```

## Side values

```text
long
short_bias
watch_only
no_trade
```

## Entry type values

```text
breakout_continuation
pullback_retest
range_reclaim
no_entry
```

## Example candidate

```json
{
  "symbol": "QQQ",
  "side": "short_bias",
  "entry_type": "pullback_retest",
  "entry_reference_price": 430.25,
  "stop_reference_price": 435.10,
  "target_reference_price": 421.00,
  "risk_percent": 0.5,
  "time_limit": "end_of_session",
  "status": "awaiting_human_approval"
}
```

## Risk rules

Default:

```text
max risk per macro-event candidate: 0.5%
max open macro candidates: 1
max same-theme exposure: 1
no trade if stop cannot be defined
no trade if reward:risk < 1.5
no trade after max chase threshold
```

## Codex task

```text
Add candidate trade generation from evidence bundles.

Only create a candidate when evidence verdict is candidate_allowed.

Candidate must:
- be paper-only
- require human approval
- use allowlisted ETF symbol
- include entry/stop/target
- include risk percent
- include walk-away reasons
- never create broker order
```

## Tests

```text
candidate_allowed bundle creates awaiting_human_approval candidate
candidate_blocked bundle creates no candidate
watch_only bundle creates watchlist record not trade candidate
missing stop blocks candidate
risk above limit rejected
broker order table remains untouched
```

## Acceptance criteria

```text
candidate created only after evidence gates
candidate status is awaiting_human_approval
no broker order created
risk limits enforced
audit trail links event → evidence → candidate
```
