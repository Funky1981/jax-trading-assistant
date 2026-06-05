# Create HomePage and Simplify Primary Nav

## Summary

- Priority: P0
- Phase: Phase 1
- Estimate: 2-3d
- Outcome: clear first-run IA

## Objective

Create a beginner-facing Home entry point and reduce primary navigation to task-first destinations instead of exposing internal modules and QA surfaces.

## In-scope touchpoints

- `frontend/src/app/App.tsx`
- `frontend/src/components/layout/AppShell.tsx`
- `frontend/src/pages/HomePage.tsx`

## Implementation notes

- Add `HomePage` as the first-route experience.
- Use one concise sentence to explain what Jax does.
- Provide three primary starting actions: find AI opportunities, place a manual trade, and test a strategy.
- Simplify top-level navigation to Home, AI Trading, Manual Trading, Approvals, Research, Analysis, Notifications, and Settings.
- Move operational or QA destinations such as System, Testing, Paper Trading Test Plan, Mobile Approval Harness, E2E Tests, static guides, and module internals behind Settings, Admin and QA, or contextual Learn surfaces.
- Preserve legacy route aliases where needed so existing links do not break.

## Acceptance criteria

- A new user sees Home first or can clearly navigate to it.
- Home explains the product without internal architecture terms.
- Primary navigation uses the new task-first structure.
- Removed top-level items remain reachable through Settings/Admin, QA, or contextual links.
- Existing deep links and redirects continue to work where current routes depend on them.

## Suggested validation

- Run targeted frontend tests for app routing and layout if present.
- Run affected e2e navigation coverage, especially `frontend/e2e/trading.spec.ts` if the navigation shell is covered there.
- Manually inspect desktop and mobile navigation states for overflow, hidden QA links, and route preservation.

## Dependencies

- None.
