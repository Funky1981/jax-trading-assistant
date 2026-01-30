# Strategy Index

Generated: 2026-01-30

## Employ (baseline / approved templates)
1. `strategies/known/trend_following_time_series_momentum.md`
2. `strategies/known/momentum_cross_sectional.md`
3. `strategies/known/breakout_volatility_squeeze.md`
4. `strategies/known/mean_reversion_vwap_bands.md`
5. `strategies/known/pairs_trading_cointegration.md`
6. `strategies/known/volatility_targeting.md`
7. `strategies/known/carry_trade_risk_aware.md` (risk-aware only)
8. `strategies/known/risk_parity_multi_asset.md`
9. `strategies/known/stop_loss_and_drawdown_control.md` (overlay, not edge)
10. `strategies/known/tail_risk_hedging_overlay.md` (overlay, advanced)
11. `strategies/known/market_making_inventory_aware.md` (advanced; venue dependent)

## Avoid (anti-patterns / guardrails)
1. `anti-patterns/martingale_and_loss_doubling.md`
2. `anti-patterns/curve_fitting_parameter_sweeps.md`
3. `anti-patterns/ignoring_transaction_costs.md`
4. `anti-patterns/latency_illusion_backtests.md`
5. `anti-patterns/unhedged_short_volatility.md`
6. `anti-patterns/no_regime_awareness.md`
7. `anti-patterns/lookahead_and_data_leakage.md`
8. `anti-patterns/survivorship_bias_universe.md`
9. `anti-patterns/strategy_sprawl_no_governance.md`

## Patterns (reusable features)
- `patterns/volatility_spike.md`
- `patterns/orderflow_imbalance.md`
- `patterns/liquidity_vacuum.md`
- `patterns/mean_reversion_zscore.md`

## Next upgrades (recommended)
- Add venue-specific execution models (fees, maker/taker, partial fills, rejections)
- Add monitoring dashboards / alerts (slippage, fill rate, DD, regime drift)
- Add “candidate mining” playbooks for Jax discovery loop
