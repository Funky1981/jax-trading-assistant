import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

test('module navigation smoke: modules, equity alpha, etf', async ({ page }) => {
  await stubBase(page);

  await page.goto('/modules');
  await expect(page.getByRole('heading', { name: 'Trading Modules' })).toBeVisible();

  await page.getByRole('link', { name: 'Open Equity Alpha Module' }).click();
  await expect(page).toHaveURL(/\/equity-alpha\/trading$/);
  await expect(page.locator('main')).toContainText('Trading');

  await page.goto('/etf/trading');
  await expect(page).toHaveURL(/\/etf\/trading$/);
  await expect(page.locator('main')).toContainText('Trading');
});
