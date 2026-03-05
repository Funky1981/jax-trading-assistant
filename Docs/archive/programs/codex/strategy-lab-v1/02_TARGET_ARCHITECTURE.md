# Target Architecture (Minimal and Sane)

## Data flow
1. `jax-market` provides UTCP tools:
   - `market.get_quote`
   - `market.get_candles` (supports intraday timeframes)
   - `market.get_earnings` (optional but required for earnings strategy)
2. Backtesting:
   - A real deterministic replay engine is called by UTCP tool `backtest.run_strategy`.
3. Paper trading:
   - Strategy runner generates signals tagged with `strategy_instance_id`.
   - Approval triggers `jax-trade-executor` to execute via IB paper account.
4. Guardrails:
   - Flat-by-close enforcement automatically closes/cancels at configured time.
   - Instance-level kill switches halt generation/execution for the rest of day.

## Separation of concerns
- `jax-market`: market data retrieval + cache/storage + UTCP tool server.
- `libs/backtest`: deterministic replay + fills + metrics.
- `jax-api`: orchestration + endpoints + persistence for runs/signals/trades.
- `jax-trade-executor`: execution + risk gates + flatten-by-close.
- `jax-strategy-runner` (new): schedules multiple strategy instances and manages research/paper loops.

## Key principle
Backtest and paper/live must share the **same decision logic** and **same config**; only the data source differs.
