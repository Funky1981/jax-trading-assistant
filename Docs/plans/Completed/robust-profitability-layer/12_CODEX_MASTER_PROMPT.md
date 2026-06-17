# 12 — Codex Master Prompt

Use this prompt to build the Robust Profitability Layer.

```text
You are working in the Jax Trading Assistant repository.

Goal:
Build the Robust Profitability Layer so Jax becomes stricter, safer, and more measurable before any live trading is considered.

Read first:
- Docs/plans/world-monitor-jax-awareness/README.md
- Docs/plans/macro-reaction-engine/README.md
- Docs/plans/analysis-intelligence-layer/README.md
- Docs/plans/robust-profitability-layer/README.md

Build order:
1. Market Regime Engine
2. Cross-Asset Confirmation
3. Economic Calendar Integration
4. Confounder Engine
5. Liquidity and Execution Quality
6. Position Sizing and Portfolio Risk
7. Strategy Playbooks
8. Walk-Away Engine
9. Post-Trade Review and Learning
10. Performance Dashboard
11. Monte Carlo and Stress Testing

Hard constraints:
- No live trading
- No broker orders
- No auto-approval
- No candidate if market regime conflicts
- No candidate if cross-asset confirmation conflicts
- No candidate if major confounder exists
- No candidate if execution quality is poor
- No candidate if sizing cannot be calculated
- No candidate if risk limits are breached
- No candidate without a matched strategy playbook
- No candidate when walk-away blocker exists

Implementation style:
- Deterministic services first
- Persist snapshots and decisions
- LLMs may summarise but never override hard vetoes
- Use fixtures before paid/live providers
- Tests must prove no broker writes
- Every rejection should have a readable reason

Deliver:
- migrations
- domain models
- services
- APIs where needed
- tests
- fixture data
- dashboard metrics
- documentation updates

Validation:
- gofmt
- targeted go test for touched packages
- migration tests
- no broker write tests
- deterministic UAT fixtures
```
