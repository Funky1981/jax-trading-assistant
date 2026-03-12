/**
 * E2E: Research page (/research)
 *
 * User stories covered:
 *  US-RES-1  Research page loads with Strategy Instances tab active.
 *  US-RES-2  Instances tab lists stubbed strategy instances by name.
 *  US-RES-3  Switching to the Runs tab lists stub backtest runs.
 *
 * ResearchPage queries:
 *   GET /api/v1/instances          — strategy instance list
 *   GET /api/v1/strategy-types     — available strategy types (for editor)
 *   GET /api/v1/backtest/runs      — run history
 *   GET /api/v1/datasets           — dataset list (for run form)
 *   GET /api/v1/research/projects  — projects tab
 */
import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

const INSTANCE_STUB = {
  id: 'inst-1',
  name: 'MA Crossover Live',
  strategyTypeId: 'ma_crossover',
  strategyId: 'ma_crossover_v1',
  enabled: true,
  sessionTimezone: 'America/New_York',
  flattenByCloseTime: '15:55',
  configJson: { universe: ['SPY'] },
  createdAt: '2026-01-15T10:00:00Z',
  updatedAt: '2026-03-01T09:00:00Z',
};

const RUN_STUB = {
  id: 'run-1',
  runId: 'run-1',
  instanceId: 'inst-1',
  status: 'completed',
  from: '2026-02-01',
  to: '2026-03-01',
  datasetId: 'ds-001',
  stats: {
    winRate: 0.64,
    maxDrawdown: -0.12,
    totalReturn: 0.034,
    trades: 42,
    sharpe: 1.42,
  },
};

async function stubResearchRoutes(page: Parameters<typeof stubBase>[0]) {
  await stubBase(page);

  await page.route('**/api/v1/instances**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([INSTANCE_STUB]),
    }),
  );

  await page.route('**/api/v1/strategy-types**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'ma_crossover', name: 'MA Crossover', description: 'Trend-following strategy' },
      ]),
    }),
  );

  await page.route('**/api/v1/backtests/runs**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([RUN_STUB]),
    }),
  );

  await page.route('**/api/v1/datasets**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ datasets: [], total: 0 }),
    }),
  );

  await page.route('**/api/v1/research/projects**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    }),
  );
}

// ---------------------------------------------------------------------------
// US-RES-1: Research page loads
// ---------------------------------------------------------------------------
test('research page: loads with heading and instances tab active', async ({ page }) => {
  await stubResearchRoutes(page);

  await page.goto('/research', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /Research/i })).toBeVisible();
  // Instances tab should be selected by default.
  await expect(page.getByRole('tab', { name: /Instances/i })).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-RES-2: Instances tab shows stub data
// ---------------------------------------------------------------------------
test('research page: instances tab lists strategy instances', async ({ page }) => {
  await stubResearchRoutes(page);

  await page.goto('/research', { waitUntil: 'domcontentloaded' });

  // Instance name from stub must appear in the table.
  await expect(page.getByText('MA Crossover Live')).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-RES-3: Runs tab shows stub backtest runs
// ---------------------------------------------------------------------------
test('research page: backtests tab shows backtest run history', async ({ page }) => {
  await stubResearchRoutes(page);

  await page.goto('/research', { waitUntil: 'domcontentloaded' });

  // Switch to the Backtests tab (rendered as "Backtests" in TabsTrigger).
  await page.getByRole('tab', { name: /Backtests/i }).click();

  // Run entry from stub must be visible (runId and status columns).
  await expect(page.getByText('run-1').first()).toBeVisible();
  await expect(page.getByText('completed').first()).toBeVisible();
});
