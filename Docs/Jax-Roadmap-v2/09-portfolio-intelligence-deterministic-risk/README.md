# Phase 09 — Portfolio Intelligence & Deterministic Risk

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
