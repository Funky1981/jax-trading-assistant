# 13 — Production Readiness Checklist

## Product Definition

Jax is ready only when it demonstrably works as:

```text
Event-driven ETF swing research system
with optional intraday paper-trade mode
```

## Architecture Checklist

```text
[ ] cmd/research owns research only
[ ] cmd/trader owns candidates/approval/execution only
[ ] ib-bridge remains explicit broker boundary
[ ] World Monitor remains separate system
[ ] Adapter sends research triggers only
[ ] AI provider is swappable
[ ] Postgres remains source of truth
```

## Safety Checklist

```text
[ ] ETF-only enforced everywhere
[ ] No options
[ ] No leveraged ETFs
[ ] No inverse ETFs
[ ] No volatility ETFs
[ ] No live trading path enabled
[ ] Paper mode proof required
[ ] Human approval required
[ ] Stop-loss required
[ ] Candidate evidence required
[ ] Guardrail evaluation required
[ ] AI cannot create execution instructions
```

## Swing Checklist

```text
[ ] Swing mode exists
[ ] Swing research mode can run without execution
[ ] Swing paper mode can create approval-gated paper candidates
[ ] Hold period target exists
[ ] Max hold days exists
[ ] Overnight risk explicitly allowed
[ ] Weekend hold disabled by default
[ ] Daily revalidation required
[ ] Thesis invalidators stored
[ ] Calendar risk check exists
[ ] Revalidation evidence stored
```

## Intraday Checklist

```text
[ ] Intraday mode exists
[ ] Flatten by close required
[ ] Overnight risk blocked
[ ] RTH required
[ ] Fresh quote required
[ ] Spread required
[ ] Same-session expiry required
```

## Current Blockers Closed

```text
[ ] gofmt issue fixed
[ ] service connectivity fixed
[ ] ETF catalog endpoint passes
[ ] pilot-status endpoint passes
[ ] testing-readiness endpoint passes
[ ] hardcoded stale_quote_pass removed
[ ] hardcoded paper_mode_pass removed
[ ] event-study bounds fixed
[ ] central ETF universe used
[ ] confounders wired
[ ] broker paper-mode proof captured
[ ] post-trade reflection proof captured
```

## Evidence Checklist

```text
[ ] Unit test report archived
[ ] Integration test report archived
[ ] UAT paper trading report archived
[ ] UAT swing research report archived
[ ] Broker paper mode JSON archived
[ ] No live trading proof archived
[ ] Candidate approval proof archived
[ ] Paper execution instruction proof archived
[ ] Revalidation proof archived
[ ] Reflection proof archived
```

## Signoff Checklist

```text
[ ] Operator signoff
[ ] Engineering signoff
[ ] Risk signoff
[ ] Evidence reviewed
[ ] Known limitations listed
[ ] Next phase explicitly deferred
```

## Phase 1 Release Verdict

Allowed release label:

```text
Paper Research Pilot
```

Do not call it:

```text
Production Trading
Live Trading
Autonomous Trading
Profitable Trading System
```

## Phase 2 Deferred Items

Do not include in phase 1:

```text
live trading
short selling
options
leveraged ETFs
inverse ETFs
volatility ETFs
auto approval
auto paper submit without evidence
portfolio optimisation
paid signal resale
```
