# Jax Event-Driven ETF Swing Research Plan

## Target Product

Jax becomes an **event-driven ETF swing research system with optional intraday paper-trade mode**.

The primary mode is no longer "same-session trade first". The primary product is:

```text
World/news event happens
  -> Jax verifies and classifies the event
  -> Jax maps it to allowlisted ETFs
  -> Jax compares historic reactions across intraday and multi-day windows
  -> Jax checks whether the move is already priced in
  -> Jax checks confounders
  -> Jax builds a swing thesis and an intraday alternative if appropriate
  -> Jax creates evidence-backed paper candidates only
  -> Human approves/rejects
  -> Paper execution only after approval
  -> Daily revalidation while open
  -> Reflection/outcome recorded
```

## Non-Negotiables

- ETF-only in phase 1.
- Paper-only until long-running test evidence proves quality.
- No options.
- No leveraged ETFs in phase 1.
- No inverse ETFs in phase 1.
- No volatility ETFs in phase 1.
- AI provider is configurable; DeepSeek can be Chris's personal cheap provider.
- AI can classify, summarise, score, and explain. It cannot create execution instructions.
- Strategy output is a candidate, not an order.
- Broker execution is paper-only and approval-gated.
- Every candidate must have an evidence bundle.
- Every paper execution must produce post-trade memory/reflection evidence.
- Existing `cmd/trader`, `cmd/research`, `ib-bridge`, `agent0-service`, Postgres, and frontend boundaries must be extended, not replaced.

## Current Branch Review Summary

The redesign/work branch already contains much of the ETF news research foundation:

- `cmd/trader` deterministic trading/API runtime.
- `cmd/research` orchestration/research/backtest/memory runtime.
- Postgres + pgvector.
- ETF allowlist policy.
- Candidate trade and approval tables.
- Event storage foundation.
- Market data tables.
- Paper trading UAT docs.
- IB bridge/paper trading plumbing.
- Testing/readiness API surfaces.

But the current branch is not ready for a swing-first release because these issues must be fixed first:

1. Event-study bounds are based on `time.Now()` instead of the event time window envelope.
2. ETF allowlist is duplicated in code and can drift from central policy/config.
3. Confounder analysis schema exists but the current event-study path writes `nil` confounders.
4. Evidence bundles currently hardcode `stale_quote_pass=true` and `paper_mode_pass=true`.
5. Swing windows are not first-class; current defaults stop at `event_to_+1d`.
6. Current risk summary assumes `flatten_by_close=true`, so it is intraday-first.
7. Prompt 10 UAT still lacks fresh broker paper-mode proof and post-trade reflection proof.
8. The last evidence checklist shows service connectivity/readiness blockers and a `gofmt` blocker.

## Build Order

Do not start with UI or strategy creativity. Fix the foundation first.

```text
0. Close current branch blockers
1. Centralise ETF universe and policy
2. Add trading horizon/mode domain model
3. Extend event-study windows for swing research
4. Add real confounder detection
5. Replace hardcoded guardrail passes with runtime checks
6. Add swing thesis/evidence bundle contract
7. Add swing candidate lifecycle and daily revalidation
8. Preserve optional intraday paper mode
9. Wire frontend approval and evidence views
10. Add production-grade UAT and evidence capture
11. Only then consider live readiness in a separate future phase
```

## Output Folder Contents

- `00-redesign-branch-review-and-blockers.md`
- `01-target-architecture.md`
- `02-trading-modes-and-horizons.md`
- `03-data-model-and-migrations.md`
- `04-event-study-and-historical-research.md`
- `05-swing-research-engine.md`
- `06-intraday-paper-mode.md`
- `07-risk-guardrails-and-execution-boundaries.md`
- `08-ai-provider-and-research-prompts.md`
- `09-world-monitor-adapter-compatibility.md`
- `10-frontend-mobile-and-approval.md`
- `11-testing-uat-and-evidence.md`
- `12-codex-implementation-prompts.md`
- `13-production-readiness-checklist.md`
