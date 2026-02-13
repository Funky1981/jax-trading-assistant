---
name: jax-golden-replay-regression
description: Deterministic regression workflow for `tests/golden/*` and `tests/replay/*` in jax-trading-assistant. Use when changing signal generation, orchestration behavior, trade execution logic, or any code that can alter externally observed decision outputs.
---

# Jax Golden Replay Regression

Protect behavior parity while allowing intentional evolution.

## Workflow

1. Determine whether the change should preserve behavior or intentionally change behavior.
2. Run deterministic checks:
   - `scripts/golden-check.ps1 -Mode verify` for normal regression validation
   - `scripts/golden-check.ps1 -Mode capture` only when intentionally updating baseline data
3. Inspect differences before accepting any baseline refresh.
4. Record why each changed snapshot is expected.

## Comparison Policy

- Treat snapshot deltas as failures until explained.
- Never refresh golden files to silence flaky or unknown changes.
- Require deterministic inputs: fixed clocks, seeds, and stable ordering.

## Escalation Triggers

- Any risk, sizing, or order-shape difference in execution snapshots.
- Any orchestration response shape changes consumed by API or frontend.
- Any non-deterministic failures across repeated runs.

Use `references/change-policy.md` for accept/reject criteria.
