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
  await page.route('**/api/v1/trades**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        trades: [
          {
            id: 'trade-1',
            symbol: 'AAPL',
            direction: 'buy',
            type: 'market',
            quantity: 10,
            order_status: 'filled',
            created_at: '2026-03-05T12:00:00Z',
          },
        ],
      }),
    });
  });

  await page.goto('/blotter', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('heading', { level: 1, name: /Blotter/ })).toBeVisible();
  await expect(page.getByText('AAPL')).toBeVisible();
});
