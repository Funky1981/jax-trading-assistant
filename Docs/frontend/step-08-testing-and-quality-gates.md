# Step 08: Testing & Quality Gates

## Objective
Ensure every UI element and workflow has automated validation to prevent regressions and performance degradation.

## Actions
1. **Unit tests**
   - Components and domain logic with React Testing Library and Jest/Vitest.

2. **Integration tests**
   - Page flows like order placement and blotter updates.

3. **E2E tests**
   - Use Playwright or Cypress for critical trading workflows.

4. **Accessibility checks**
   - Automated checks with axe or equivalent.

5. **Performance tests**
   - Measure render cadence and memory under streaming data.

## Deliverables
- Unit tests: domain + component coverage in `frontend/src/**/__tests__/`.
- Integration tests: order ticket flow + blotter update harness in `frontend/src/tests/integration/`.
- Accessibility checks with `vitest-axe` in `frontend/src/tests/accessibility/`.
- Performance guardrails in `frontend/src/tests/performance/`.
- E2E Playwright suite in `frontend/e2e/` with `frontend/playwright.config.ts`.
- Scripts: `npm run test`, `npm run test:e2e` (run `npx playwright install --with-deps` once).

## Exit Criteria
- CI blocks merges on failing tests.
- Performance regressions are detected automatically.

