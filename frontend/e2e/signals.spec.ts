/**
 * E2E: Signals Queue panel (rendered on the Dashboard)
 *
 * User stories covered:
 *  US-SIG-1  Signals queue renders pending signals with symbol, type, and status.
 *  US-SIG-2  Approve Trade posts to /api/v1/signals/:id/approve.
 *  US-SIG-3  Reject posts to /api/v1/signals/:id/reject.
 *
 * The SignalsQueuePanel is embedded in DashboardPage.
 * Route strategy: register mutation-specific routes AFTER broad wildcards so
 * Playwright's last-registered-first evaluation gives them priority.
 */
import { expect, test } from '@playwright/test';
import { stubBase, stubRecommendations, stubSignals } from './helpers';

const SIGNAL_STUB = {
  id: 'signal-1',
  symbol: 'AAPL',
  strategy_id: 'ma_crossover_v1',
  signal_type: 'BUY',
  confidence: 0.87,
  entry_price: 185.5,
  stop_loss: 179.04,
  take_profit: 192.89,
  status: 'pending',
  generated_at: '2026-03-06T13:10:00Z',
  created_at: '2026-03-06T13:10:00Z',
};

// ---------------------------------------------------------------------------
// US-SIG-1: Signals queue renders
// ---------------------------------------------------------------------------
test('signals queue: shows pending signal with symbol and type', async ({ page }) => {
  await stubBase(page);
  await stubSignals(page);
  await stubRecommendations(page);

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  // Signal card must show symbol and badge
  await expect(page.getByText('AAPL').first()).toBeVisible();
  await expect(page.getByText('BUY').first()).toBeVisible();
  // Status badge
  await expect(page.getByText('pending').first()).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-SIG-2: Approve Trade
// ---------------------------------------------------------------------------
test('signals queue: approve trade posts to approve endpoint', async ({ page }) => {
  await stubBase(page);
  await stubRecommendations(page);

  let approveRequests = 0;

  // Register before broad wildcard so it wins in last-registered-first order.
  await page.route('**/api/v1/signals**', (route) => {
    const method = route.request().method();
    const url = route.request().url();

    if (method === 'POST' && url.includes('/approve')) {
      approveRequests += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    }

    // GET — return signal list
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ signals: [SIGNAL_STUB], total: 1 }),
    });
  });

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  // Default approver is 'dashboard@local' so the button is enabled.
  await expect(page.getByRole('button', { name: 'Approve Trade' })).toBeVisible();
  await page.getByRole('button', { name: 'Approve Trade' }).click();

  await expect.poll(() => approveRequests).toBe(1);
});

// ---------------------------------------------------------------------------
// US-SIG-3: Reject
// ---------------------------------------------------------------------------
test('signals queue: reject posts to reject endpoint', async ({ page }) => {
  await stubBase(page);
  await stubRecommendations(page);

  let rejectRequests = 0;

  await page.route('**/api/v1/signals**', (route) => {
    const method = route.request().method();
    const url = route.request().url();

    if (method === 'POST' && url.includes('/reject')) {
      rejectRequests += 1;
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    }

    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ signals: [SIGNAL_STUB], total: 1 }),
    });
  });

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('button', { name: 'Reject' })).toBeVisible();
  await page.getByRole('button', { name: 'Reject' }).click();

  await expect.poll(() => rejectRequests).toBe(1);
});
