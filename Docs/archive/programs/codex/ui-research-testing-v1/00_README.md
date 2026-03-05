# UI: Research, Analysis, Testing pages (Jax Trader)

Goal: add first-class UI to **set up** and **inspect**:
- Strategy Instances (configs that can be backtested/paper traded)
- Research Projects (parameter sweeps + walk-forward)
- Backtest Runs (results + trades)
- Testing/Trust Gates (data reconciliation + execution correctness proof)

This pack is written for Copilot/Codex to implement in your existing React frontend and Go services.

Repo facts used:
- Routes currently live in `frontend/src/app/App.tsx` with placeholders.
- Navigation lives in `frontend/src/components/layout/AppShell.tsx`.
- Existing API client pattern in `frontend/src/data/*-service.ts` and `frontend/src/data/http-client.ts`.

---

## What to add (high-level)

### Frontend routes
Add routes:
- `/research` (Strategy Instances + Projects)
- `/analysis` (Runs + run comparison)
- `/testing` (Trust Gates + diagnostics)

### Sidebar nav
Add nav items:
- Research
- Analysis
- Testing

### Backend/API (minimal)
Add endpoints in `jax-api` (or a dedicated service behind it):
- Strategy Instances:
  - `GET /api/v1/instances`
  - `GET /api/v1/instances/{id}`
  - `POST /api/v1/instances`
  - `PUT /api/v1/instances/{id}`
  - `POST /api/v1/instances/{id}/enable|disable`
- Backtest:
  - `POST /api/v1/backtests/run`
  - `GET /api/v1/backtests/runs?instanceId=...`
  - `GET /api/v1/backtests/runs/{runId}`
- Research Projects:
  - `GET /api/v1/research/projects`
  - `POST /api/v1/research/projects`
  - `GET /api/v1/research/projects/{id}`
  - `POST /api/v1/research/projects/{id}/run` (kicks sweeps)
  - `GET /api/v1/research/projects/{id}/runs`
- Testing/Trust:
  - `GET /api/v1/testing/status` (latest recon run summaries)
  - `POST /api/v1/testing/recon/data`
  - `POST /api/v1/testing/recon/pnl`
  - `POST /api/v1/testing/failure-tests/run`

The UI can initially show "Not implemented" for endpoints until the backend lands, but the shape must be stable.

---

## Deliverables in this pack
- Page specs (UX and components)
- API client additions
- Route/nav changes
- Minimal backend contract (DTOs + endpoints)
- Acceptance criteria
