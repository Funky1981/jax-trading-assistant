# Strategy Lab V1 (Copilot/Codex Pack)

## Goal
Implement a **credible research + paper trading lab** for **5 same-day (flat by close) strategies**, running as **multiple isolated Jax strategy instances**, using **intraday candles**, a **real backtest engine**, and **execution guardrails**.

## Non-negotiables
- Replace the **fake backtest** (`libs/utcp/backtest_local_tools.go`) with a real engine.
- Provide **intraday candle access** through UTCP tools (jax-market must expose `/tools`).
- Enforce **flat-by-close** automatically.
- Strategy definitions must be **executable config**, not prose rules.

## Scope boundaries
- Focus: **equities/ETFs via IB**.
- Not attempting HFT/headline-speed trading.
- First iteration uses **simple fill models**; refine later.

## Deliverables
1. `jax-market` exposes UTCP tools: quotes/candles/earnings (minimum: candles for `1m` + `1d`).
2. New backtest engine (deterministic replay) hooked into UTCP `backtest.run_strategy`.
3. Strategy config V2 schema + instance isolation (both **files and DB**).
4. Five same-day strategies implemented + backtested + paper-tested harness.
5. DB changes to store runs/trades/signals per instance and persist research runs.

## How to use this pack
- Put these files into your repo at: `Docs/codex/strategy-lab-v1/`
- Feed **11_IMPLEMENTATION_TASK_BREAKDOWN.md** to Copilot/Codex first.
- Then feed the remaining docs in order.
