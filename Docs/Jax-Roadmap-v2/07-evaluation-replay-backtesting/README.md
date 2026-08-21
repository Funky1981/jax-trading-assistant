# Phase 07 — Evaluation, Replay & Backtesting

## Purpose
    Prove whether Jax recommendations and strategies have value and remain reproducible before granting them more authority.

## Prerequisites
- Phase 06 GO

## Reference systems to inspect at this phase
Fincept Alpha Arena replay, Fincept backtesting provider abstraction, LEAN, VectorBT and custom event replay.

## Work packages
- `07.01` — Historical event/research replay engine
- `07.02` — Frozen benchmark registry
- `07.03` — Model/prompt/algorithm version comparison
- `07.04` — Recommendation outcome tracking
- `07.05` — Traditional strategy backtesting adapter evaluation
- `07.06` — Transaction cost/slippage assumptions
- `07.07` — Walk-forward/out-of-sample protocol
- `07.08` — Operational replay tooling

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
A frozen historical case can reproduce the research context and recommendation, outcomes are tracked without leakage, and candidate logic has explicit out-of-sample evaluation evidence.

See `GATE.md`. Every work package requires independent review before the next one starts.
