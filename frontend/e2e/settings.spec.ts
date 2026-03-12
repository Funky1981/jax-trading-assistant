/**
 * E2E: Settings page (/settings)
 *
 * User stories covered:
 *  US-SET-1  Settings page renders heading and all preference controls.
 *  US-SET-2  Compact layout checkbox is checked by default.
 *
 * SettingsPage is a static UI form (no backend calls). Auth and health stubs
 * are still registered for the protected route wrapper.
 */
import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

// ---------------------------------------------------------------------------
// US-SET-1: Settings page renders all controls
// ---------------------------------------------------------------------------
test('settings page: renders heading and preference controls', async ({ page }) => {
  await stubBase(page);

  await page.goto('/settings', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /Settings/i })).toBeVisible();
  await expect(page.getByText('Adjust basic preferences for the UI')).toBeVisible();
  // Theme selector, order size input, compact checkbox
  await expect(page.getByText('Default Order Size')).toBeVisible();
  await expect(page.locator('#compact')).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-SET-2: Compact layout checkbox default state
// ---------------------------------------------------------------------------
test('settings page: compact layout checkbox is checked by default', async ({ page }) => {
  await stubBase(page);

  await page.goto('/settings', { waitUntil: 'domcontentloaded' });

  await expect(page.locator('#compact')).toBeChecked();
});
