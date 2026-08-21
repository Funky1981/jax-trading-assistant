# Phase 05 Reference Notes — Quant Core

## Fincept concepts
Its analytics catalogue covers portfolio analytics, risk, valuation, derivatives, economics and quantitative methods, with wrappers around mature Python libraries.

## Candidate libraries
- NumPy/SciPy: numerical foundation.
- pandas/Polars: tabular/time-series transformations depending measured needs.
- statsmodels: statistical estimation/tests.
- skfolio: portfolio metrics/optimization with scikit-learn-style model selection and validation.
- Riskfolio-Lib: advanced portfolio/risk optimization.
- FinancePy/QuantLib only if fixed-income/derivatives requirements emerge.

## Jax target
The first quant core is deliberately boring: returns, volatility, ATR, drawdown, beta/correlation/covariance, volume/liquidity anomalies, risk-adjusted metrics and sizing primitives.

Do not introduce ML, RL, factor mining or portfolio optimizers before basic deterministic metrics are frozen and tested.
