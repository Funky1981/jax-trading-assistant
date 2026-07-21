import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

test('candidate evidence page shows monitor news, chart evidence, sentiment, and sizing', async ({ page }) => {
  await stubBase(page);

  await page.route('**/api/v1/candidates/candidate-evidence-1', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'candidate-evidence-1',
        strategyInstanceId: 'instance-1',
        signalId: 'signal-1',
        strategyId: 'etf_news_sector_momentum_v1',
        symbol: 'SPY',
        signalType: 'BUY',
        status: 'awaiting_approval',
        entryPrice: 530,
        stopLoss: 527.5,
        takeProfit: 536,
        confidence: 0.84,
        reasoning: 'Softer inflation news supports a tactical SPY paper long while price holds above trend.',
        sessionDate: '2026-06-12',
        detectedAt: '2026-06-12T10:30:00Z',
        expiresAt: '2026-06-12T11:00:00Z',
        dataProvenance: 'world-monitor',
        metadata: {
          worldMonitor: {
            source: 'world-monitor',
            sourceEventId: 'wm-1',
            headline: 'Inflation cools more than expected',
            summary: 'Treasury yields moved lower after the inflation print.',
            eventType: 'macro_rates',
            sourceURLs: ['https://example.com/inflation', 'https://example.com/yields'],
            sourceCount: 2,
            assetThemes: ['rates', 'growth'],
            confidenceReasons: ['trusted macro source', 'mapped to SPY'],
            mappingReason: 'SPY was selected because broad US equities often react to lower-rate surprises.',
            route: 'approval_required',
          },
          chartConfirmation: {
            confirmed: true,
            reasonCode: 'above_sma20',
            reason: 'SPY held above the 20-period moving average and the last five candles were positive.',
            candleCount: 30,
            lastClose: 531.25,
            sma20: 528.1,
            fiveCandleChangePct: 0.012,
            checkedAt: '2026-06-12T10:31:00Z',
          },
          sizing: {
            model: 'paper_fixed_risk_v1',
            status: 'available',
            riskBudget: 100,
            shares: 40,
            quantity: 40,
            notional: 21200,
            riskToStop: 100,
            rewardToTarget: 240,
            riskReward: 2.4,
            source: 'world-monitor-promoter',
          },
        },
        sentiment: {
          label: 'positive',
          state: 'available',
          score: 0.68,
          confidence: 0.74,
          window: '24h',
          sourceCount: 2,
          priceAgreement: 'agreeing',
          topDrivers: ['Lower yields supported equity risk appetite.'],
          limitations: ['Only two trusted sources were available.'],
          summary: 'News tone is positive for broad US equity exposure.',
          snapshotAt: '2026-06-12T10:32:00Z',
        },
      }),
    }),
  );

  await page.route('**/api/v1/operator-evidence/candidates/candidate-evidence-1', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        evidenceScore: 0.85,
        evidenceStatus: 'sufficient',
        gateStatus: 'ready_for_risk_review',
        riskStatus: 'ready_for_approval_review',
        approvalId: 'approval-1',
        approvalDecision: 'approved',
        approvedBy: 'operator',
        approvalReason: 'Approved for retrospective paper evidence.',
        approvalAt: '2026-06-12T10:40:00Z',
        paperTicketId: 'paper-1',
        paperTicketStatus: 'paper_ticket_created',
        entry: 530,
        stop: 527.5,
        target: 536,
        quantity: 40,
        notional: 21200,
        plannedRisk: 100,
        plannedReward: 240,
        rewardRisk: 2.4,
        checkpoints: [{
          name: '1h', trackingStartedAt: '2026-06-12T10:40:00Z', trackingStartSource: 'paper_ticket_created_at',
          dueAt: '2026-06-12T11:40:00Z', observationAt: '2026-06-12T11:45:00Z', entryPrice: 530,
          checkpointPrice: 532, percentageReturn: 0.377, hypotheticalPnl: 80, maximumFavourableExcursion: 3,
          maximumAdverseExcursion: 1, targetTouched: false, stopTouched: false, status: 'completed',
          dataQualityStatus: 'complete_candle_window', marketDataSource: 'persisted_candles:test_fixture',
          createdAt: '2026-06-12T10:40:00Z', updatedAt: '2026-06-12T11:45:00Z',
        }],
        selectedExecutionCounts: { executionInstructions: 0, orderIntents: 0, brokerOrders: 0, trades: 0, fills: 0 },
        historicalExecutionCounts: { executionInstructions: 2, orderIntents: 1, brokerOrders: 0, trades: 0, fills: 0 },
      }),
    }),
  );

  await page.goto('/candidates/candidate-evidence-1/evidence', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'SPY trade setup' })).toBeVisible();
  await expect(page.getByRole('heading', { name: /Overview of the trade and why/i })).toBeVisible();
  await expect(page.getByText('Inflation cools more than expected')).toBeVisible();
  await expect(page.getByRole('link', { name: /example.com\/inflation/i })).toBeVisible();
  await expect(page.getByRole('heading', { name: /What the charts are saying/i })).toBeVisible();
  await expect(page.getByText('Chart confirmed')).toBeVisible();
  await expect(page.getByText(/held above the 20-period moving average/i)).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Sentiment and source' })).toBeVisible();
  await expect(page.getByText('News tone is positive for broad US equity exposure.')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Persisted paper sizing evidence' })).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Paper ticket — hypothetical only' })).toBeVisible();
  await expect(page.getByText('$21,200.00', { exact: true })).toBeVisible();
  await expect(page.getByText('HYPOTHETICAL — NOT A FILL', { exact: true })).toBeVisible();
  await expect(page.getByText('persisted_candles:test_fixture')).toBeVisible();
  await expect(page.getByText('NO FILL OCCURRED', { exact: true })).toBeVisible();
  await expect(page.getByText(/Historical execution instructions elsewhere: 2/)).toBeVisible();
  await expect(page.getByRole('button', { name: /approve|reject|create|submit|execute/i })).toHaveCount(0);
});
