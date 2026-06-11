# Quick Start

## Prerequisites

- Docker Desktop
- Node.js 20+
- Go 1.22+ (optional for local Go runs)

## Start Everything

```powershell
.\start.ps1
```

This starts Docker services, applies migrations, starts the local Vite frontend, starts the Playwright test agent, and opens the dashboard.

Run startup plus a quick whole-platform smoke test:

```powershell
.\start.ps1 -TestMode quick
```

Run the full local site test, including Playwright e2e:

```powershell
.\start.ps1 -TestMode full
```

Force rebuild first:

```powershell
.\start.ps1 -Build -TestMode quick
```

Core runtime services:
- `jax-trader` (`http://localhost:8081/health`, runtime port `8100`)
- `jax-research` (`http://localhost:8091/health`)
- `ib-bridge` (`http://localhost:8092/health`)
- `agent0-service` (`http://localhost:8093/health`)

Frontend URL: `http://localhost:5173`

Full paper-trading test runbook: `Docs/LOCAL_PAPER_TRADING_TESTING.md`

## Manual Frontend Only

```powershell
cd frontend
npm install
npm run dev
```

Use this only when the backend is already running and you want to restart the frontend manually.

## Health Checks

```powershell
curl http://localhost:8081/health
curl http://localhost:8091/health
curl http://localhost:8092/health
curl http://localhost:8093/health
```

## Authentication (Optional)

Enable JWT mode by setting `JWT_SECRET` for `jax-trader`.

Optional first-user bootstrap:
- `AUTH_BOOTSTRAP_USERNAME`
- `AUTH_BOOTSTRAP_PASSWORD`
- `AUTH_BOOTSTRAP_ROLE` (`admin` or `user`, default `admin`)

## Common Logs

```powershell
docker compose logs -f jax-trader
docker compose logs -f jax-research
docker compose logs -f ib-bridge
docker compose logs -f agent0-service
```

## Stop Stack

```powershell
docker compose down
```

Preferred full stop:

```powershell
.\stop.ps1
```

Full cleanup (removes volumes):

```powershell
docker compose down -v
```

## Validate and Audit

- Automated validation plan: `Docs/TEST_PLAN.md`
- Run quick automated checks: `.\scripts\test-platform.ps1 -Mode quick`
- Trade and decision trace queries: `Docs/AUDIT_TRAIL.md`
