# Phase 11 — High-Fidelity Paper Trading

## Purpose
    Simulate execution realistically enough to judge the end-to-end system before considering live trading.

## Prerequisites
- Phase 10 GO

## Reference systems to inspect at this phase
Fincept broker/paper venue concepts and independent broker/provider documentation.

## Work packages
- `11.01` — Broker adapter capability contract
- `11.02` — Paper venue/order matcher
- `11.03` — Fees/spread/slippage/latency model
- `11.04` — Partial-fill and market-hours semantics where relevant
- `11.05` — Position/account ledger
- `11.06` — Broker/public-market reconciliation
- `11.07` — Paper outcome attribution
- `11.08` — Long-duration paper soak protocol

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
    Approved paper intents produce realistic auditable orders/fills/positions, reconcile correctly, and can run for an agreed sustained evaluation period without live execution being enabled.

See `GATE.md`. Every work package requires independent review before the next one starts.
