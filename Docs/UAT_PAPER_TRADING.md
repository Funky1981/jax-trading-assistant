# Paper Trading UAT

Run the platform test runner from repo root:

```powershell
.\scripts\uat-paper-trading.ps1 -Mode quick
```

If frontend API auth is enabled, the runner automatically logs in with the configured bootstrap operator credentials before probing protected routes.

For full validation (includes Playwright HTML report generation):

```powershell
.\scripts\uat-paper-trading.ps1 -Mode full
```

To open the Playwright report after full run:

```powershell
.\scripts\uat-paper-trading.ps1 -Mode full -OpenVisualReport
```

## What It Checks

- Service health: trader/research/ib-bridge/agent0.
- API smoke endpoints: signals, artifacts, testing status, runs, AI decisions.
- Backend verification:
  - quick mode: targeted package checks + golden utility tests.
  - full mode: `go-verify` full + golden verify.
- Frontend verification:
  - lint + typecheck + unit tests.
  - full mode adds Playwright e2e (HTML report).

## ETF Phase-1 UAT

Before any ETF paper pilot session, verify:

- `GET /api/v1/instruments/etfs` returns catalog version `phase1-2026-05-13`.
- `GET /api/v1/trading/pilot-status` reports `etfPhase1Enabled=true` and `etfEntryWorkflow=candidate_approval_only`.
- `GET /api/v1/testing/readiness` includes `etfPhase1Readiness` with catalog, rollout stage, and sign-off evidence.
- An approved allowlist candidate such as `SPY` reaches the approval queue with ETF policy metadata.
- A manual `SPY` order from the order ticket is blocked with `manual ETF entry orders must use the approval workflow`.
- An excluded ETF such as `TQQQ` is blocked before approval.
- A stale quote, missing bid/ask, wide spread, after-hours timestamp, or missing stop loss rejects before broker submission.
- Close, cancel, and protection actions remain available for managing existing paper exposure.

ETF readiness sign-off is explicit. Set these only after evidence is reviewed:

```powershell
$env:ETF_PHASE1_AUTOMATED_VALIDATION="passed"
$env:ETF_PHASE1_OPERATOR_UAT="passed"
$env:ETF_PHASE1_PAPER_PILOT_SIGNOFF="passed"
$env:ETF_PHASE1_ENGINEERING_SIGNOFF="true"
$env:ETF_PHASE1_OPERATIONS_SIGNOFF="true"
$env:ETF_PHASE1_TRADING_RISK_SIGNOFF="true"
```

To capture the operator evidence bundle before setting those values:

```powershell
.\scripts\etf-paper-pilot-evidence.ps1
```

If frontend API auth is enabled, the evidence script automatically logs in with the configured bootstrap operator credentials before probing protected routes.

Use the sign-off switches only when the matching evidence has been reviewed:

```powershell
.\scripts\etf-paper-pilot-evidence.ps1 `
  -AutomatedValidationPassed `
  -OperatorUATPassed `
  -PaperPilotSignedOff `
  -EngineeringSignoff `
  -OperationsSignoff `
  -TradingRiskSignoff
```

## Output

- Timestamped run output is written to `Docs/runs/` as:
  - `test_run_*.md`
  - `test_run_*.json`

## Parameters

- `-ApiBase` (default `http://localhost:8081`)
- `-ResearchBase` (default `http://localhost:8091`)
- `-IbBridgeBase` (default `http://localhost:8092`)
- `-Agent0Base` (default `http://localhost:8093`)
- `-ResearchBase` (default `http://localhost:8091`)
- `-OutputDir` (default `Docs/runs`)
