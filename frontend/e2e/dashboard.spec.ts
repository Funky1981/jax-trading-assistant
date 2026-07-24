import { expect, test } from '@playwright/test';

test('loads beginner Home and opens Evidence Inbox', async ({ page }) => {
  await page.route('**/auth/status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false }),
    });
  });
  await page.route('**/health', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy', uptime: 'stubbed' }),
    });
  });
  await page.route('**/api/v1/operator-evidence/overview', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        runtimeMode: 'paper',
        allowLiveTrading: false,
        executionEnabled: false,
        executionWorkerEnabled: false,
        brokerExecutionAllowed: false,
        maximumLeverage: 1,
        genuineEvents: 6,
        syntheticEvents: 0,
        rejectedEvents: 0,
        deduplicatedEvents: 0,
        candidates: 2,
        approvals: 1,
        paperTickets: 1,
        pendingCheckpoints: 1,
        completedCheckpoints: 2,
        missingDataCheckpoints: 0,
        ambiguousCheckpoints: 0,
        checkedAt: '2026-07-21T10:00:00Z',
      }),
    });
  });
  await page.route('**/api/v1/research/events/world-monitor/inbox**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [],
        counts: { total: 0, pending: 0, candidatesCreated: 0, rejected: 0, ignored: 0 },
      }),
    });
  });

  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Jax overview' })).toBeVisible();
  await expect(page.getByRole('status')).toContainText('Paper-safe mode is on');

  await page.getByRole('link', { name: 'Open Evidence Inbox' }).last().click();
  await expect(page).toHaveURL(/\/monitor\/inbox$/);
  await expect(page.locator('h1')).toContainText('Monitor Inbox');
});
