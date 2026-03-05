# Strategy Types Pack v1 (preload multiple strategies in code)

Goal: implement a set of **strategy types in code** up front, so you can later choose which to test via UI-created **instances** (config only).

Key distinction:
- **Strategy Type (code)**: defines logic + required parameters + validation.
- **Strategy Instance (config)**: selects a type and supplies parameters, symbols, windows, and risk limits. No code change per instance.

Repo facts:
- There is already a registry-style pattern in `services/jax-signal-generator` using `libs/strategies` (RSI/MACD/MA).
- `jax-api` already loads JSON strategy configs and exposes endpoints; but those configs are currently descriptive.
- Same-day strategies require intraday candles and time window logic; this pack defines the code structure to support them.

Deliverables:
1) A new strategy interface suitable for same-day strategies (intraday, flat-by-close).
2) Five strategy types implemented and registered.
3) A metadata endpoint to let the UI list available strategy types and their parameter schema.
4) A validation layer so instances can be rejected early.

This pack is Copilot/Codex-ready and describes exact files to add/modify.
