import { expect, test } from '@playwright/test';

const username = process.env.JAX_RUNTIME_USERNAME;
const password = process.env.JAX_RUNTIME_PASSWORD;
const baseURL = process.env.JAX_RUNTIME_BASE_URL ?? 'http://localhost:3000';

test.skip(!username || !password, 'Runtime credentials are required for proxy-auth verification.');

test('runtime login reaches the auth handler through the frontend proxy', async ({ page }) => {
  await page.goto(`${baseURL}/login`);
  await page.getByLabel('Username').fill(username!);
  await page.getByLabel('Password').fill(password!);
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page).toHaveURL(`${baseURL}/`);
  await expect(page.getByRole('heading', { name: 'Jax overview' })).toBeVisible();
});
