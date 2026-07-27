import { expect, test } from '@playwright/test';

const username = process.env.JAX_RUNTIME_USERNAME;
const password = process.env.JAX_RUNTIME_PASSWORD;

test.skip(!username || !password, 'Runtime credentials are required for persisted-data proof.');

for (const width of [320, 768, 1280]) {
  test(`persisted Evidence Inbox runtime proof at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/login');
    await page.getByLabel('Username').fill(username!);
    await page.getByLabel('Password').fill(password!);
    await page.getByRole('button', { name: 'Sign in' }).click();
    await expect(page).toHaveURL(/\/$/);
    const homeDecisions = page.getByRole('region', { name: 'Persisted evidence decisions' });
    await expect(homeDecisions.getByText('NO_TRADE', { exact: true })).toBeVisible();
    await expect(homeDecisions.getByText('WATCH', { exact: true })).toBeVisible();
    await expect(homeDecisions.getByText('CANDIDATE', { exact: true })).toBeVisible();
    await expect(homeDecisions.getByText('Awaiting processing', { exact: true })).toBeVisible();

    if (width < 768) {
      await page.getByRole('button', { name: 'Open sidebar' }).click();
    }
    await page.getByRole('link', { name: 'Evidence Inbox' }).first().click();
    await expect(page).toHaveURL(/\/monitor\/inbox$/);
    await expect(page.getByRole('heading', { name: 'Evidence Inbox' })).toBeVisible();
    await expect(
      page.getByLabel('Evidence items').getByRole('button', { name: /Houthi allies/i }).first(),
    ).toBeVisible();
    await expect(
      page.getByText(/Open an evidence item to see its source, timestamps/i),
    ).toHaveCount(0);

    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: `../images/acceptance/evidence-inbox-refinement-collapsed-${width}.png`,
      fullPage: true,
    });

    await page.getByRole('button', { name: 'Genuine', exact: true }).click();
    await page.getByLabel('Evidence items').getByRole('button').first().click();

    await expect(page.getByText('GENUINE').first()).toBeVisible();
    await expect(page.getByText('WATCH').last()).toBeVisible();
    await expect(page.getByText('Published time')).toBeVisible();
    await expect(page.getByText('Collection time').first()).toBeVisible();
    await expect(page.getByText('Jax receipt time').first()).toBeVisible();
    await page.getByText('Analysis', { exact: true }).click();
    await expect(page.getByText('DETERMINISTIC ANALYSIS')).toBeVisible();
    await expect(page.getByText('No AI used')).toBeVisible();
    await expect(page.getByText('Unknown assets', { exact: true }).nth(1)).toBeVisible();
    await page.getByText('Decision — WATCH', { exact: true }).click();
    await expect(page.getByText('genuine-event-decision-v1')).toBeVisible();
    await expect(page.getByText('jax-genuine-event-decision-processor')).toBeVisible();
    await expect(page.getByText('unknown_assets_prevent_candidate')).toBeVisible();
    await expect(page.getByRole('link', { name: /Open original source/i }).first()).toHaveAttribute(
      'href',
      /^https:\/\//,
    );
    await expect(page.getByRole('link', { name: 'Open Candidate Review' })).toHaveCount(0);

    await page.getByText('Audit', { exact: true }).click();
    await expect(page.getByText('Source-event ID')).toBeVisible();
    await expect(page.getByText('Show raw payload', { exact: true })).toBeVisible();
    await expect(
      page.getByRole('button', {
        name: /^(approve|reject candidate|execute|place trade|create order|submit order)$/i,
      }),
    ).toHaveCount(0);

    const overflow = await page.evaluate(() => ({
      document: document.documentElement.scrollWidth - document.documentElement.clientWidth,
      body: document.body.scrollWidth - document.body.clientWidth,
      scrollX: window.scrollX,
      offenders: [...document.querySelectorAll<HTMLElement>('body *')]
        .filter(
          (element) =>
            element.getBoundingClientRect().right + window.scrollX >
              document.documentElement.clientWidth + 1 ||
            element.getBoundingClientRect().left + element.scrollWidth >
              document.documentElement.clientWidth + 1,
        )
        .slice(-10)
        .map((element) => ({
          tag: element.tagName,
          className: element.className,
          right: Math.round(element.getBoundingClientRect().right),
          clientWidth: element.clientWidth,
          scrollWidth: element.scrollWidth,
          text: element.textContent?.trim().slice(0, 80),
        })),
    }));
    expect(overflow.document, JSON.stringify(overflow)).toBeLessThanOrEqual(1);
    expect(overflow.body, JSON.stringify(overflow)).toBeLessThanOrEqual(1);
    const evidenceSection = page
      .getByRole('heading', { name: 'Evidence received' })
      .locator('..')
      .locator('..');
    expect(
      await evidenceSection
        .locator('[class*="sticky"], [class*="overflow-y"], [class*="h-screen"]')
        .count(),
    ).toBe(0);

    await page.getByRole('button', { name: 'NO_TRADE', exact: true }).click();
    await page.getByLabel('Evidence items').getByRole('button').first().click();
    await expect(page.getByText('NO_TRADE').last()).toBeVisible();
    await expect(page.getByRole('link', { name: 'Open Candidate Review' })).toHaveCount(0);

    if (width < 768) {
      await page.getByRole('button', { name: 'Open sidebar' }).click();
    }
    await page.getByRole('link', { name: 'System' }).first().click();
    await expect(page.getByRole('heading', { name: 'System Safety' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Paper-safe mode is on' })).toBeVisible();

    await page.evaluate(() => window.scrollTo(0, 0));
    await page.screenshot({
      path: `../images/acceptance/evidence-inbox-refinement-expanded-${width}.png`,
      fullPage: true,
    });
  });
}
