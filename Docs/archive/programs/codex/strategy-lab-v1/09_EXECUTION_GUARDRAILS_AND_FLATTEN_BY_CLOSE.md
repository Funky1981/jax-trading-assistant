# Execution Guardrails and Flatten-by-Close

## Objective
Ensure the system cannot accidentally carry risk overnight.

## Additions to jax-trade-executor (recommended location)

### 1) Flatten scheduler
Implement a per-instance scheduled process:
- input: `instance_id`, `flatten_time`, timezone
- at flatten time:
  - cancel open orders for the instance
  - close open positions for the instance

If IB bridge lacks “cancel all by tag”, implement:
- store order IDs per instance in DB
- cancel by those IDs

### 2) Instance-aware risk gates
Existing gates are global-ish. Add:
- max open positions per instance
- max daily loss per instance
- max trades per day per instance

### 3) Hard cut-offs
- reject any execute request after flatten time (instance timezone)
- reject any new signals after flatten time

### 4) Paper vs live safety
- require explicit env var `ALLOW_LIVE_TRADING=true` for live.
- default to paper ports and paper account.

## Acceptance criteria
- At flatten time, all positions are closed and orders cancelled.
- No trade can open after flatten time.
- Breaching kill switch halts the instance for the rest of day.
