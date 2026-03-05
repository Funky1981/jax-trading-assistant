# Strategy Instances and Runtime

## Objective
Run multiple strategies concurrently with isolation and guardrails.

## New service: jax-strategy-runner
Create `services/jax-strategy-runner/`:
- Loads enabled instances (DB + config files)
- Schedules:
  - research runs (backtest projects)
  - paper trading signal generation
  - end-of-day flatten checks (if not in executor)

## Isolation rules
Each instance MUST have:
- independent daily counters (trades, P/L)
- independent kill-switch state
- independent run IDs and signal IDs

## Paper trading loop (v1)
1. At entry window, compute signals for configured symbols.
2. Store signals with:
   - `instance_id`
   - `strategy_id`
   - timestamps
   - entry/stop/target
   - confidence (if applicable)
   - status `proposed`
3. For paper mode, optionally auto-approve.
4. Call trade executor `/api/v1/execute`.
5. Store outcomes and metrics.

## Kill switches (per instance)
- breach of max daily loss => stop for day
- max consecutive losses => stop for day
- after flatten time => stop; close positions

## Acceptance criteria
- You can run 5 instances concurrently.
- One instance failing does not stop others.
- Each instance can be enabled/disabled without redeploying code (DB flag).
