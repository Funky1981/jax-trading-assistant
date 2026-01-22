# Pages & Layouts

## Page Composition Guidelines
- Pages are assemblies of components, not custom one-off UI.
- Each page must have a layout template, data container, and view.
- Navigation, app shell, and panels are reused across pages.

## Core Pages
### 1) Trading Dashboard
- Purpose: real-time overview of market conditions and portfolio.
- Components: Watchlist, MarketChart, OrderBook, PositionsTable, AlertsPanel.
- Layout: configurable grid with persistent layout storage.

### 2) Order Ticket
- Purpose: fast order placement with minimal latency.
- Components: OrderTicket, PriceInput, QuantityStepper, RiskGuard.
- Layout: fixed panel with keyboard shortcuts.

### 3) Blotter / Activity Log
- Purpose: audit trail and execution monitoring.
- Components: DataGrid, FilterBar, ExportActions.
- Layout: full-screen table with virtualization.

### 4) Portfolio & Risk
- Purpose: positions, PnL, exposure, and limits.
- Components: PositionsTable, RiskSummary, PnLIndicator, ExposureChart.
- Layout: split panes (summary + deep details).

### 5) Settings & Preferences
- Purpose: user customization (theme, layout, shortcuts).
- Components: FormSections, ToggleGroup, ThemePicker.

## Layout Templates
- AppShell: persistent nav, header, connection status.
- Grid Layout: used for dashboards; widgets are dockable.
- Split Pane: used for side-by-side data density.

## Implemented Pages
- Dashboard: `frontend/src/pages/DashboardPage.tsx`
- Order Ticket: `frontend/src/pages/OrderTicketPage.tsx`
- Blotter: `frontend/src/pages/BlotterPage.tsx`
- Portfolio: `frontend/src/pages/PortfolioPage.tsx`
- Settings: `frontend/src/pages/SettingsPage.tsx`

Routing is defined in `frontend/src/app/App.tsx` with `AppShell` as the shared layout.
