# Implementation Task Breakdown (Do This In Order)

## Order of operations (do not reorder)
1) **Make jax-market a UTCP provider**
- Add `/tools` endpoint
- Implement `market.get_quote`, `market.get_candles`, `market.get_earnings`
- Update `config/providers.json` endpoint to `http://jax-market:8095/tools`
- Confirm `jax-api` can call `MarketDataService.GetCandles()` for intraday

2) **Replace fake backtest**
- Implement `libs/backtest`
- Wire UTCP `backtest.run_strategy` to real engine
- Persist runs/trades to DB

3) **Implement Strategy Instance V2**
- Add file configs: `config/strategy-instances/*.json`
- Add DB table `strategy_instances`
- Implement loader that:
  - reads files
  - upserts to DB
  - returns enabled instances

4) **Add instance-aware execution metadata**
- Add `instance_id` to `strategy_signals`
- Add `instance_id` to `trades`
- Ensure `jax-trade-executor` reads and writes instance IDs

5) **Add flat-by-close enforcement**
- Add scheduler in `jax-trade-executor` OR a companion service
- Implement:
  - cancel orders for instance
  - close positions for instance
  - reject execute after flatten time

6) **Create jax-strategy-runner**
- Run multiple instances
- Generate signals during entry windows
- Trigger paper executions (auto-approve in paper mode)
- Log every decision and reason for skips

7) **Implement the 5 strategies**
Implement sequentially:
- Strategy 1 first, then 2, then 3, then 4, then 5
Each strategy must:
- define signals deterministically
- backtest
- paper trade

## Definition of done (per strategy)
- Backtest produces trade list + metrics for >= 2 years.
- Paper trading runs (paper) for >= 4 weeks.
- Flat-by-close verified.
- Kill switches verified.
- All outputs stored and queryable by `instance_id`.

## Hard risks
- IB paper data may be delayed/incomplete. Backtest should rely on stored candles; paper can tolerate delays but must be logged.
- Intraday fill modelling is approximate; make it explicit and consistent.
