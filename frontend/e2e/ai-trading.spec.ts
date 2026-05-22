import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

test('AI Trading opens from the shell and renders route-aware opportunities', async ({ page }) => {
  await stubBase(page);
  await page.route('**/api/v1/signals**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        signals: [
          {
            id: 'signal-1',
            symbol: 'AAPL',
            strategy_id: 'breakout',
            signal_type: 'BUY',
            confidence: 0.86,
            reasoning: 'Momentum and volume confirm the setup.',
            generated_at: '2026-05-22T09:30:00Z',
            status: 'pending',
            created_at: '2026-05-22T09:30:00Z',
          },
        ],
        total: 1,
        limit: 12,
        offset: 0,
      }),
    })
  );
  await page.route('**/api/v1/candidates**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'candidate-blocked',
          strategyInstanceId: 'instance-1',
          symbol: 'SPY',
          signalType: 'BUY',
          status: 'blocked',
          confidence: 0.72,
          reasoning: 'ETF setup is positive but blocked by policy.',
          blockReason: 'ETF entries require approval queue routing.',
          sessionDate: '2026-05-22',
          detectedAt: '2026-05-22T09:35:00Z',
          dataProvenance: 'paper',
        },
      ]),
    })
  );
  await page.route('**/api/v1/approvals/queue**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'candidate-approval',
          symbol: 'QQQ',
          signalType: 'SELL',
          confidence: 0.67,
          reasoning: 'Risk-off rotation detected.',
          detectedAt: '2026-05-22T09:40:00Z',
          instanceName: 'ETF guardrail',
        },
      ]),
    })
  );

  await page.goto('/', { waitUntil: 'domcontentloaded' });
  await page.getByRole('link', { name: 'AI Trading' }).click();

  await expect(page).toHaveURL(/\/ai-trading$/);
  await expect(page.getByRole('heading', { name: 'AI Trading' })).toBeVisible();
  await expect(page.getByText('Opportunity queue')).toBeVisible();
  await expect(page.getByText('AAPL')).toBeVisible();
  await expect(page.getByText('SPY')).toBeVisible();
  await expect(page.getByText('QQQ')).toBeVisible();
  await expect(page.getByRole('link', { name: /Review order/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /Send to approval/i })).toBeVisible();
  await expect(page.getByRole('link', { name: /Open blocked-state guidance/i })).toBeVisible();
});
