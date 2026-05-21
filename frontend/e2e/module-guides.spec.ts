import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

test('equity alpha guide content and links smoke', async ({ page }) => {
  await stubBase(page);

  await page.goto('/equity-alpha/guide');
  await expect(page.getByRole('heading', { name: 'Equity Alpha Beginner Guide' })).toBeVisible();
  await expect(page.locator('main')).toContainText('Quick Start Checklist');
  await expect(page.locator('main')).toContainText('Glossary');

  await page.getByRole('link', { name: 'Open Equity Alpha Trading' }).click();
  await expect(page).toHaveURL(/\/equity-alpha\/trading$/);

  await page.goto('/equity-alpha/guide');
  await page.getByRole('link', { name: 'Open Equity Alpha Strategies' }).click();
  await expect(page).toHaveURL(/\/equity-alpha\/strategies$/);

  await page.goto('/equity-alpha/guide');
  await page.getByRole('link', { name: 'Open Equity Alpha Timeline' }).click();
  await expect(page).toHaveURL(/\/equity-alpha\/timeline$/);
});

test('etf guide route smoke', async ({ page }) => {
  await stubBase(page);

  await page.goto('/etf/guide');
  await expect(page.getByRole('heading', { name: 'ETF Beginner Guide' })).toBeVisible();
  await expect(page.locator('main')).toContainText('Quick Start Checklist');
});
