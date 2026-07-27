import { expect, test, type Page } from '@playwright/test';
import { stubBase } from './helpers';

const safeOverview = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
  genuineEvents: 1,
  syntheticEvents: 0,
  rejectedEvents: 0,
  deduplicatedEvents: 0,
  candidates: 1,
  approvals: 1,
  paperTickets: 1,
  pendingCheckpoints: 1,
  completedCheckpoints: 2,
  missingDataCheckpoints: 0,
  ambiguousCheckpoints: 0,
  historicalExecutionInstructions: 1,
  historicalOrderIntents: 0,
  historicalBrokerOrders: 0,
  historicalTrades: 0,
  historicalFills: 0,
  checkedAt: '2026-07-27T12:00:00Z',
};

async function stubSystem(page: Page) {
  await stubBase(page);
  await page.route('**/api/v1/operator-evidence/overview', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(safeOverview),
    }),
  );
  await page.route('**/api/v1/operator-evidence/candidates/qqq-proof', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        selectedExecutionCounts: {
          executionInstructions: 0,
          orderIntents: 0,
          brokerOrders: 0,
          trades: 0,
          fills: 0,
        },
      }),
    }),
  );
  await page.route('**/api/v1/events**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ events: [] }),
    }),
  );
  await page.route('**/api/v1/datasets**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ datasets: [] }),
    }),
  );
}

for (const width of [320, 768, 1280]) {
  test(`System Safety is paper-safe and responsive at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await stubSystem(page);
    await page.goto('/system?candidateId=qqq-proof');
    await expect(page.getByRole('heading', { level: 1, name: 'System Safety' })).toBeVisible();
    await expect(
      page.getByText(
        'Paper-safe mode is on. Live trading, execution and broker activity are disabled.',
      ),
    ).toBeVisible();
    await expect(page.getByText('Paper', { exact: true })).toBeVisible();
    await expect(page.getByText('Off', { exact: true })).toBeVisible();
    await expect(page.getByText('Disabled', { exact: true })).toBeVisible();
    await expect(page.getByText('Stopped', { exact: true })).toBeVisible();
    await expect(page.getByText('Not allowed', { exact: true })).toBeVisible();
    await expect(page.getByText('1x', { exact: true })).toBeVisible();
    await expect(page.getByText('This journey created no execution records.')).toBeVisible();
    await expect(page.getByText('Technical diagnostics').locator('..')).not.toHaveAttribute(
      'open',
      '',
    );
    await expect(
      page.getByRole('button', { name: /enable|disable|start|stop|delete|clear|restart/i }),
    ).toHaveCount(0);
    const overflow = await page.evaluate(() => ({
      document: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      body: document.body.scrollWidth - document.body.clientWidth,
    }));
    expect(overflow.document).toBeLessThanOrEqual(1);
    expect(overflow.body).toBeLessThanOrEqual(1);
  });
}
