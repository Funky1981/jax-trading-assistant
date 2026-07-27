import { expect, test, type Page } from '@playwright/test';

const overview = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
};

const genuineEvidence = {
  id: 'evidence-1',
  source: 'world-monitor',
  sourceEventId: 'source-event-1',
  worldMonitorEventId: 'world-event-1',
  status: 'researching',
  eventType: 'macro_rates',
  headline:
    'A long genuine evidence headline remains understandable at narrow mobile widths without clipping',
  summary: 'A persisted source described a rates event for research.',
  sourceUrls: ['https://example.com/genuine-evidence'],
  sourceCount: 1,
  eventTime: '2026-07-20T14:14:00Z',
  collectedAt: '2026-07-20T14:20:00Z',
  receivedAt: '2026-07-20T14:21:00Z',
  rawEventId: 'raw-1',
  isSynthetic: false,
  discoveryMethod: 'rss',
  analysisIdentity: 'deterministic-keywords-v1',
  possibleAffectedEtfs: [],
  assetThemes: [],
  severity: 'medium',
  sourceTier: 'tier1',
  confidence: 0.73,
  confidenceReasons: ['Configured deterministic rule matched.'],
  mappingReason: 'No truthful asset mapping was retained.',
  normalizedEventId: 'normalised-1',
  normalizedAt: '2026-07-20T14:21:01Z',
  outcomeCount: 0,
  rawPayload: { fixture: true },
};

async function mockRuntime(page: Page) {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false }),
    }),
  );
  await page.route('**/api/v1/operator-evidence/overview', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(overview) }),
  );
  await page.route('**/api/v1/research/events/world-monitor/inbox**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [genuineEvidence],
        total: 1,
        counts: {
          genuine: 1,
          syntheticTests: 0,
          rejected: 0,
          duplicates: 0,
          candidatesCreated: 0,
        },
        checkedAt: '2026-07-20T14:22:00Z',
      }),
    }),
  );
}

for (const width of [320, 768, 1280]) {
  test(`Evidence Inbox primary flow reflows without horizontal overflow at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 900 });
    await mockRuntime(page);
    await page.goto('/');

    if (width < 768) {
      await page.getByRole('button', { name: 'Open sidebar' }).click();
    }
    await page.getByRole('link', { name: 'Evidence Inbox' }).first().click();
    await expect(page).toHaveURL(/\/monitor\/inbox$/);
    await expect(page.getByRole('heading', { name: 'Evidence Inbox' })).toBeVisible();
    await expect(
      page.getByText(/Open an evidence item to see its source, timestamps/i),
    ).toHaveCount(0);
    await expect(page.getByText('1–1 of 1')).toBeVisible();

    await page.getByRole('button', { name: 'Genuine', exact: true }).click();
    await page.getByRole('button', { name: /long genuine evidence headline/i }).click();

    await expect(page.getByRole('link', { name: 'Open original source' })).toBeVisible();
    await expect(page.getByText('Published time')).toBeVisible();
    await expect(page.getByText('Collection time').first()).toBeVisible();
    await expect(page.getByText('Jax receipt time').first()).toBeVisible();
    await expect(page.getByText('Source and provenance').locator('..')).not.toHaveAttribute('open');
    await expect(page.getByText('Analysis', { exact: true }).locator('..')).not.toHaveAttribute(
      'open',
    );
    await expect(page.getByText(/Journey —/).locator('..')).not.toHaveAttribute('open');
    await expect(page.getByText('Audit').locator('..')).not.toHaveAttribute('open');

    await page.getByText('Analysis', { exact: true }).click();
    await expect(page.getByText('DETERMINISTIC ANALYSIS')).toBeVisible();
    await expect(page.getByText('No AI used')).toBeVisible();
    await expect(page.getByText('Unknown assets', { exact: true })).toBeVisible();
    await expect(
      page.getByText('No candidate was created. This is a valid outcome.', { exact: true }),
    ).toBeVisible();

    const audit = page.getByText('Audit', { exact: true });
    await audit.click();
    await expect(page.getByText('Source-event ID')).toBeVisible();
    await expect(page.getByText(/fixture/)).not.toBeVisible();

    const overflow = await page.evaluate(() => ({
      document: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      body: document.body.scrollWidth - document.body.clientWidth,
    }));
    expect(overflow.document).toBeLessThanOrEqual(1);
    expect(overflow.body).toBeLessThanOrEqual(1);
    const evidenceSection = page
      .getByRole('heading', { name: 'Evidence received' })
      .locator('..')
      .locator('..');
    expect(
      await evidenceSection
        .locator('[class*="sticky"], [class*="overflow-y"], [class*="h-screen"]')
        .count(),
    ).toBe(0);
    await expect(
      page.getByRole('button', { name: /long genuine evidence headline/i }),
    ).toBeVisible();
  });
}
