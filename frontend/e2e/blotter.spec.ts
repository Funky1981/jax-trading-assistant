import { expect, test } from '@playwright/test';

test('shows blotter rows', async ({ page }) => {
  await page.goto('/blotter');
  await expect(page.getByRole('heading', { name: 'Blotter' })).toBeVisible();
  await expect(page.getByText('AAPL')).toBeVisible();
});
