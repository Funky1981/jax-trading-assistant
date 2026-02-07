# Step 06: Pages & Routing

## Objective

Compose full application pages from documented components and ensure navigation is consistent and fast.

## Actions

1. Implement routing
   - Define core routes: dashboard, order ticket, blotter, portfolio, settings.

2. Compose pages
   - Assemble pages from component library modules.
   - Use layout templates (AppShell, Grid, Split Pane).

3. Wire data containers
   - Data fetching and state management remains in containers.
   - Presentation components remain stateless.

## Deliverables
- `frontend/src/pages/` with core screens.
- `frontend/src/components/layout/AppShell.tsx` for shared layout.
- Updated documentation in `Docs/frontend/pages-and-layouts.md`.

## Exit Criteria
- All pages render using shared components.
- No page contains duplicated UI logic.
