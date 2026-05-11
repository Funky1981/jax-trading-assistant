# ETF Execution and Risk Controls Plan

## Objective

Add the backend execution controls required so approved ETFs can be traded in paper mode without relying on operator discipline alone.

## Current-state signals

- Direct paper execution is already blocked unless it comes through the internal approval path in `cmd/trader/main.go`.
- Quote staleness is configured in `config/jax-market.json` and `cmd/trader/market_ingester.go`, but that is not yet the same as a pre-trade ETF rejection rule.
- The platform persists bid/ask data and already has microstructure utilities, which is enough to support spread and liquidity gates.
- IB paper/live connection guidance exists in `Docs/IB_GUIDE.md`, so the remaining work is enforcement and consistency.

## Required controls

### 1. Eligibility gate before submission

Every ETF order path should reject:

- symbols outside the approved ETF allowlist
- excluded ETF classes
- orders whose runtime mode is not explicitly permitted

### 2. Quote freshness gate

Reject ETF submissions when quote data is older than the approved freshness threshold for the environment.

Plan details:

- define the source of truth for quote timestamp comparison
- reject instead of warning when freshness fails
- surface the rejection reason in API and UI responses

### 3. Spread and liquidity gate

Before ETF submission, enforce:

- maximum spread in basis points
- minimum bid/ask presence
- minimum size or liquidity threshold where applicable

### 4. Session gate

Define whether phase 1 is regular-trading-hours only and enforce that decision in the execution path.

### 5. Protective-exit requirement

Phase 1 ETF entries should require an exit plan such as:

- stop loss
- flatten-by-close
- both, when the strategy requires it

### 6. Paper/live safety consistency

Unify and document:

- approved IB paper connection settings
- explicit live-trading enablement rules
- environment checks that prevent ambiguous runtime behavior

### 7. Audit and persistence additions

Ensure ETF trades persist enough metadata to answer:

- why the symbol was eligible
- which gate checks passed
- whether the trade ran in paper or live mode
- what protective exits were attached

## Deliverables

- an execution gate checklist mapped to each order entry path
- a risk-threshold decision table for ETF paper trading
- operator-visible error reasons for rejected ETF orders
- updated readiness docs once the gates exist

## Acceptance criteria

- an unapproved ETF cannot reach broker submission
- stale or illiquid ETF quotes cause hard rejection
- ETF orders outside the approved session are blocked
- ETF entries cannot be accepted without required protective controls
- audit records clearly show gate decisions

## Exit gate

Do not enable ETF paper trading until all mandatory pre-trade and audit controls reject unsafe submissions by default.
