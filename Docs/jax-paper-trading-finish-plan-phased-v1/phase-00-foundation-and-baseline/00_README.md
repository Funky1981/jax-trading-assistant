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
- strategy-instance CRUD is already present
- Research / Analysis / Testing / Approvals / Assistant pages already exist
- candidate / approval / chat tables already exist
- AI decision and acceptance tables already exist
- trust-gate endpoints and reports already exist

Still missing or incomplete:
- strategy-instance config drift between legacy `universe` and canonical `symbols`
- candidate provenance linkage and blocked-reason persistence
- approval-driven paper execution worker that consumes `execution_instructions`
- paper-mode bypass closure for direct `/api/v1/execute`
- assistant queue/search/knowledge tools and assistant audit coverage
- trust-gate hardening for approval -> instruction -> trade -> fill linkage
- stronger flatten-proof, provenance, and paper-readiness evidence

## Non-negotiables
- no fake/synthetic data in research truth path or paper-trading decisions
- AI is advisory only
- no order submission without explicit approval
- no gate pass, no paper-trading sign-off

## Rebaseline
Treat this pack as a rebaseline-and-hardening program, not a greenfield implementation checklist.

Execution now starts with:
- truth-path contract hardening
- approval-driven paper execution
- watcher lifecycle completion
- assistant and AI-audit completion
- trust-gate proof hardening

Keep ADR-0012 runtime boundaries intact:
- `cmd/trader` remains deterministic execution and paper runtime
- `cmd/research` remains orchestration and research runtime
