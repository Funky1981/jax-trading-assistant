# Phase 13 — Optional Live Execution

## Purpose
    If sustained paper evidence justifies it and the user explicitly chooses to proceed, add the smallest safe live-execution boundary.

## Prerequisites
- Phase 11 sustained paper evidence GO
- Explicit user decision
- Separate live risk review

## Reference systems to inspect at this phase
Fincept broker abstraction/Alpha Arena live gates as references only; Jax retains separate explicit live safety design.

## Work packages
- `13.01` — Live broker adapter selection
- `13.02` — Credential/session security
- `13.03` — Live-only risk limits
- `13.04` — Typed explicit acknowledgement/approval
- `13.05` — Order idempotency and reconciliation
- `13.06` — Emergency kill/disable paths
- `13.07` — Live observability/runbook
- `13.08` — Small-capital staged rollout

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
    Live execution is technically and operationally isolated, reconciled and killable; AI still cannot bypass deterministic risk and explicit approval policy.

See `GATE.md`. Every work package requires independent review before the next one starts.
