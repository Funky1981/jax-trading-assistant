# Step 01: Foundation & Repo Setup

## Objective
Establish a clean React frontend foundation with tooling, conventions, and a predictable structure that supports fast iteration and professional trading‑grade reliability.

## Actions
1. **Create the frontend workspace**
   - Add a new `frontend/` directory at the repo root (or a dedicated `apps/frontend/` if the repo uses a monorepo layout).
   - Define an initial `src/` structure based on the architecture document.

2. **Select the component library + design system approach**
   - Choose a component library (e.g., MUI, Chakra, or a custom design system) and confirm it supports theming, accessibility, and high‑density layouts.
   - Record the choice in `Docs/docs/frontend/component-library-playbook.md` and align tokens in `Docs/docs/frontend/styling-and-theming.md`.

3. **Setup TypeScript & linting**
   - Configure TypeScript for strict mode.
   - Add ESLint + Prettier with consistent rules.
   - Include lint and format scripts in the package manager.

4. **Define repository conventions**
   - Confirm naming conventions for components, hooks, files, and folders.
   - Specify commit message standards and review checklist.

5. **Baseline CI**
   - Add CI pipeline steps for linting, tests, and type checks.

## Deliverables
- `frontend/` (or `apps/frontend/`) skeleton folder.
- TypeScript configuration and linting config.
- CI pipeline entries for lint and type checks.

## Exit Criteria
- Frontend compiles locally.
- Linting + type checks run in CI.

