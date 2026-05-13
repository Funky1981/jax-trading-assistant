# Paper Trading UAT

Run the platform test runner from repo root:

```powershell
.\scripts\uat-paper-trading.ps1 -Mode quick
```

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
- `GET /api/v1/trading/pilot-status` reports `etfPhase1Enabled=true` and `etfEntryWorkflow=approval_only`.
- An approved allowlist candidate such as `SPY` reaches the approval queue with ETF policy metadata.
- A manual `SPY` order from the order ticket is blocked with `manual ETF entry orders must use the approval workflow`.
- An excluded ETF such as `TQQQ` is blocked before approval.
- A stale quote, missing bid/ask, wide spread, after-hours timestamp, or missing stop loss rejects before broker submission.
- Close, cancel, and protection actions remain available for managing existing paper exposure.

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
