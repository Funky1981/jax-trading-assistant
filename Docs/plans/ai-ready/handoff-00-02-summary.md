# AI-ready redesign handoff: tickets 00-02

## Branch and commits

- Branch: `redesign`
- Completed commits:
  - `b4a742a feat(frontend): add redesign design-system foundation`
  - `341b0f8 feat(frontend): add home page and simplify navigation`
  - `27de108 feat(frontend): add ai trading opportunity feed`
- Current working tree also contains uncommitted cleanup fixes for IDE Problems tab items.

## Ticket 00: Design system foundation

Implemented the redesign foundation before feature work:

- Added centralized styling/design-system documentation and tests.
- Confirmed app styling is driven from shared CSS tokens in `frontend/src/index.css`.
- Added MUI theme integration through the app provider stack.
- Added Storybook setup, scripts, preview styling, and baseline component stories.
- Added design-system verification coverage.
- Ran `npm audit fix` without force; remaining advisories require breaking upgrades.

Validation passed for ticket 00:

- `npm run typecheck`
- `npm run lint`
- `npm run test`
- `npm run build`
- `npm run test:e2e`
- `npm run build-storybook`

## Ticket 01: Home page and simplified navigation

Implemented a task-first first-run experience:

- Added `frontend/src/pages/HomePage.tsx`.
- Root route `/` now renders Home instead of Dashboard.
- Preserved Dashboard at `/dashboard`.
- Added route aliases:
  - `/ai-trading`
  - `/manual-trading`
  - `/notifications`
- Simplified primary navigation to:
  - Home
  - AI Trading
  - Manual Trading
  - Approvals
  - Research
  - Analysis
  - Notifications
  - Settings
- Moved operational/legacy destinations into collapsed Advanced groups.
- Kept legacy module routes and deep links working.
- Updated unit and Playwright tests for the new home/navigation behavior.
- Checked for redundant nav leftovers; old nav arrays were removed, while legacy pages remain intentionally reachable.

Validation passed for ticket 01:

- Targeted route/nav Vitest tests
- `npm run typecheck`
- `npm run lint`
- affected Playwright route/nav specs
- `npm run test`
- `npm run build`
- `npm run test:e2e`
- `npm run build-storybook`

## Ticket 02: AI Trading opportunity feed

Implemented a real dedicated AI Trading page:

- Replaced `/ai-trading -> /modules` redirect with `AiTradingPage`.
- Added `OpportunitySummary` read model types in `frontend/src/data/types.ts`.
- Added `frontend/src/data/opportunity-adapter.ts` to normalize:
  - signals
  - candidate trades
  - approval queue items
- Added deterministic adapter tests with signal, candidate, approval, partial/blocked fixtures.
- Added `frontend/src/pages/AiTradingPage.tsx` with:
  - scanner status
  - unified Opportunity queue
  - loading state
  - empty state
  - partial error state
  - stale-data state
  - visible expiry/status
  - route-aware actions: review order, send to approval, watch, dismiss, blocked-state guidance
- Added Playwright coverage for opening AI Trading from the shell and seeing route-aware opportunities.
- Cleaned one fragile approval-page test mock that could return `undefined` after query invalidation.

Validation passed for ticket 02:

- `npm run typecheck`
- `npm run lint`
- `npm run test` with 32 files / 60 tests
- `npm run build`
- `npm run test:e2e` with 32 tests passing
- `npm run build-storybook`

## Problems-tab cleanup after ticket 02

The user pasted IDE Problems entries. Cleanup fixes have been applied but are not yet committed:

- Added missing `limit_price` fields to e2e order fixtures.
- Split non-component exports out of component files to satisfy `react-refresh/only-export-components`.
- Fixed ARIA values for `aria-expanded` and `aria-pressed`.
- Replaced inline style usages flagged by Edge Tools.
- Cleaned markdownlint formatting issues in `Docs/deep-research-report.md`.
- Fixed Go analyzer findings:
  - tautological nilness condition
  - unused parameters
  - staticcheck switch suggestion

Validation after cleanup:

