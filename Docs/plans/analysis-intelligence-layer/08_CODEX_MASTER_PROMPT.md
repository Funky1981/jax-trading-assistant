# 08 — Codex Master Prompt

Use this prompt to build the Analysis Intelligence Layer.

```text
You are working in the Jax Trading Assistant repository.

Goal:
Build the Analysis Intelligence Layer so Jax can perform structured technical analysis and fundamental analysis before creating paper-only trade candidates.

Read first:
- Docs/plans/world-monitor-jax-awareness/README.md
- Docs/plans/macro-reaction-engine/README.md
- Docs/plans/analysis-intelligence-layer/README.md
- Docs/plans/analysis-intelligence-layer/01_TECHNICAL_ANALYSIS_ENGINE.md
- Docs/plans/analysis-intelligence-layer/02_FUNDAMENTAL_ANALYSIS_ENGINE.md

Important:
Jax must not trade from a headline alone.
Fundamental analysis must consider other events that could affect the ETF/stock/sector, not just the headline.
Technical analysis must use structured candle/level/reaction checks, not vague AI chart commentary.

Build order:
1. Technical Analysis Engine
2. Fundamental Analysis Engine
3. Event Playbook Library
4. Analyst Scoring Model
5. Multi-Analyst Review Flow
6. Analyst Memory and Feedback
7. TA/FA Backtesting and UAT

Hard constraints:
- No live trading
- No broker orders
- No auto-approval
- No candidate without TA + FA + risk pass
- No candidate when chart confirmation is missing
- No candidate when fundamental impact is unclear
- No candidate when another event explains the move
- No candidate when priced-in verdict is priced_in or unclear
- No options, single stocks, inverse ETFs, leveraged ETFs, volatility ETFs, crypto, forex, or futures in phase 1

Design:
- Keep trader runtime deterministic
- Keep analysis services testable
- LLM may summarise deterministic findings but cannot override hard vetoes
- Persist snapshots and decisions
- Every candidate must link to evidence

Deliver:
- migrations
- domain models
- services
- tests
- API updates if needed
- evidence output contracts
- UAT fixtures
- documentation updates

Validation:
- gofmt
- targeted go test for touched packages
- migration tests
- no broker write tests
- deterministic fixture tests
```
