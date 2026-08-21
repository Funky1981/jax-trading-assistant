# Phase 11 Reference Notes — Paper Trading

## Fincept concepts
- `IBroker` unified vocabulary/capabilities.
- PaperTrading / OrderMatcher separation.
- Alpha Arena paper venue's explicit fees, funding/slippage/latency assumptions and crash recovery.

## Jax target
Paper trading is an evaluation environment, not a cosmetic preview. It must model the execution effects relevant to Jax's intended instruments and holding periods.

Do not inherit Fincept's leverage/perpetual assumptions. Jax's paper model should be calibrated to its own broker/asset universe.
