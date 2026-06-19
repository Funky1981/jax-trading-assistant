# Local Paper Trading Test Runbook

Use this runbook when you want the whole local site online for research and paper-trading validation.

## Start Everything

```powershell
.\start.ps1 -TestMode full
```

This is the canonical local command. It applies migrations, starts Docker services, starts the Vite frontend, starts the Playwright test agent, opens the dashboard, and runs the full platform verification suite.

For a faster smoke run:

```powershell
.\start.ps1 -TestMode quick
```

For CI-style or headless local runs:

```powershell
.\start.ps1 -TestMode full -NoBrowser
```

## Safety Defaults

The startup script defaults to:

- Trader runtime: `paper`
- Research runtime: `research`
- IB bridge: paper trading enabled
- Live trading: disabled
- Container provider config: `config/providers.json`, with broker routed to `ib-bridge.paper`

Do not set `ALLOW_LIVE_TRADING=true` for local validation.

## Local URLs

- Frontend: `http://localhost:5173`
- Trader API: `http://localhost:8081`
- Trader ready endpoint: `http://localhost:8100/ready`
- Research service: `http://localhost:8091`
- IB paper bridge: `http://localhost:8092`
- Agent0 service: `http://localhost:8093`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001`

## What The Full Test Covers

The full platform test writes reports under `Docs/runs/` and checks:

- Auth bootstrap login
- Trader, research, IB bridge, and Agent0 health
- Read-only API smoke checks for signals, artifacts, testing status, readiness, runs, AI decisions, ETF instruments, trading pilot status, and robust performance
- Full Go verification
- Golden replay verification
- ETF backend validation
- Frontend lint, typecheck, and unit tests
- Frontend Playwright e2e suite

Latest successful local full report during setup:

- `Docs/runs/test_run_20260611_105717.md`

## Manual Paper-Trading Walkthrough

1. Open `http://localhost:5173`.
2. Check `Research` and confirm guided research/backtest workflows render.
3. Check `AI Trading` and confirm scanner state and opportunity feeds load.
4. Check `Macro Events` for macro-event context and paper candidate review.
5. Check `Approvals` for candidate approval state and blocked/recovery paths.
6. Check `Manual Trading` only for paper-order UX; do not enable live trading.
7. Check `Analysis` for robust profitability/performance summaries.
8. Check `Notifications` for operational alerts.
9. Use `Docs/runs/` reports as evidence for each validation run.

## Stop Everything

```powershell
.\stop.ps1
```

This stops the frontend dev server, Playwright test agent, and Docker services.

