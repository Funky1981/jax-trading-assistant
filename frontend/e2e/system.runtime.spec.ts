import { expect, test } from '@playwright/test';

const username = process.env.JAX_RUNTIME_USERNAME;
const password = process.env.JAX_RUNTIME_PASSWORD;

test.skip(!username || !password, 'Runtime credentials are required for persisted-data proof.');

for (const width of [320, 768, 1280]) {
  test(`persisted System Safety runtime evidence at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/login');
    await page.getByLabel('Username').fill(username!);
    await page.getByLabel('Password').fill(password!);
    await page.getByRole('button', { name: 'Sign in' }).click();
    await expect(page).toHaveURL(/\/$/);
    await expect(page.getByText('Paper-safe', { exact: true })).toBeVisible();

    const runtime = await page.evaluate(async () => {
      const token = localStorage.getItem('jax_token');
      const headers = { Authorization: `Bearer ${token}` };
      const response = await fetch('http://localhost:8081/api/v1/operator-evidence/candidates', {
        headers,
      });
      const candidates = (await response.json()) as Array<{
        candidateId: string;
        symbol: string;
        paperTicketId?: string;
      }>;
      return candidates.find((candidate) => candidate.symbol === 'QQQ' && candidate.paperTicketId);
    });
    expect(runtime?.candidateId).toBeTruthy();
    await page.goto(`/system?candidateId=${runtime!.candidateId}`);

    await expect(page.getByRole('heading', { level: 1, name: 'System Safety' })).toBeVisible();
    await expect(page.getByText('Paper', { exact: true })).toBeVisible();
    await expect(page.getByText('Off', { exact: true })).toBeVisible();
    await expect(page.getByText('Disabled', { exact: true })).toBeVisible();
    await expect(page.getByText('Stopped', { exact: true })).toBeVisible();
    await expect(page.getByText('Not allowed', { exact: true })).toBeVisible();
    await expect(page.getByText('1x', { exact: true })).toBeVisible();
    await expect(
      page.getByText(
        'Paper-safe mode is on. Live trading, execution and broker activity are disabled.',
      ),
    ).toBeVisible();
    await expect(page.getByText('This journey created no execution records.')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Historical records' })).toBeVisible();
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

    await page.screenshot({
      path: `../images/acceptance/phase4-system-safety-default-${width}.png`,
      fullPage: true,
    });
    await page.getByText('Technical diagnostics').click();
    await expect(page.getByText('ALLOW_LIVE_TRADING')).toBeVisible();
    await page.screenshot({
      path: `../images/acceptance/phase4-system-safety-diagnostics-${width}.png`,
      fullPage: true,
    });
  });
}
