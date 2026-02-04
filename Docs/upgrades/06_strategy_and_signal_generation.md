# Strategy and Signal Generation

The current trade generation logic is simplistic: it looks for gap events and applies a static target rule. To operate in production, you need a flexible strategy engine that supports multiple strategies, configurable parameters, and signal quality checks.

## Why it matters

Rigid strategy logic limits the system’s ability to adapt to changing market conditions. A robust signal generation layer allows for rapid experimentation, diversification across strategies, and better risk‑adjusted returns.

## Tasks

1. **Expand the `StrategyConfig` model**
   - Include fields for entry rules, stop rules, target rules, timeframe (intraday vs. daily), instrument filters, and risk parameters.
   - Allow strategies to specify which event types they respond to (e.g. gaps, breakouts, momentum shifts).

2. **Introduce a strategy registry**
   - Maintain a registry of available strategies, loaded from configuration or a database. Enable runtime enabling/disabling of strategies without redeployment.
   - Provide an API endpoint to list strategies and their parameters (`GET /strategies`) and update settings (`PATCH /strategies/{id}`).

3. **Generalise event‑to‑signal mapping**
   - Implement modular rule functions that take an event and return one or more trade setups. For example, a breakout strategy might look for price closing above a moving average, whereas a mean‑reversion strategy might look for oversold conditions.
   - Support multi‑leg targets (e.g. scale out at 1R, 2R, 3R) and trailing stops.

4. **Quality filters**
   - Add checks to filter out low‑quality signals: insufficient data, abnormal spreads, high volatility, or overlapping signals on the same symbol.
   - Log skipped signals and reasons via the audit logger.

5. **Backtesting hooks**
   - Ensure strategy code can be reused in a backtesting module (see `jax-utcp-smoke` or planned `jax-backtest`). Functions should be deterministic and not rely on network calls.

6. **Documentation and examples**
   - Document each strategy type with examples, parameter descriptions, and expected behaviour.
   - Provide sample configuration files for enabling strategies and tuning parameters.
