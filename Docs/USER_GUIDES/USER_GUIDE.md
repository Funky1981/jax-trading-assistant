# JAX Trading Assistant - User Guide

## Beginner operator navigation

The current primary workflow is:

`Home → Guide → Evidence Inbox → Candidates → Outcomes → System Safety`

- `Home` uses the authenticated operator-evidence overview to show runtime safety and persisted activity.
- `Evidence Inbox` keeps its existing `/monitor/inbox` route.
- `Candidates` keeps the compatible `/etf/approvals` route, but now presents a read-only candidate evidence list. The former mutation-oriented approval queue remains available at `/review/approvals`.
- `Outcomes` uses `/outcomes` to show persisted paper plans and their 1-hour, 1-day and 1-week checkpoint evidence. Values are hypothetical and are never fills or realised profit and loss.
- `System Safety` keeps its existing `/system` route.
- All other existing destinations and deep links remain available under the collapsed `Review` section. Review pages are explicitly marked as not yet redesigned.

Hypothetical paper outcomes are not orders, fills or realised profit and loss.

### Confirm system safety

1. Open `System Safety` and read the overall banner first. Safe means paper mode, live trading off, execution disabled, the execution worker stopped, broker execution not allowed, and maximum leverage no greater than `1x`.
   Local paper runtime requires `BROKER_EXECUTION_ALLOWED=false` and `MAX_LEVERAGE=1` in its ignored `.env`; the committed `.env.example` documents these non-secret values.
2. Treat every `Unknown` value as unconfirmed. Missing configuration is not shown as safe, false, or zero.
3. When arriving from Candidate Review or Outcomes, read `This journey` for execution instructions, order intents, broker orders, trades, and fills linked to that candidate.
4. Read `Historical records` separately. These database-wide totals may be unrelated to the selected journey; a historical execution instruction does not prove the selected candidate created one.
5. Open `Technical diagnostics` only when internal configuration names, health, datasets, metrics, logs, IDs, timestamps, or the raw read-only response are required.
6. Use `What does this mean?` or the Guide link for plain-language definitions. System Safety contains no controls that alter runtime or persisted evidence.

### Review evidence

1. Open `Evidence Inbox`.
2. Review ten compact records per page by default, or select twenty. Use `Previous` and `Next` to move between accurate result ranges; changing a filter safely returns to the first page.
3. Use `Genuine`, `Synthetic tests`, `Rejected`, or `Candidate created` to narrow the list.
4. Expand one evidence record inline. Opening another record closes the first.
5. Read the complete headline and primary summary, then open `Source and provenance` for the source, published time, collected time, and Jax receipt time.
6. Open `Analysis`. Deterministic rules are not described as AI; an AI provider or model appears only when persisted metadata proves it.
7. Open `Journey` to see what Jax did next. `Awaiting processing`, `Research only`, `Rejected`, `Duplicate ignored`, and `Candidate created` are distinct outcomes.
8. If a persisted candidate exists, use `Open Candidate Review`. Otherwise the record explains why no candidate link is available. The Evidence Inbox itself cannot approve or place a trade.
9. On `Candidate Review`, read the human decision, persisted paper-plan assumptions and selected-journey no-fill counts. Open `Audit details` only when record IDs or raw metadata are needed.
10. Open `Hypothetical Outcomes` to review checkpoint status, market-data provenance, hypothetical return and hypothetical P&L. Pending and missing-data states are not zero returns.
11. Open `Audit` only when technical IDs, provenance, or the nested collapsed raw payload are needed.

`Unknown assets` means no truthful persisted asset mapping exists. Jax does not insert a fallback symbol. A missing candidate is not automatically an error; check whether the evidence is awaiting a persisted decision or has a completed research-only outcome.

The compact Evidence Inbox refinement was manually accepted from operator-supplied desktop and mobile persisted-runtime screenshots. Automated responsive checks passed at 320, 768 and 1280 px. Codex did not independently capture the full six-screenshot refinement inventory. This refinement changed presentation only: all persisted technical evidence remains available through collapsed disclosures, and no mutation capability was added.

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
