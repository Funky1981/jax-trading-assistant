# 14 — Codex Prompts

## ETF Hardening

```text
Implement ETF-only hardening for Jax phase one.

Use:
SPY, QQQ, DIA, IWM, XLK, XLF, XLE, SMH, SOXX, TLT, GLD.

Block:
options, leveraged ETFs, inverse ETFs, volatility ETFs, single-name stocks, crypto, forex, futures.

Do not enable live trading.
```

## Event Study Schema

```text
Add event-study schema:
event_windows, event_confounders, event_priced_in_scores, etf_context_snapshots, research_summaries.

Add migrations, down migrations, indexes, constraints, and tests.
```

## Backfill

```text
Implement historical backfill jobs for ETF candles, news/events, macro events, and event-study calculations.

Must be idempotent and provider-aware.
```

## Priced-In Engine

```text
Implement priced-in scoring with verdicts:
not_priced_in, partially_priced_in, priced_in, overreaction, unclear.

priced_in and unclear block candidates.
```
