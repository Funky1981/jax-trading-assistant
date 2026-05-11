# ETF Validation and Rollout Plan

## Objective

Define the evidence required to declare the platform ready for controlled phase-1 ETF paper trading.

## Validation principles

- reuse the existing paper-trading, production-readiness, and regression workflows where possible
- add ETF-specific coverage instead of relying on generic equity coverage
- require both automated evidence and operator-run UAT
- keep rollout reversible

## Required evidence

### 1. Backend automated coverage

Add targeted coverage for:

- ETF allowlist acceptance and rejection
- excluded ETF-class rejection
- stale-quote rejection
- spread and liquidity rejection
- regular-trading-hours enforcement
- protective-exit validation
- audit payload and persistence checks

### 2. Frontend and API workflow coverage

Add coverage for:

- approval flow for an allowed ETF
- blocked flow for an ineligible ETF
- blocked flow for stale or wide-spread ETF quotes
- UI messaging for read-only or paper-only restrictions

### 3. Manual UAT

Create an ETF-specific UAT sequence that proves:

- operator can approve an allowed ETF candidate
- trade reaches paper broker path only after approval
- blocked ETFs fail with clear reasons
- kill-switch or revoke flow works without ambiguity

### 4. Rollout stages

Use staged rollout:

1. implementation complete in development
2. automated ETF validation green
3. paper-only operator UAT on the initial allowlist
4. limited paper pilot with explicit sign-off
5. broaden allowlist only after review of pilot evidence

### 5. Sign-off

Require sign-off from:

- engineering
- operations
- trading/risk owner

## Recommended command set to reuse

These existing entry points should be extended rather than replaced:

- `Docs/UAT_PAPER_TRADING.md`
- `Docs/PRODUCTION_READINESS.md`
- `skills/jax-go-change-workflow/scripts/go-verify.ps1`
- `skills/jax-golden-replay-regression/scripts/golden-check.ps1`

## Launch gate

ETF paper trading is ready only when:

- all previous plan documents are complete
- ETF-specific automation is passing
- ETF-specific UAT is documented and passing
- rollback and revoke procedures are confirmed
- the initial allowlist has written sign-off

## Rollback rule

Any of the following should pause ETF rollout immediately:

- an ineligible ETF reaches submission
- an ETF bypasses approval
- a stale-quote or spread gate fails open
- audit records cannot explain an ETF order decision
