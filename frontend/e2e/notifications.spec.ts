import { expect, test } from '@playwright/test';
import { stubBase } from './helpers';

test('notifications inbox shows unread/read, stale, and route-aware destinations', async ({ page }) => {
  await stubBase(page);

  await page.route('**/api/v1/events**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        events: [
          {
            id: 'event-approval-1',
            kind: 'approval_required',
            title: 'Approval required for QQQ candidate',
            summary: 'Open approvals to continue this workflow.',
            eventTime: '2026-05-27T12:00:00Z',
            attributes: { channels: ['in_app'] },
          },
          {
            id: 'event-sentiment-1',
            kind: 'sentiment_invalidated',
            title: 'Sentiment invalidated previous setup',
            summary: 'Review AI Trading for updated evidence and next steps.',
            eventTime: '2026-05-20T08:00:00Z',
            attributes: { route: '/ai-trading', channels: ['in_app', 'email'] },
          },
        ],
        total: 2,
        limit: 100,
        offset: 0,
      }),
    })
  );

  await page.goto('/notifications', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Notification Centre' })).toBeVisible();
  await expect(page.getByText('Approval required for QQQ candidate')).toBeVisible();
  await expect(page.getByText('Sentiment invalidated previous setup')).toBeVisible();
  await expect(page.getByText('Stale')).toBeVisible();

  const destinationLinks = page.getByRole('link', { name: 'Open destination' });
  await expect(destinationLinks).toHaveCount(2);
  await expect(destinationLinks.nth(0)).toHaveAttribute('href', '/etf/approvals');
  await expect(destinationLinks.nth(1)).toHaveAttribute('href', '/ai-trading');

  await page.getByRole('button', { name: 'Mark read' }).first().click();
  await expect(page.getByLabel('Notification inbox').getByText('Read', { exact: true }).first()).toBeVisible();

  await page.getByRole('button', { name: 'Mark all read' }).click();
  await expect(page.locator('p.text-2xl.font-semibold').first()).toHaveText('0');
});
