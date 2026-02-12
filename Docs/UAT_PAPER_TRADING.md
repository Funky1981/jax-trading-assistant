# Paper Trading UAT Guide

This guide describes how to run the end-to-end UAT checks for paper trading using the scripted runner.

## Script

Run the UAT script from repo root:

```powershell
.\scripts\uat-paper-trading.ps1
```

### Options

```powershell
.\scripts\uat-paper-trading.ps1 `
  -Symbol AAPL `
  -ExecuteOrders `
  -JwtToken "<jwt token>"
```

Parameters:
- `-Symbol` selects the symbol used in the orchestration test.
- `-ExecuteOrders` enables approval and order execution.
- `-JwtToken` passes an auth token for protected endpoints.
- `-SkipDb` skips database validation checks.
- `-OutputPath` sets the report output path.

## What It Validates

- Service health for core components.
- Market data freshness and basic ingestion visibility.
- Orchestration API can trigger a run and return status.
- Optional approval and execution path.

## Output

The script writes a report to `Docs/uat/` by default, with a timestamped filename.
