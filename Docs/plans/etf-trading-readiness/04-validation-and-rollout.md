# ETF Validation and Rollout Plan

## Objective

Define the evidence required to declare the platform ready for controlled phase-1 ETF paper trading.

## Current status

Phase 4 implementation is technically complete.

- The Docker-backed paper stack now serves the ETF catalog, pilot-status, and readiness endpoints with the expected `candidate_approval_only` workflow.
- `./scripts/etf-paper-pilot-evidence.ps1` and `./scripts/test-platform.ps1` now auto-authenticate when frontend API JWT auth is enabled, using the configured bootstrap operator credentials.
- Verified evidence artifacts were generated on 2026-05-14:
	- `Docs/runs/etf-paper-pilot/etf_pilot_evidence_20260514_174732.md`
	- `Docs/runs/test_run_20260514_175050.md`

Phase 4 is not operationally signed off yet. The remaining blockers are the explicit ETF phase-1 approval flags listed in the launch gate below.

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

## Implemented validation entry points

- Backend: `go test ./internal/modules/instruments ./internal/modules/candidates ./internal/modules/approvals ./internal/modules/execution ./cmd/trader`.
- Frontend focused checks: `npm test -- ApprovalsPage OrderTicketPanel`.
- UAT runner: `./scripts/uat-paper-trading.ps1` now probes ETF catalog, pilot status, testing readiness, and runs ETF backend validation. When JWT auth is enabled on the frontend API, it authenticates automatically before hitting protected routes.
- Pilot evidence: `./scripts/etf-paper-pilot-evidence.ps1` captures catalog, pilot-status, readiness, and operator sign-off evidence into `Docs/runs/etf-paper-pilot/`. When JWT auth is enabled on the frontend API, it authenticates automatically before hitting protected routes.
- Readiness endpoint: `/api/v1/testing/readiness` now includes `etfPhase1Readiness` with catalog status, rollout stages, paper-only/manual-entry/live-trading safety state, and sign-off evidence.
- Full regression remains gated by `.\scripts\go-verify.ps1 -Mode standard ...`, `.\scripts\golden-check.ps1 -Mode verify`, frontend typecheck/lint, and Playwright `trading.spec.ts` plus `system.spec.ts`.

## Implemented launch gate

ETF phase 1 remains `not_ready` until all of the following are true:

- catalog loads successfully
- `ETF_PHASE1_AUTOMATED_VALIDATION=passed`
- `ETF_PHASE1_OPERATOR_UAT=passed`
- `ETF_PHASE1_PAPER_PILOT_SIGNOFF=passed`
- `ETF_PHASE1_ENGINEERING_SIGNOFF=true`
- `ETF_PHASE1_OPERATIONS_SIGNOFF=true`
- `ETF_PHASE1_TRADING_RISK_SIGNOFF=true`

These values are operational evidence flags. They should be set only after the corresponding run artifact or written approval exists.
