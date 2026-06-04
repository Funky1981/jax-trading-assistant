# ETF Workflow and Operator Guardrails Plan

## Objective

Make the approved ETF workflow the only path operators can use, while keeping the paper-trading UX clear and auditable.

## Current-state signals

- The frontend and trader runtime already expose pilot-mode broker actions through `cmd/trader/frontend_api_trading_pilot.go`.
- Paper execution already expects candidate approval in `cmd/trader/main.go`.
- Existing UAT and production-readiness docs cover generic paper trading, not ETF-specific workflow rules.

## Required outcomes

- one approved ETF submission path
- no manual ETF broker submission that bypasses approval and eligibility checks
- clear operator feedback when ETF trading is blocked
- updated runbooks for ETF-specific approval, rejection, and rollback actions

## Workstreams

### 1. Approval-path consolidation

Review the current trade candidate, signal approval, and pilot broker write flows and define:

- the single approval path allowed for ETF entries
- what legacy or parallel approval routes must be disabled or constrained
- the audit events required for approvals, rejections, and overrides

### 2. Frontend guardrails

Update operator-facing screens so they:

- show ETF eligibility state
- show gate-failure reasons before submission
- prevent unsupported ETF order composition
- distinguish read-only pilot status from ETF approval readiness

### 3. Operator runbook updates

Add ETF-specific procedures for:

- enabling paper ETF trading
- approving an ETF candidate
- investigating a blocked ETF trade
- revoking ETF eligibility or halting ETF trading

### 4. Documentation alignment

Update the related docs once implementation lands:

- production readiness checklist
- paper-trading UAT guide
- operations runbook
- IB guide where connection rules affect ETF rollout

## Acceptance criteria

- an operator cannot submit an ETF trade through a bypass path
- the approved path shows why a trade is allowed or blocked
- ETF actions are traceable in audit logs
- operator documentation matches the enforced runtime behavior

## Exit gate

Do not announce ETF readiness while more than one materially different ETF submission path remains available to operators.

## Implemented workflow

- ETF entries are approval-only through candidate approval and execution instructions.
- Direct broker ETF entries from `/api/v1/broker/orders` and `/api/v1/broker/orders/bracket` are blocked.
- Legacy raw signal approvals from `/api/v1/signals/{id}/approve` reject catalog ETF symbols so operators cannot bypass candidate qualification.
- The frontend order ticket disables manual ETF entry symbols from the active catalog.
- Approval queue rows show ETF policy metadata when candidates carry it.
- The System page shows ETF workflow readiness separately from read-only/trade-enabled pilot status.
- Manual close, cancel, and protect paths stay available for exposure management.

## Phase-3 completion evidence

- Backend guard: `legacySignalApprovalETFBlock` blocks catalog ETF symbols on the legacy signal approval route.
- Operator feedback: blocked ETF manual orders and blocked legacy approvals return explicit approval-workflow reasons.
- Frontend visibility: `/system` displays the ETF entry workflow, policy version, and readiness reasons from `TradingPilotStatus`.
- Runbooks: `Docs/OPERATIONS.md`, `Docs/UAT_PAPER_TRADING.md`, `Docs/PRODUCTION_READINESS.md`, and `Docs/IB_GUIDE.md` describe the phase-1 ETF paper workflow and safety rules.
