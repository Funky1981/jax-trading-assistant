# Frontend Design System Foundation

The canonical runtime design-token values live in `frontend/src/index.css` as semantic CSS variables. Tailwind maps to those variables through `frontend/tailwind.config.js`, TypeScript helpers in `frontend/src/styles/tokens.ts` reference those variables, MUI theme values in `frontend/src/styles/theme.ts` reference the same variables, and Storybook imports the same `index.css` baseline.

Use shared components from `frontend/src/components/ui` and `frontend/src/components/primitives` before adding page-local styling. Redesign pages should use semantic tokens, compact operational layouts, icon buttons for tool actions, 8px-or-less card radius unless a shared primitive requires otherwise, no nested cards, and no decorative gradient or orb backgrounds.
