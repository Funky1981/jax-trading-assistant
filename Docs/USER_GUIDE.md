# JAX Trading Assistant - User Guide

## Running A Backtest (UI)

1. Start the stack so `jax-trader` and `jax-research` are healthy.
2. Open the UI and go to `Research`.
3. Create or select a Strategy Instance.
   - Set `Strategy Type`, `Session Timezone`, and `Flatten By Close`.
   - Paste a JSON config in the editor and click `Save`.
4. Select a **Dataset Snapshot**.
   - Backtests in the research runtime require a dataset. The default dataset directory is `data/datasets` or the `DATASET_DIR` environment variable.
   - If no datasets appear, add a dataset snapshot and restart `jax-research`.
5. Set the date range and optional symbol override.
6. Click `Run` on the instance.
7. The run appears under `Runs`. Click a run to open `Analysis`.

## Screenshots

### Dataset Snapshots
![System dataset snapshots](../frontend/public/user-guide/system-datasets.png)

### Run a Backtest
![Research backtest](../frontend/public/user-guide/research-backtest.png)

### Review Runs and Analysis
![Backtest runs](../frontend/public/user-guide/backtest-runs.png)

![Analysis run](../frontend/public/user-guide/analysis-run.png)

## Analysis Page

- Review metrics, trades, dataset provenance, and the run timeline.
- Use `CSV` export to download trade history.
- The `Events` section can classify news or macro events for run context.

## Projects (Sweeps / Walk-Forward)

1. Create a project with a parameter grid and training/testing dates.
2. Select the project and click `Run`.
3. Project runs appear in `Project Runs` and as backtest runs under `Runs`.

## Testing / Trust Gates

- Use `Testing` to trigger paper-mode diagnostics.
- Each gate produces an artifact report under `/reports/<gate>/<date>/...`.

## Assistant Chat

The `Assistant` page is advisory-only research support. It cannot approve candidates, place trades, or execute orders.

- Use the chat input for broad questions about candidates, signals, strategies, and runs.
- Use the tool picker when you want a specific lookup or summary.
- Assistant answers can show an evidence badge:
  - `High evidence`: backed by hard internal data.
  - `Mixed evidence`: backed by derived internal summaries.
  - `Weak evidence`: limited support; language should remain more uncertain.
- Assistant answers can also show a trace link. Open it to inspect:
  - tool calls
  - tool arguments/results
  - validation attempts
  - the final audited answer

Mode notes:

- In `research` mode, the broadest set of read-only assistant tools is available.
- In `paper` and `live` modes, some tools may be disabled to enforce stricter evidence policy.

## Troubleshooting

- **Backtest returns 400 or fails**:
  - Ensure a dataset snapshot is selected.
  - Confirm `BACKTEST_DATASET_ID` or `DATASET_DIR` is set correctly for `jax-research`.
  - Verify dataset integrity in `System` -> `Dataset Snapshots`.

- **No datasets show up**:
  - Place dataset snapshots under `data/datasets`.
  - Restart `jax-research` to reload the catalog.
