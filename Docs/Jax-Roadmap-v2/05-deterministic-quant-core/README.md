# Phase 05 — Deterministic Quant Core

## Purpose
    Provide reproducible numerical context for research and risk using established numerical libraries instead of LLM arithmetic.

## Prerequisites
- Phase 03 GO
- Canonical market observations proven

## Reference systems to inspect at this phase
Fincept analytics catalogue plus established Python numerical/portfolio libraries.

## Work packages
- `05.01` — Quant service/library boundary and versioned request/response contract
- `05.02` — Returns/log returns and benchmark-relative performance
- `05.03` — Volatility/ATR/drawdown
- `05.04` — Correlation/beta/covariance
- `05.05` — Liquidity/volume anomaly metrics
- `05.06` — Basic risk-adjusted metrics
- `05.07` — Position sizing primitives
- `05.08` — Portfolio exposure primitives
- `05.09` — Library evaluation: NumPy/SciPy/statsmodels/skfolio/Riskfolio where justified

## Out of scope
- Work belonging to later phases.
- Unverified Fincept features merely because they exist in documentation.
- Direct copying/porting of Fincept implementation source.
- Live/execution authority unless this phase explicitly says otherwise.

## Exit gate
Given a frozen canonical dataset, every core quant result is deterministic, versioned, tested against known values and independently reproducible.

See `GATE.md`. Every work package requires independent review before the next one starts.
