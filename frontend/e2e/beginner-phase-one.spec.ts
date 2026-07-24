import { expect, test } from '@playwright/test';

const overview = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
  genuineEvents: 6,
  syntheticEvents: 2,
  rejectedEvents: 3,
  deduplicatedEvents: 0,
  candidates: 2,
  approvals: 1,
  paperTickets: 1,
  pendingCheckpoints: 1,
  completedCheckpoints: 2,
  missingDataCheckpoints: 0,
  ambiguousCheckpoints: 0,
  checkedAt: '2026-07-21T10:00:00Z',
};

test.beforeEach(async ({ page }) => {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false }),
    }),
  );
  await page.route('**/api/v1/operator-evidence/overview', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(overview) }),
  );
  await page.route('**/api/v1/research/events/world-monitor/inbox**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [],
        counts: { total: 0, pending: 0, candidatesCreated: 0, rejected: 0, ignored: 0 },
      }),
    }),
  );
});

test('beginner journey preserves Review routes', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('status')).toContainText('Paper-safe mode is on');
  await page.getByRole('link', { name: 'Start the guide' }).click();
  await expect(
    page.getByRole('heading', { name: 'Start with the current Jax workflow' }),
  ).toBeVisible();
  await page.getByRole('link', { name: 'Evidence Inbox' }).first().click();
  await expect(page).toHaveURL(/\/monitor\/inbox$/);
  const review = page.getByRole('button', { name: 'Review' });
  await expect(review).toHaveAttribute('aria-expanded', 'false');
  await review.click();
  await page.getByRole('link', { name: 'Settings' }).click();
  await expect(page).toHaveURL(/\/settings$/);
  await expect(page.getByRole('note')).toContainText('not yet been redesigned');
});

for (const width of [320, 768, 1280]) {
  test(`navigation hierarchy works at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/');
    if (width < 768) await page.getByRole('button', { name: 'Open sidebar' }).click();
    const primary = page.getByLabel('Primary navigation');
    await expect(primary.getByRole('link')).toHaveCount(6);
    await expect(page.getByRole('button', { name: 'Review' })).toBeVisible();
    await expect(page.getByRole('status')).toBeVisible();
  });
}
