# 15 — Codex Implementation Prompts

## Prompt 1 — ETF-Only Hardening

```text
Implement ETF-only hardening for Jax phase one.

Use the approved ETF list:
SPY, QQQ, DIA, IWM, XLK, XLF, XLE, SMH, SOXX, TLT, GLD.

Block:
options, leveraged ETFs, inverse ETFs, volatility ETFs, single-name stocks, crypto, forex, futures.

Tasks:
1. Audit current defaults for non-ETF symbols.
2. Remove non-ETF defaults from ETF phase-one paths.
3. Ensure config/etf-instruments.json is the ETF source of truth.
4. Enforce allowlist before candidate creation, approval, and execution instruction.
5. Add tests for approved ETFs and rejected symbols.
6. Update docs/UAT.

Do not enable live trading.
```

## Prompt 2 — Event Study Schema

```text
Add the historical ETF event-study schema.

Create migrations for:
- event_windows
- event_confounders
- event_priced_in_scores
- etf_context_snapshots
- research_summaries

Include indexes, uniqueness constraints, down migrations, and integration tests.

Reuse existing event and candle tables.
Do not create a separate database.
```

## Prompt 3 — Historical Backfill

```text
Implement historical backfill jobs for ETF candles, news/events, macro calendar events, and event-study calculations.

Requirements:
- idempotent
- provider-aware
- stores raw and normalized data
- supports the 11 ETF allowlist
- can rerun safely
- outputs run summaries

Start with provider abstractions already present in Jax.
```

## Prompt 4 — ETF Event Classification

```text
Implement rule-based ETF event classification.

Categories:
macro_rates, inflation, central_bank, energy_oil, semiconductor_ai, broad_market, financial_credit, geopolitical, gold_safe_haven, small_caps, technology, earnings_bellwether, regulation, unknown.

Map events to ETFs using deterministic rules.
Unclear events must not create trade candidates.
Store reason and confidence.
```

## Prompt 5 — Priced-In Engine

```text
Implement the priced-in scoring engine.

Inputs:
event timestamp, ETF candles, benchmark, volume, spread, confounders.

Outputs:
not_priced_in, partially_priced_in, priced_in, overreaction, unclear.

Hard rule:
priced_in or unclear must block trade candidates.

Store score and reason.
Add tests.
```

## Prompt 6 — Evidence Bundle Builder

```text
Implement research evidence bundle generation.

Bundle must include:
event summary, source list, selected ETF, why this ETF, price reaction windows, confounders, priced-in verdict, risk notes, guardrail results, beginner summary.

Store in research_summaries and link to candidate trades where possible.
```

## Prompt 7 — AI Guardrails

```text
Constrain AI to advisory output only.

AI may summarize evidence and recommend trade/wait/reject.
AI must not create orders, override guardrails, invent missing data, or approve trades.

Validate AI output schema and reject unsafe output.
Add tests for failed guardrails overriding AI recommendations.
```

## Prompt 8 — Mobile Approval

```text
Implement phase-one mobile approval using Telegram or a simple notification outbox.

Requirements:
- sends beginner-friendly candidate summary
- approve/reject/snooze actions
- one-time token
- expiry
- audit trail
- approval creates paper execution instruction only
- no live trading possible
```

## Prompt 9 — Beginner UX

```text
Add beginner-friendly UI surfaces:
- ETF universe screen
- strategy cards
- candidate evidence screen
- research timeline
- simple/detailed/technical toggle

Every trade candidate must clearly explain:
what happened, why this ETF, priced-in verdict, risks, stop-loss, target, and walk-away reason.
```

## Prompt 10 — Full UAT

```text
Add ETF news research UAT.

Prove:
- ETF-only defaults
- event ingestion
- event study generation
- priced-in scoring
- evidence bundle
- candidate creation
- approval
- paper execution instruction
- broker paper mode
- no live trading
- post-trade memory/reflection
```
