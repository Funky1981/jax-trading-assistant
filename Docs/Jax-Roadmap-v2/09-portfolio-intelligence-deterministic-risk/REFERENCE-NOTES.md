# Phase 09 Reference Notes — Portfolio & Risk

## Fincept concepts
- portfolio service/metrics and unified portfolio concepts;
- workflow `RiskManager`: max position/value/exposure, order, loss, asset and time restrictions.

## Jax target
Portfolio-aware recommendation evaluation should be deterministic and reason-coded:
- existing exposure,
- sector/theme concentration,
- correlation,
- volatility contribution,
- cash/risk budget,
- scenario stress,
- proposed position size.

A good standalone idea can correctly become NO_TRADE because portfolio context is poor.
