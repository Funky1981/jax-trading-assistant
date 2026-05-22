# Design System Foundation for Redesign

## Summary

- Priority: P0
- Phase: Phase 0
- Estimate: 2-4d
- Outcome: consistent redesign foundation

## Objective

Create a reliable frontend design-system foundation before building the AI-ready redesign screens, so tokens, shared components, and Storybook all represent one coherent product language.

## In-scope touchpoints

- `frontend/tailwind.config.js`
- `frontend/src/index.css`
- `frontend/src/styles/tokens.ts`
- `frontend/src/styles/tokens.css`
- `frontend/src/styles/theme.ts`
- `frontend/.storybook/main.cjs`
- `frontend/.storybook/preview.tsx`
- `frontend/package.json`
- `frontend/src/components/ui/*`
- `frontend/src/components/primitives/*`

## Current-state findings

- Tailwind semantic colors are mapped to CSS variables in `tailwind.config.js`.
- Runtime CSS variables are defined in `frontend/src/index.css`.
- A separate token object exists in `frontend/src/styles/tokens.ts`.
- A separate `tokens.css` file defines another variable set and is used by Storybook.
- MUI theme support exists in `frontend/src/styles/theme.ts`, but the main app is not wrapped in MUI `ThemeProvider`.
- Storybook config exists, but package scripts and installed Storybook dependencies are missing from `frontend/package.json`.
- Shared components exist in `components/ui` and `components/primitives`, but only a small subset has stories.

## Implementation notes

- Pick one source of truth for semantic tokens and make Tailwind, app CSS, MUI theme, and Storybook consume it consistently.
- Keep the redesign token set operational and trading-focused: background, foreground, surface, border, muted, primary, accent, success, warning, destructive, chart up/down, risk, info, focus ring, spacing, radius, typography, and elevation.
- Align dark and light mode semantics across `index.css`, `tokens.ts`, and Storybook preview.
- Decide whether MUI is retained as a supported styling layer. If retained, wrap the app and Storybook with the same `ThemeProvider` and `CssBaseline`; if not, remove unused MUI theme dependence from redesign work.
- Add Storybook scripts and dependencies for the configured `@storybook/react-vite` setup.
- Add stories for all shared UI primitives that will be used by the redesign: button, badge, card, dialog, input, select, tabs, table, progress, skeleton, help hint, empty state, loading card, status card, and form controls.
- Add a token showcase story or docs page showing color, typography, spacing, radius, elevation, and state colors.
- Replace page-level hardcoded one-off colors in redesign-adjacent screens with semantic tokens or shared component variants.
- Document component usage rules for the redesign: icon buttons for tool actions, restrained card radius, dense operational layouts, no nested cards, and no decorative gradient/orb backgrounds.

## Acceptance criteria

- There is one documented source of truth for design tokens.
- Tailwind utilities, CSS variables, Storybook, and any retained MUI theme produce matching colors, typography, radius, and spacing.
- Storybook can be launched from `frontend/package.json`.
- Shared UI primitives used by the redesign have stories with normal, disabled, loading or empty, destructive/warning/success, and compact states where applicable.
- New redesign pages can be built without hardcoded hex colors or repeated one-off component styling.
- The app and Storybook render the same dark-mode baseline.
- Accessibility checks are available in Storybook for core primitives.

## Suggested validation

- Run `npm install` in `frontend` if Storybook dependencies are added.
- Run `npm run typecheck`.
- Run `npm run lint`.
- Run `npm run storybook` or the added Storybook dev script and inspect the token showcase plus core component stories.
- Run affected Vitest coverage for `frontend/src/styles/__tests__/theme.test.ts` and any new component tests.
- Manually inspect desktop and mobile examples for text fit, focus states, contrast, and component consistency.

## Dependencies

- None.

## Blocks

- `01-homepage-simplified-nav.md`
- `02-ai-trading-opportunity-feed.md`
- `04-scanner-sentiment-controls.md`
- `06-notification-centre-inbox.md`
- `07-research-wizard-v1.md`
- `12-opportunity-drawer-sentiment-explanation.md`
