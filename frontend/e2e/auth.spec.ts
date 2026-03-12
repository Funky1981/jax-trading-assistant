/**
 * E2E: Authentication flow
 *
 * User stories covered:
 *  US-AUTH-1  Anonymous mode — no login required, dashboard is directly accessible.
 *  US-AUTH-2  Auth required — unauthenticated visit to a protected route redirects to /login.
 *  US-AUTH-3  Login page — form elements are present and accessible.
 *  US-AUTH-4  Invalid credentials — submitting bad creds shows an error alert.
 *  US-AUTH-5  Successful login — valid creds store a session and show the dashboard.
 */
import { expect, test } from '@playwright/test';
import { TEST_JWT } from './helpers';

// ---------------------------------------------------------------------------
// US-AUTH-1: Anonymous mode
// ---------------------------------------------------------------------------
test('anonymous mode: dashboard accessible without login', async ({ page }) => {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false }),
    }),
  );
  await page.route('**/health', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy', uptime: 'stubbed' }),
    }),
  );

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(page).not.toHaveURL(/\/login/);
});

// ---------------------------------------------------------------------------
// US-AUTH-2: Unauthenticated access → redirect
// ---------------------------------------------------------------------------
test('auth required: unauthenticated visit redirects to /login', async ({ page }) => {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: true }),
    }),
  );
  // Clear any existing token before the page loads.
  await page.addInitScript(() => localStorage.removeItem('jax_token'));

  await page.goto('/', { waitUntil: 'domcontentloaded' });

  await expect(page).toHaveURL(/\/login/);
});

// ---------------------------------------------------------------------------
// US-AUTH-3: Login page form elements
// ---------------------------------------------------------------------------
test('login page: renders username, password, and sign-in button', async ({ page }) => {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: true }),
    }),
  );
  await page.addInitScript(() => localStorage.removeItem('jax_token'));

  await page.goto('/login', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'JAX Trading' })).toBeVisible();
  await expect(page.locator('#username')).toBeVisible();
  await expect(page.locator('#password')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
  await expect(page.getByText('Sign in to access the workspace')).toBeVisible();
});

// ---------------------------------------------------------------------------
// US-AUTH-4: Invalid credentials → error alert
// ---------------------------------------------------------------------------
test('login page: invalid credentials show an error alert', async ({ page }) => {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: true }),
    }),
  );
  // Backend returns 401 with a message field (AuthContext reads body.message).
  await page.route('**/auth/login', (route) =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'Invalid credentials' }),
    }),
  );
  await page.addInitScript(() => localStorage.removeItem('jax_token'));

  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.locator('#username').fill('wronguser');
  await page.locator('#password').fill('wrongpass');
  await page.getByRole('button', { name: 'Sign in' }).click();

  // LoginPage renders the error in a <p role="alert"> element.
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByRole('alert')).toContainText(/invalid credentials/i);
});

// ---------------------------------------------------------------------------
// US-AUTH-5: Successful login → dashboard
// ---------------------------------------------------------------------------
test('login page: valid credentials navigate to dashboard', async ({ page }) => {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: true }),
    }),
  );
  // Return a structurally valid JWT (fake sig) so AuthContext can decode it.
  await page.route('**/auth/login', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      // AuthContext reads the `access_token` field.
      body: JSON.stringify({ access_token: TEST_JWT }),
    }),
  );
  await page.route('**/health', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy', uptime: 'stubbed' }),
    }),
  );
  await page.addInitScript(() => localStorage.removeItem('jax_token'));

  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.locator('#username').fill('admin');
  await page.locator('#password').fill('password');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
});
