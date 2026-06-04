# ETF Trading Readiness Plan

This plan set defines the minimum work needed to move Jax from general paper-trading support to a controlled phase-1 ETF trading posture.

## Why this plan exists

The repository already has the core paper-trading building blocks:

- runtime mode controls in `Docs/OPERATIONS.md`
- IB bridge paper-trading defaults in `Docs/IB_GUIDE.md`
- a direct paper execution guard in `cmd/trader/main.go`
- pilot broker write paths in `cmd/trader/frontend_api_trading_pilot.go`
- paper-trading UAT coverage in `Docs/UAT_PAPER_TRADING.md`

What is still missing is a strict ETF-specific operating model: instrument eligibility, execution guardrails, workflow enforcement, and ETF-focused validation.

## Plan structure

1. `01-scope-and-instrument-policy.md`
   - defines the allowed ETF universe and the policy boundaries for phase 1
2. `02-execution-and-risk-controls.md`
   - defines trading-time protections needed before ETF trading should be enabled
3. `03-workflow-and-operator-guardrails.md`
   - defines approval-path, frontend, and operational constraints so ETF trades cannot bypass review
4. `04-validation-and-rollout.md`
   - defines test evidence, staged rollout, and sign-off needed for launch

## Phase-1 target state

Jax should be considered ETF-ready for phase 1 only when all of the following are true:

- only approved plain-vanilla ETFs can enter the trading flow
- leveraged, inverse, volatility, and other excluded ETF classes are blocked
- ETF trades can only be submitted through the approved workflow
- quote freshness, spread, and session guards are enforced before submission
- protective exits and audit fields are present for ETF orders
- ETF-specific automated and manual validation passes in paper trading

## Implementation status

Implemented phase-1 controls:

- `config/etf-instruments.json` is the versioned ETF catalog.
- `/api/v1/instruments/etfs` exposes the active catalog and thresholds.
- ETF manual entry orders are blocked from direct broker submit endpoints.
- Candidate, approval, and execution paths apply ETF eligibility checks.
- Execution enforces quote freshness, bid/ask presence, spread, session, stop-loss, flatten-by-close, and paper-only requirements.
- `/api/v1/testing/readiness` exposes ETF phase-1 rollout stages and sign-off state.
- `.\scripts\uat-paper-trading.ps1` includes ETF catalog, pilot status, readiness, and backend validation checks.
- `.\scripts\etf-paper-pilot-evidence.ps1` captures pilot evidence before sign-off flags are set.

## Recommended work order

1. Lock scope and instrument policy.
2. Add backend execution and risk gates.
3. Restrict frontend and operator workflows to the approved ETF path.
4. Add ETF-specific validation and staged rollout evidence.

## Out of scope for phase 1

- options trading
- futures trading
- leveraged or inverse ETF support
- volatility-product trading
- fully autonomous live ETF trading without human approval
