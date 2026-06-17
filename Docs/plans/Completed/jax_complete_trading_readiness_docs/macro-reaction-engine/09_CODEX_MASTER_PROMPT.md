# 09 — Codex Master Prompt

Use this prompt at the start of the build.

```text
You are working in the Jax Trading Assistant repository.

Goal:
Build the Macro Reaction Engine that connects accepted macro/news research triggers to chart-confirmed, evidence-backed, paper-only candidate trades.

Important existing context:
- World Monitor is only an awareness radar.
- The adapter only sends research triggers.
- Jax must validate, research, check charts, score priced-in risk, check confounders, build evidence, and only then create a candidate.
- No live trading.
- No broker order creation.
- No options, single stocks, inverse ETFs, leveraged ETFs, volatility ETFs, crypto, forex, or futures in phase 1.
- ETF-only allowlist: SPY, QQQ, DIA, IWM, XLK, XLF, XLE, SMH, SOXX, TLT, GLD.
- Human approval remains mandatory.

Read these docs first:
1. Docs/plans/world-monitor-jax-awareness/README.md
2. Docs/plans/world-monitor-jax-awareness/02-signal-contract.md
3. Docs/plans/world-monitor-jax-awareness/03-jax-ingestion-flow.md
4. Docs/plans/world-monitor-jax-awareness/04-guardrails-and-safety.md
5. Docs/plans/macro-reaction-engine/README.md
6. Docs/plans/macro-reaction-engine/00_IMPLEMENTATION_ORDER.md

Build order:
1. Macro event model and calendar data
2. Candle/chart reaction engine
3. ETF mapping and scenario playbooks
4. Priced-in and confounder checks
5. Evidence bundle builder
6. Candidate trade generator
7. UI/API integration
8. Backtesting and UAT

Hard constraints:
- Do not create broker orders.
- Do not enable live trading.
- Do not bypass existing risk/approval flow.
- Do not allow World Monitor payloads to become trades.
- Do not create candidates without evidence bundles.
- Do not create candidates if chart confirmation is missing.
- Do not create candidates if priced-in verdict is priced_in or unclear.
- Do not create candidates if high-severity confounders exist.
- Do not create candidates outside the ETF allowlist.

Implementation style:
- Keep trader runtime deterministic.
- Keep fuzzy/research logic in research runtime or clearly separated modules.
- Use small packages with tests.
- Add migrations with constraints and indexes.
- Add deterministic fixture tests.
- Add API tests for rejection paths.
- Add audit trail from event → reaction → evidence → candidate.

Validation:
- gofmt
- targeted go test for touched packages
- migration tests
- frontend tests only for touched UI
- prove no broker write occurs in macro event tests

Deliverables:
- migrations
- domain models
- validation
- services
- API endpoints
- tests
- UI screens if phase requires it
- UAT fixture data
- updated docs
```
