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

- Service health: trader/research/ib-bridge/agent0/hindsight.
- API smoke endpoints: signals, artifacts, testing status, runs, AI decisions.
- Backend verification:
  - quick mode: targeted package checks + golden utility tests.
  - full mode: `go-verify` full + golden verify.
- Frontend verification:
  - lint + typecheck + unit tests.
  - full mode adds Playwright e2e (HTML report).

## Output

- Timestamped run output is written to `Docs/runs/` as:
  - `test_run_*.md`
  - `test_run_*.json`

## Parameters

- `-ApiBase` (default `http://localhost:8081`)
- `-ResearchBase` (default `http://localhost:8091`)
- `-IbBridgeBase` (default `http://localhost:8092`)
- `-Agent0Base` (default `http://localhost:8093`)
- `-HindsightBase` (default `http://localhost:8888`)
- `-OutputDir` (default `Docs/runs`)
