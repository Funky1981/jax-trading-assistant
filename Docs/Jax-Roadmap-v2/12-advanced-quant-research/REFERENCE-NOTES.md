# Phase 12 Reference Notes — Advanced Quant

## Fincept concepts
AI Quant Lab wraps Qlib and RDAgent for feature engineering, factor research, backtesting, portfolio optimization, drift/retraining and automated hypotheses.

## External candidate
Microsoft Qlib covers data processing, model training, backtesting, alpha research, risk, portfolio optimization and online/adaptive workflows. It is a research platform candidate, not something to insert into Jax core prematurely.

## Jax gate
No factor/ML model is promoted because its backtest looks good. Require:
- explicit hypothesis,
- point-in-time/leakage-safe data,
- baseline,
- train/validation/test or walk-forward protocol,
- multiple-regime stability,
- transaction-cost sensitivity,
- experiment registry,
- drift monitoring,
- promotion/rejection rule.
