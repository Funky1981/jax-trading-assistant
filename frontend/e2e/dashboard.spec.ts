import { expect, test } from '@playwright/test';

test('loads dashboard and navigates to order ticket', async ({ page }) => {
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

  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

  await page.getByRole('link', { name: 'Order Ticket' }).click();
  await expect(page.getByRole('heading', { name: 'Order Ticket', exact: true })).toBeVisible();
});
