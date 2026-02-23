# Testing Page (Trust Gates + Diagnostics)

Route: `/testing`

## Primary goals
- Show whether the system is *trustworthy enough* to proceed.
- Provide buttons to run reconciliation jobs (paper mode) and display results.

## UI layout
Sections:

### 1) Trust Gates Checklist
Render Gate 0..7 with status:
- Not started / Passing / Failing
- Last run timestamp
- Link to latest artifact (if stored)

Data source:
- `GET /api/v1/testing/status`

### 2) Quick Diagnostics
Cards:
- ib-bridge health
- jax-market health
- jax-api health
- trade-executor health
- DB health

Data sources:
- existing health endpoints or new aggregated endpoint.

### 3) Run tests buttons (paper only)
Buttons:
- Run Data Reconciliation
- Run P/L Reconciliation
- Run Failure Test Suite

These call:
- `POST /api/v1/testing/recon/data`
- `POST /api/v1/testing/recon/pnl`
- `POST /api/v1/testing/failure-tests/run`

All must require paper mode (guarded server-side).

## Acceptance criteria
- You can see the latest pass/fail state per gate.
- Running a recon test updates status and shows a summary.
