# Component Library Playbook

## Purpose
Define the design system and component library for a professional trading UI. Components must be stable, composable, and performance‑aware.

## Component Taxonomy
1. **Foundations**
   - Typography, colors, spacing, elevation, motion, icons.
2. **Primitives**
   - Button, Input, Select, Tabs, Tooltip, Popover, Badge.
3. **Data Display**
   - Table, DataGrid, Chart, Sparkline, OrderBook, MarketDepth.
4. **Trading Modules**
   - OrderTicket, PositionCard, PnLIndicator, RiskSummary.
5. **Layout & Containers**
   - AppShell, Grid, SplitPane, Panel, Drawer.
6. **Feedback & Status**
   - Alert, Toast, InlineStatus, ConnectionIndicator.

## Component Contract Guidelines
- **Props are explicit and typed** (no implicit global state).
- **Avoid side effects** inside components.
- **Prefer composition over configuration** for complex UIs.
- **Accessibility**
  - Keyboard navigation for all interactive elements.
  - Semantic roles for tables, charts, and panels.
  - Visible focus states.

## Variant Strategy
- Variants must be defined centrally (e.g., `Button` → `primary`, `secondary`, `danger`, `ghost`).
- Theming overrides must not leak into component internals.

## Performance Requirements
- **Tables:** use virtualization and windowing for order/position logs.
- **Charts:** decimate data points for high‑frequency feeds.
- **Order Book:** batch updates and prioritize top‑of‑book changes.

## Documentation Standards
Each component doc must include:
- Purpose
- Props and types
- Variants and theming guidance
- Accessibility notes
- Performance considerations
- Example usage

