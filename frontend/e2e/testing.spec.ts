/**
 * E2E: Testing / Trust Gates page (/testing)
 *
 * User stories covered:
 *  US-TEST-1  Testing page loads with gate status table populated from API.
 *  US-TEST-2  "Run Config Integrity" button triggers POST to /api/v1/testing/config-integrity.
 *  US-TEST-3  "Run All Gates" button triggers POST to /api/v1/testing/run-all.
 *
 * Gate status endpoint: GET /api/v1/testing/status → TestingGateStatus[]
 * Test run history endpoint: GET /api/v1/test-runs?limit=50 → TestRunSummary[]
 * Trigger endpoints (POST): /api/v1/testing/config-integrity, /api/v1/testing/run-all, etc.
 */
import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

const GATE_STUB = {
  gate: 'config-integrity',
  status: 'passed',
  lastRunAt: '2026-03-06T12:00:00Z',
  updatedAt: '2026-03-06T12:00:05Z',
  details: {},
};

async function stubTestingRoutes(page: Parameters<typeof stubBase>[0]) {
  await stubBase(page);

  await page.route('**/api/v1/testing/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([GATE_STUB]),
    }),
  );

  await page.route('**/api/v1/test-runs**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    }),
  );
}

// ---------------------------------------------------------------------------
// US-TEST-1: Gate status table renders
// ---------------------------------------------------------------------------
test('testing page: loads with Trust Gates table and gate name', async ({ page }) => {
  await stubTestingRoutes(page);

  await page.goto('/testing', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /Testing/i })).toBeVisible();
  await expect(page.getByText('Trust Gates Checklist')).toBeVisible();
  // Gate name from stub must appear in the table.
  await expect(page.getByText('config-integrity')).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-TEST-2: Run Config Integrity gate
// ---------------------------------------------------------------------------
test('testing page: run config integrity button fires POST request', async ({ page }) => {
  await stubTestingRoutes(page);

  let configTriggers = 0;
  await page.route('**/api/v1/testing/config-integrity', (route) => {
    if (route.request().method() === 'POST') {
      configTriggers += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'triggered', message: 'Config integrity check queued' }),
      });
    }
    return route.continue();
  });

  await page.goto('/testing', { waitUntil: 'domcontentloaded' });

  await page.getByRole('button', { name: 'Run Config Integrity' }).click();

  await expect.poll(() => configTriggers).toBe(1);
});

// ---------------------------------------------------------------------------
// US-TEST-3: Run All Gates
// ---------------------------------------------------------------------------
test('testing page: run all gates button fires POST to run-all endpoint', async ({ page }) => {
  await stubTestingRoutes(page);

  let runAllTriggers = 0;
  await page.route('**/api/v1/testing/run-all', (route) => {
    if (route.request().method() === 'POST') {
      runAllTriggers += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'triggered', message: 'All gates queued' }),
      });
    }
    return route.continue();
  });

  await page.goto('/testing', { waitUntil: 'domcontentloaded' });

  await page.getByRole('button', { name: 'Run All Gates' }).click();

  await expect.poll(() => runAllTriggers).toBe(1);
});
