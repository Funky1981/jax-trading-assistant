# Phase 09 — Portfolio Intelligence & Deterministic Risk

**Status:** **EXIT DEMONSTRATED / EXTERNAL REVIEW PENDING**

## Purpose
    Evaluate recommendations in the context of the user's actual portfolio and explicit risk policy.

## Prerequisites
- Phase 07 GO
- Phase 05 portfolio primitives GO

## Reference systems to inspect at this phase
Fincept portfolio analytics and deterministic workflow risk manager.

## Work packages
- `09.01` — Canonical portfolio/position/account state
- `09.02` — Exposure/concentration/correlation analytics
- `09.03` — Risk budget policy
- `09.04` — Deterministic recommendation risk checks
- `09.05` — Position proposal calculation
- `09.06` — Stress/scenario analysis
- `09.07` — Risk reason codes and audit

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
A recommendation can be accepted, amended or rejected deterministically based on portfolio/risk state, with reproducible reason codes and no execution side effect.

See `GATE.md`. Every work package requires independent review before the next one starts.

## Internal phase evidence

The reproducible phase gate is `internal/modules/portfoliorisk/phase09_exit_test.go`.
It demonstrates canonical synthetic portfolio identity, deterministic exposure,
versioned policy, ACCEPT/AMEND/REJECT, descriptive position proposal, frozen
stress analysis, reason-code audit reconstruction, stale/unknown fail-closed
behaviour and no execution side effect. Phase 10 is not started.
