# How to trigger research/tests (and record it)

## Research triggers (UI-driven)
From the UI you will trigger:
- Backtest run for an instance
- Research project run (parameter sweep + walk-forward)
- Paper session run (signal generation window)

Each trigger must create a `run` row:
- `run_id`
- `run_type`: `backtest` | `research` | `paper_session` | `live_session`
- `requested_by` (user)
- `requested_at`
- `config_snapshot` (JSONB) including instance config + parameter overrides
- `code_version` (git SHA if available)
- `status`: queued/running/succeeded/failed
- `artifacts` (paths/urls)

## Testing triggers (Trust Gates)
Buttons on `/testing` should create `test_run` rows:
- `test_run_id`
- `test_type`: `data_recon` | `pnl_recon` | `failure_suite`
- `requested_by`
- `status`
- `summary`
- `artifact_refs`

### Gate mapping
Each test updates gate statuses:
- Gate1: data recon
- Gate5: pnl recon
- Gate6: failure suite
- Gate7: flatten proof (derived from trade/position check)

All stored for analysis.

