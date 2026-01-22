# Dashboards & Widgets

## Dashboard Goals
- Fast, dense information display with zero input lag.
- Fully customizable layout with persistence.
- Safe refresh behavior for real‑time trading data.

## Widget Registry
- Widgets are registered with metadata:
  - `id`, `title`, `minSize`, `defaultSize`, `dataNeeds`, `refreshPolicy`.
- Registry drives the dashboard builder and personalization UI.

## Layout System
- **Grid‑based** drag and drop layout.
- **Persistence:** save per user + per workspace.
- **Versioning:** layout schema version to support upgrades.

## Data Refresh Policy
- **Streaming widgets** (prices, order book): high‑frequency, batched updates.
- **Snapshot widgets** (risk summary, allocation): periodic refresh with debounce.
- **Manual refresh** for expensive queries.

## Dashboard Examples
- **Trader View:** Watchlist, Order Ticket, Level II, Time & Sales, Positions.
- **Risk View:** Exposure, Limits, PnL Heatmap, Alerts.
- **Ops View:** System status, latency metrics, error logs.

## Performance Notes
- Avoid layout thrash: lock widget sizes during drag operations.
- Use virtualization for any widget with > 100 rows.
- Use memoized selectors for live data feeds.

