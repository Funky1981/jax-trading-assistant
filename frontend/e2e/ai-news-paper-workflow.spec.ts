import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

test('ai news paper workflow promotes an opportunity through approval into a paper instruction', async ({ page }) => {
  await stubBase(page);

  let approved = false;
  let approveRequests = 0;

  const headline = 'Softer inflation headline supports QQQ review.';
  const candidateId = 'candidate-ai-news-workflow-1';
  const detectedAt = '2026-06-11T13:00:00Z';

  const queueCandidate = {
    id: candidateId,
    symbol: 'QQQ',
    signalType: 'BUY',
    confidence: 0.74,
    entryPrice: 500,
    stopLoss: 492,
    takeProfit: 516,
    reasoning: `World Monitor: ${headline} Two trusted sources linked the macro event to growth ETFs.`,
    detectedAt,
    expiresAt: '2026-06-11T13:45:00Z',
    instanceName: 'etf-news-sector-momentum-paper-v1',
    metadata: {
      etfPolicy: {
        allowed: true,
        reasonCode: 'allowed',
        reason: 'QQQ is approved for ETF phase-1 paper trading.',
        catalogVersion: 'phase1-2026-05-13',
      },
      worldMonitor: {
        sourceEventId: 'wm-ai-news-workflow-1',
        headline,
        sourceCount: 2,
      },
    },
  };

  const approvedCandidate = {
    id: candidateId,
    strategyInstanceId: 'instance-etf-news',
    signalId: 'signal-ai-news-workflow-1',
    strategyId: 'etf_news_sector_momentum_v1',
    symbol: 'QQQ',
    signalType: 'BUY',
    status: 'approved',
    entryPrice: 500,
    stopLoss: 492,
    takeProfit: 516,
    confidence: 0.74,
    reasoning: queueCandidate.reasoning,
    sessionDate: '2026-06-11',
    detectedAt,
    dataProvenance: 'world-monitor',
    latestApproval: {
      id: 'approval-ai-news-workflow-1',
      decision: 'approved',
      approvedBy: 'admin',
      decidedAt: '2026-06-11T13:05:00Z',
    },
    executionInstructionId: 'instruction-ai-news-workflow-1',
  };

  await page.route('**/api/v1/ai/overview', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        checkedAt: '2026-06-11T13:01:00Z',
        scanner: {
          enabled: true,
          assetScope: 'etf',
          symbols: ['SPY', 'QQQ', 'IWM'],
          universePreset: 'etf-core',
          intervalSeconds: 300,
          minimumConfidence: 0.7,
          sentiment: {
            enabled: true,
            sourceScope: 'news',
            window: '24h',
            threshold: 0.6,
            minimumSourceCount: 2,
            sourceTrustWeightingMode: 'equal',
            mode: 'boost',
          },
          status: 'ready',
          channels: {
            inApp: true,
            desktopWeb: false,
            mobilePush: false,
          },
          policy: {
            manualRouteEnabled: true,
            approvalRouteEnabled: true,
            requiresHumanApproval: true,
          },
        },
        opportunityCounts: {
          signalsPending: 0,
          candidates: 0,
          approvals: approved ? 0 : 1,
        },
        policySummary: {
          requiresHumanApproval: true,
          manualRouteEnabled: true,
          approvalRouteEnabled: true,
        },
        channelSummary: {
          inApp: true,
          desktopWeb: false,
          mobilePush: false,
        },
      }),
    }),
  );

  await page.route('**/api/v1/signals**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ signals: [], total: 0, limit: 12, offset: 0 }),
    }),
  );

  await page.route('**/api/v1/approvals/queue**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(approved ? [] : [queueCandidate]),
    }),
  );

  await page.route('**/api/v1/approvals/*/approve', (route) => {
    approved = true;
    approveRequests += 1;
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        candidateId,
        latestApproval: approvedCandidate.latestApproval,
        execution: {
          id: 'instruction-ai-news-workflow-1',
          approvalId: 'approval-ai-news-workflow-1',
          candidateId,
          symbol: 'QQQ',
          signalType: 'BUY',
          entryPrice: 500,
          stopLoss: 492,
          takeProfit: 516,
          status: 'pending',
          createdAt: '2026-06-11T13:05:01Z',
          updatedAt: '2026-06-11T13:05:01Z',
        },
      }),
    });
  });

  await page.route('**/api/v1/candidates**', (route) => {
    const url = new URL(route.request().url());
    const status = url.searchParams.get('status');
    const rows = approved && status === 'approved' ? [approvedCandidate] : [];
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(rows),
    });
  });

  await page.goto('/ai-trading', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Find Trade Ideas' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Opportunity queue' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'QQQ' })).toBeVisible();
  await expect(page.getByText('Approval required')).toBeVisible();
  await expect(page.getByText('Medium confidence')).toBeVisible();
  await expect(page.getByText(headline)).toBeVisible();
  await expect(page.getByText('Approval is required before this opportunity can move to execution.')).toBeVisible();
  await expect(page.getByRole('link', { name: /Send to approval/i })).toBeVisible();

  await page.getByRole('link', { name: /Send to approval/i }).click();
  await expect(page).toHaveURL(/\/etf\/approvals$/);
  await expect(page.getByRole('heading', { name: 'Approval Queue' })).toBeVisible();
  await expect(page.getByText('QQQ', { exact: true })).toBeVisible();
  await expect(page.getByText('74%')).toBeVisible();
  await page.getByRole('button', { name: 'Show reasoning' }).click();
  await expect(page.getByText(headline)).toBeVisible();

  await page.getByRole('button', { name: /Approve for paper order/i }).click();
  await expect(page.getByText('Create paper instruction?')).toBeVisible();
  await page.getByRole('button', { name: /Yes, create paper order/i }).click();

  await expect.poll(() => approveRequests).toBe(1);
  await expect(page.getByText('Decision recorded: approve. Execution: pending')).toBeVisible();
  await expect(page.getByText('instruct...')).toBeVisible();
});
