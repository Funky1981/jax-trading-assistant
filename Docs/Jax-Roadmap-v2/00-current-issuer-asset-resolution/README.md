# Phase 00 — Current Issuer & Asset Resolution

**Status:** GO / complete (2026-08-21)

The accepted architecture is `Event -> typed causal attribution -> deterministic policy -> DIRECT / PROXY / UNRESOLVED -> deterministic resolver`.

## Purpose
    Finish and prove the current frozen issuer-recognition/asset-resolution work before widening scope.

## Prerequisites
- Latest handover/repo state verified
- Paper-safe invariants preserved

## Reference systems to inspect at this phase
Jax current issuer benchmark/asset-resolution work.

## Work packages
- `00.01` — Fix benchmark/manifest execution gate — complete
- `00.02` — Run frozen v4 issuer benchmark — complete
- `00.03` — Analyse failures and harden deterministic resolution — complete, including the reviewed C1F causal-attribution evolution and accepted Luna/Terra cells
- `00.04` — Freeze Phase 00 evidence baseline — complete as the gate/closure package; no further model or resolver implementation is required

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
Jax can reliably identify the issuer/instrument or return an explicit unknown/ambiguous result on the frozen benchmark; results are reproducible and paper-safe.

See `GATE.md`. Every work package requires independent review before the next one starts.

The compact retained-evidence inventory is `../../evidence/PHASE-00-ISSUER-RESOLUTION-CLOSEOUT.md`. No Phase 00 work remains. Phase 01 has not started.
