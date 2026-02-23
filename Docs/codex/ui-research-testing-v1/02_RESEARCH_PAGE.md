# Research Page (Setup + Run)

Route: `/research`

## Primary goals
1. Create/edit **Strategy Instances** (file-backed + DB-backed).
2. Create and run **Research Projects** (parameter sweeps / walk-forward).
3. Kick off a **Backtest run** from a selected instance.

## UI layout (single page with tabs)
Tabs:
- **Instances**
- **Projects**
- **Runs**

### Instances tab
Left: table of instances
- columns: instanceId, strategyId, enabled, universe (count), updatedAt
- actions: View/Edit, Enable/Disable, Run Backtest

Right: Instance editor panel
- JSON editor (monaco optional; start with textarea)
- Validation errors list
- Save (POST/PUT)
- "Export to file" and "Import from file" (download/upload JSON)

Data source:
- `GET /api/v1/instances`

### Projects tab
- Create project form:
  - name, description
  - instanceId (dropdown)
  - date range (from/to)
  - parameter grid (JSON)
  - walk-forward split (train/test)
- Project list table
- Run project button

Data source:
- `GET /api/v1/research/projects`

### Runs tab
- Filter: instanceId, date range
- Table: runId, instanceId, from/to, createdAt, key metrics (pnl, drawdown, winRate)
- Click row -> navigates to `/analysis?runId=...`

Data source:
- `GET /api/v1/backtests/runs?instanceId=...`

## Acceptance criteria
- You can create/update an instance in DB.
- You can export/import the JSON config file.
- You can trigger a backtest run and see it appear in Runs.
