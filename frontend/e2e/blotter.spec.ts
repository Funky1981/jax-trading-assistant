import { expect, test } from '@playwright/test';

test('shows blotter rows', async ({ page }) => {
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

  await page.goto('/blotter', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { name: 'Blotter' })).toBeVisible();
  await expect(page.getByText('AAPL')).toBeVisible();
});
