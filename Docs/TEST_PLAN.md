# Test Plan

## Objective

Validate the current production path end-to-end for:
- runtime health
- API contract availability
- backend regression stability
- frontend correctness
- replay/golden comparison utility behavior

## Entry Criteria

- `docker compose up -d` completed successfully
- `jax-trader` and `jax-research` are healthy
- Postgres migrations are applied
- frontend dependencies are installed (`frontend/node_modules`)

## Automated Execution

### Quick Gate (recommended for every change)

```powershell
.\scripts\test-platform.ps1 -Mode quick
```

### Full Gate (pre-release)

```powershell
.\scripts\test-platform.ps1 -Mode full
```

### Full Gate + Visual E2E Report

```powershell
.\scripts\test-platform.ps1 -Mode full -OpenVisualReport
```

## Coverage Matrix

1. Health checks:
   - `8081` trader API
   - `8091` research
   - `8092` ib-bridge
   - `8093` agent0-service
   - `8888` hindsight
2. API smoke:
   - `/api/v1/signals`
   - `/api/v1/artifacts`
   - `/api/v1/testing/status`
   - `/api/v1/runs`
   - `/api/v1/ai-decisions`
3. Backend quality:
   - `scripts/go-verify.ps1` quick/full modes
   - `scripts/golden-check.ps1 -Mode verify`
4. Frontend quality:
   - `npm run lint`
   - `npm run typecheck`
   - `npm run test`
   - `npx playwright test --reporter=html` (full mode)

## Evidence Output

Each automated run writes:
- `Docs/runs/test_run_<timestamp>.md`
- `Docs/runs/test_run_<timestamp>.json`

Playwright full runs generate:
- `frontend/playwright-report/index.html`

## Manual Spot Checks (Release Candidate)

1. Login/auth status path works (`/auth/status`, `/auth/login` if enabled).
2. Strategy/artifact list pages load with no frontend console errors.
3. Run detail and AI decisions pages load and show timeline data.
4. Artifact validation endpoint (`/api/v1/artifacts/{id}/validate`) returns trust-gate evidence.
5. Audit trail queries in `Docs/AUDIT_TRAIL.md` return expected rows.
