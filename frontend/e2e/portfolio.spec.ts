/**
 * E2E: Portfolio & Risk page (/portfolio)
 *
 * User stories covered:
 *  US-PORT-1  Portfolio page loads with positions and account data.
 *  US-PORT-2  Protect position — dialog opens, stop-loss confirmed, POST fires.
 *  US-PORT-3  Close position — dialog confirmation, POST fires.
 *
 * PositionsPanel and RiskSummaryPanel are shared components also used on
 * the /trading page. The full modal workflows are tested here via /portfolio
 * to verify the standalone portfolio route end-to-end.
 */
import { expect, test } from '@playwright/test';
import { stubAccount, stubBase, stubPositions } from './helpers';

// ---------------------------------------------------------------------------
// US-PORT-1: Portfolio page renders
// ---------------------------------------------------------------------------
test('portfolio page: shows heading and open positions', async ({ page }) => {
  await stubBase(page);
  await stubAccount(page);
  await stubPositions(page);

  await page.goto('/portfolio', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /Portfolio & Risk/i })).toBeVisible();
  // The SPY position from our stub must appear in the positions panel.
  await expect(page.getByText('SPY', { exact: true }).first()).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-PORT-2: Protect a position
// ---------------------------------------------------------------------------
test('portfolio page: protect position opens dialog and sends request', async ({ page }) => {
  await stubBase(page);
  await stubAccount(page);
  await stubPositions(page);

  let protectRequests = 0;
  await page.route('**/positions/*/protect', (route) => {
    protectRequests += 1;
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        order_ids: [3001, 3002],
        cancelled_order_ids: [],
        message: 'Submitted protection for SPY',
      }),
    });
  });

  await page.goto('/portfolio', { waitUntil: 'domcontentloaded' });

  // Open the protect dialog.
  await page.getByRole('button', { name: 'Protect' }).first().click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();

  // Fill stop-loss and confirm the pilot checklist.
  await dialog.getByLabel('Stop Loss').fill('650');
  await dialog
    .getByRole('checkbox', { name: /I confirmed these protective levels in IB\/TWS/i })
    .check();
  await dialog.getByRole('button', { name: 'Submit Protection' }).click();

  await expect.poll(() => protectRequests).toBe(1);
});

// ---------------------------------------------------------------------------
// US-PORT-3: Close a position
// ---------------------------------------------------------------------------
test('portfolio page: close position confirms and sends request', async ({ page }) => {
  await stubBase(page);
  await stubAccount(page);
  await stubPositions(page);

  let closeRequests = 0;
  await page.route('**/positions/*/close', (route) => {
    closeRequests += 1;
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        order_id: 3003,
        message: 'Close order submitted for SPY',
      }),
    });
  });

  await page.goto('/portfolio', { waitUntil: 'domcontentloaded' });

  await page.getByRole('button', { name: 'Close' }).first().click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toBeVisible();

  await dialog
    .getByRole('checkbox', { name: /I confirmed this exit in IB\/TWS/i })
    .check();
  await dialog.getByRole('button', { name: 'Submit Close' }).click();

  await expect.poll(() => closeRequests).toBe(1);
});
