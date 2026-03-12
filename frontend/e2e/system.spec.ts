/**
 * E2E: System page — datasets and events tables (/system)
 *
 * User stories covered:
 *  US-SYS-1  Dataset snapshots table shows stubbed snapshot entries.
 *  US-SYS-2  System events table shows stubbed event entries.
 *
 * Note: Pilot-status and health checks on the system page are already covered
 * in trading.spec.ts via installTradingStubs. These tests focus on the
 * datasets and events panels that are not yet tested.
 */
import { expect, test } from '@playwright/test';
import { stubBase, stubPilotStatus } from './helpers';

async function stubSystemRoutes(page: Parameters<typeof stubBase>[0]) {
  await stubBase(page);
  await stubPilotStatus(page);

  await page.route('**/api/v1/system/market-data-status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        connected: true,
        marketDataMode: 'delayed',
        paperTrading: true,
        checkedAt: '2026-03-06T13:15:00Z',
      }),
    }),
  );

  await page.route('**/api/v1/datasets**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        datasets: [
          {
            datasetId: 'ds-001',
            name: 'SPY 15m sample',
            symbol: 'SPY',
            datasetHash: 'abc123',
          },
        ],
        total: 1,
      }),
    }),
  );

  await page.route('**/api/v1/events**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        events: [
          {
            id: 'evt-1',
            kind: 'market_news',
            primarySymbol: 'SPY',
            title: 'Fed holds rates steady',
            eventTime: '2026-03-06T14:00:00Z',
          },
        ],
        total: 1,
      }),
    }),
  );

  await page.route('**/api/v1/metrics**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ metrics: [], total: 0 }),
    }),
  );
}

// ---------------------------------------------------------------------------
// US-SYS-1: Dataset snapshots table
// ---------------------------------------------------------------------------
test('system page: dataset snapshots table shows stub entry', async ({ page }) => {
  await stubSystemRoutes(page);

  await page.goto('/system', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /System/i })).toBeVisible();
  await expect(page.getByText('SPY 15m sample')).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-SYS-2: System events table
// ---------------------------------------------------------------------------
test('system page: events table shows stub event title', async ({ page }) => {
  await stubSystemRoutes(page);

  await page.goto('/system', { waitUntil: 'domcontentloaded' });

  await expect(page.getByText('Fed holds rates steady')).toBeVisible();
});
