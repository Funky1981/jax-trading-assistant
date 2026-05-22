import { expect, test } from '@playwright/test';

test('loads home and navigates to manual trading', async ({ page }) => {
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
  await expect(page.getByRole('heading', { name: 'Home' })).toBeVisible();

  await page.getByRole('link', { name: /Place a manual trade/i }).click();
  await expect(page).toHaveURL(/\/equity-alpha\/trading$/);
  await expect(page.locator('h1').filter({ hasText: 'Trading' })).toBeVisible();
});
