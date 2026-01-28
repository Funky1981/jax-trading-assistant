import { expect, test } from '@playwright/test';

test('loads dashboard and navigates to order ticket', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

  await page.getByRole('link', { name: 'Order Ticket' }).click();
  await expect(page.getByRole('heading', { name: 'Order Ticket' })).toBeVisible();
});
