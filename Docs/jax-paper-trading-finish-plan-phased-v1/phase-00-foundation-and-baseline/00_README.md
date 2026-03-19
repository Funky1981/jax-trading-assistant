# Jax Paper Trading / Backtesting Finish Plan v1

This pack is a **Codex-ready completion plan** to take the current `work` branch of `jax-trading-assistant` to **fully ready for deterministic backtesting and controlled paper trading**.

## Goal
Finish the remaining work needed so Jax can:
- run real backtests on real data
- support configurable strategy instances
- continuously scan for setups
- create candidate trades
- require human approval before execution
- paper trade through a controlled execution path
- explain decisions and blockers
- provide Research / Analysis / Testing operator pages
- pass trust gates before any real money is used

## Source of truth
This plan is based on the reviewed `work` branch state, including:
- `cmd/trader`
- `cmd/research`
- `cmd/shadow-validator`
- `cmd/trader/frontend_api.go`
- `cmd/research/main.go`
- `cmd/trader/strategy_instances_loader.go`
- `internal/modules/backtest/engine.go`
- `internal/modules/execution/engine.go`
- `cmd/trader/handlers_artifacts.go`
- `frontend/src/app/App.tsx`
- `frontend/src/components/layout/AppShell.tsx`

## Current reality
Strong base exists:
- runtime split
- artifact system
- backtest module
- execution module
- strategy instance loading
- orchestration and research runtime
- auth-enabled frontend shell

Still missing or incomplete:
- no-fake-data hard enforcement
- event data foundation completion
- strategy type / instance management UX
- always-on watcher and candidate trades
- approval queue and instruction model
- Research / Analysis / Testing pages
- chat assistant
- AI audit + replay model
- trust gates automation
- paper-trading readiness proof

## Non-negotiables
- no fake/synthetic data in research truth path or paper-trading decisions
- AI is advisory only
- no order submission without explicit approval
- no gate pass, no paper-trading sign-off