- `npm run typecheck`
- `npm run lint`
- `npm run test`
- `npm run build`
- `npm run test:e2e`
- `go test ./internal/modules/execution ./cmd/trader`
- `go vet ./cmd/trader ./internal/modules/execution`
- `markdownlint-cli2` on `Docs/deep-research-report.md` with `MD013` disabled

## Next recommended step

Review and commit the Problems-tab cleanup, then continue with ticket 03 or reconcile ticket 03 with the adapter work already introduced during ticket 02.

---

## Update: tickets 03-05 continuation on `redesign`

Follow-up commits now completed on `redesign`:

- `6577b16 chore: address IDE problems cleanup`
- `00e1232 feat(frontend): harden opportunity adapter`
- `708cf50 feat(frontend): add scanner sentiment controls`
- `055af14 feat(frontend): add manual ETF policy reroute card`

### Ticket 05 scope completed

- Updated `frontend/src/components/dashboard/OrderTicketPanel.tsx` to classify manual entry routing for ETF symbols as:
  - `manual_allowed`
  - `approval_required`
  - `blocked`
- Replaced the old dead-end ETF message with route-aware reroute cards:
  - approval-required CTA: Open approval flow
  - blocked CTA set: Choose another symbol and Open approved ETF workflow
- Preserved submit blocking and existing backend/pilot guardrails.
- Updated targeted tests in `frontend/src/components/dashboard/OrderTicketPanel.test.tsx` for manual-allowed, approval-required, and blocked states.
- Updated Playwright trading coverage in `frontend/e2e/trading.spec.ts` for approval-reroute assertions.

### Validation run (Ticket 05)

- `npm run test -- src/components/dashboard/OrderTicketPanel.test.tsx`
- `npm run typecheck`
- `npm run lint`
- `npm run build`
- `npm run test:e2e -- e2e/trading.spec.ts e2e/dashboard.spec.ts`

Result: all commands passed. Frontend build still reports the existing Vite large chunk warning.

---

## Update: tickets 06-07 continuation on `redesign`

Follow-up commits completed on `redesign`:

- `04a92b7 feat(frontend): add notification centre inbox v1`

### Ticket 06 scope completed

- Added `NotificationCentrePage` as the `/notifications` destination.
- Implemented durable in-app inbox semantics in phase 1:
  - unread/read state with persisted read markers
  - event type, title, body, destination route, created time, and delivery channel labels
  - stale badges for old notifications
- Added notification event adapter and typed models:
  - `frontend/src/data/notification-adapter.ts`
  - `frontend/src/data/types.ts` additions
- Wired route and routing tests:
  - `/notifications` now renders Notification Centre
- Added targeted tests for adapter/page/routing.

Validation run (Ticket 06):

- `npm run test -- src/data/notification-adapter.test.ts src/pages/NotificationCentrePage.test.tsx src/app/__tests__/AppRoutes.test.tsx`
- `npm run typecheck`
- `npm run lint`
- `npm run build`

### Ticket 07 scope completed

- Added Guided Research Wizard V1 to `frontend/src/pages/ResearchPage.tsx` with beginner-first steps:
  - strategy template
  - market scope
  - period selection
  - sentiment step visible but disabled for phase 1
- Kept existing advanced tabs and raw controls (Instances / Projects / Backtests) intact.
- Added missing-data UX in product language with operator path guidance.
- Wizard output launches existing backtest flow by building the backend-compatible run payload.
- Added helper logic and tests:
  - `frontend/src/pages/research-guided-wizard.ts`
  - `frontend/src/pages/research-guided-wizard.test.ts`
  - `frontend/src/pages/ResearchPage.test.tsx`

Validation run (Ticket 07 and notifications e2e coverage):

- `npm run test -- src/pages/research-guided-wizard.test.ts src/pages/ResearchPage.test.tsx src/data/notification-adapter.test.ts src/pages/NotificationCentrePage.test.tsx src/app/__tests__/AppRoutes.test.tsx`
- `npm run test:e2e -- e2e/notifications.spec.ts`
- `npm run typecheck`
- `npm run lint`
- `npm run build`

Result: commands passed; frontend build retains the existing Vite large chunk warning.
