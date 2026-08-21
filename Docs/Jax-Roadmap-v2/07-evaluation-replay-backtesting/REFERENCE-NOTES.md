# Phase 07 Reference Notes — Evaluation & Backtesting

## Fincept concepts
- Alpha Arena: deterministic context builder, append-only event trail and reproducible decision reconstruction.
- Backtesting provider process: provider abstraction rather than one hard-coded engine.

## External engines
- LEAN: strongest candidate for realistic event-driven strategy/execution simulation.
- VectorBT: strong for fast vectorized research and parameter sweeps.
- Custom Jax event replay: required because market-strategy backtesters do not reproduce news/evidence/research context.

## Jax target
Separate three evaluation problems:
1. **event/research replay** — could Jax have known this then?
2. **recommendation outcome evaluation** — did the thesis/invalidation behave as expected?
3. **trading/backtest simulation** — what would realistic entry/exit/cost mechanics have produced?

Never let current revised macro/fundamental data leak into historical decision reconstruction.
