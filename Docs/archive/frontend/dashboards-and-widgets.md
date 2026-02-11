# Dashboards & Widgets

## Dashboard Goals
- Fast, dense information display with zero input lag.
- Fully customizable layout with persistence.
- Safe refresh behavior for real-time trading data.

## Widget Registry
- Widgets are registered with metadata:
  - `id`, `title`, `minSize`, `defaultSize`, `dataNeeds`, `refreshPolicy`.
- Registry drives the dashboard builder and personalization UI.

Current registry lives in `frontend/src/features/dashboard/registry.ts`.

## Layout System
- Grid-based layout with widget coordinates and sizes.
- Persistence in local storage for now (versioned payloads).
- Presets for Trader, Risk, and Ops views.

Layout + persistence live in:
- `frontend/src/features/dashboard/layouts.ts`
- `frontend/src/features/dashboard/persistence.ts`

## Data Refresh Policy
- Streaming widgets (prices, order book): high-frequency, batched updates.
- Snapshot widgets (risk summary, allocation): periodic refresh with debounce.
- Manual refresh for expensive queries.

## Dashboard Examples
- Trader View: Watchlist, Order Ticket, Positions, Risk Summary.
- Risk View: Risk Summary, Positions, System Status.
- Ops View: System Status, Blotter, Watchlist.

## Performance Notes
- Avoid layout thrash: lock widget sizes during drag operations.
- Use virtualization for any widget with > 100 rows.
- Use memoized selectors for live data feeds.
