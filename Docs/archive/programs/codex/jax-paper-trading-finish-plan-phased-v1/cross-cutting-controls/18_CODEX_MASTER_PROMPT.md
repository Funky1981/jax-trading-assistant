# Codex Master Prompt — Finish Jax for Backtesting and Paper Trading

You are continuing work on the `work` branch of `jax-trading-assistant`.

Goal:
Finish the platform so it is fully ready for deterministic backtesting and controlled paper trading, based on the existing architecture already present in `cmd/trader`, `cmd/research`, artifacts, strategy instance loading, backtest module, and execution module.

You must continue from the current codebase. Do not redesign the whole system. Extend what is already there.

Non-negotiable rules:
- no fake/synthetic data in research or paper-trading truth paths
- AI is advisory only
- no chat or AI path can execute trades directly
- no candidate can execute without explicit approval
- no paper-trading sign-off without trust gates passing

Implement in this strict order:
1. truth path hardening (mode policy, provenance, replay)
2. strategy/data model completion
3. always-on watcher and candidate trades
4. approval queue and paper execution instruction flow
5. frontend operator pages
6. assistant and AI audit
7. trust gates and paper-readiness sign-off

For every step:
- add or modify migrations
- add backend services/handlers
- add tests
- update frontend pages and hooks where relevant
- add evidence to scoreboard

Important branch-fit rules:
- use `cmd/trader` as the authoritative always-on runtime
- use `cmd/research` for research/backtest/orchestration
- keep `frontend_api.go` as the main frontend-facing API surface and extend it or split into handlers cleanly
- preserve current auth shell and routing model
- extend existing strategies and strategy instance loading, do not throw them away

Stop conditions:
- if a change would let chat or AI execute trades directly, block it
- if a change would allow synthetic/fake paths in research/paper, block it
- if a feature cannot be trusted yet, add a gate before sign-off

Start with Phase 0 truth path hardening.
