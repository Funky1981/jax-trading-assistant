/**
 * Shared Playwright stub helpers.
 *
 * Prefer calling these before page.goto() so all routes are registered
 * before any navigation fires. For tests that need to track specific
 * mutation endpoints, register those routes AFTER calling these helpers —
 * Playwright evaluates routes last-registered-first, so a more-specific
 * route registered later will win over a broad wildcard registered earlier.
 */
import type { Page } from '@playwright/test';

// ---------------------------------------------------------------------------
// Test JWT fixture
// AuthContext only decodes the base64 payload (token.split('.')[1]) to read
// username / role / exp. No signature is verified client-side.
// Expiry is far in the future so this never needs regenerating.
// btoa() is available in DOM lib (TS) and Node 18+ (Playwright runtime).
// ---------------------------------------------------------------------------
const _jwtPayload = btoa(
  JSON.stringify({ username: 'admin', role: 'admin', exp: 9_999_999_999 }),
);

/** A structurally valid JWT (fake signature) that AuthContext will accept. */
export const TEST_JWT = `x.${_jwtPayload}.y`;

// ---------------------------------------------------------------------------
// Base stubs (used by almost every spec)
// ---------------------------------------------------------------------------

export async function stubAuthDisabled(page: Page): Promise<void> {
  await page.route('**/auth/status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false }),
    }),
  );
}

export async function stubHealth(page: Page): Promise<void> {
  await page.route('**/health', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy', uptime: 'stubbed' }),
    }),
  );
}

/** Stubs both auth/status (disabled) and /health. */
export async function stubBase(page: Page): Promise<void> {
  await stubAuthDisabled(page);
  await stubHealth(page);
}

// ---------------------------------------------------------------------------
// Domain stubs
// ---------------------------------------------------------------------------

export async function stubAccount(page: Page): Promise<void> {
  await page.route('**/account', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        account_id: 'DU123456',
        net_liquidation: 1_041_631.08,
        total_cash: 120_000,
        buying_power: 300_000,
        equity_with_loan: 1_041_631.08,
        currency: 'USD',
      }),
    }),
  );
}

export async function stubPositions(page: Page): Promise<void> {
  await page.route('**/positions', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        positions: [
          {
            contract_id: 'spy',
            symbol: 'SPY',
            quantity: 810,
            avg_cost: 694.42,
            market_price: 675.94,
            unrealized_pnl: -14_969.05,
            market_value: 547_511.15,
          },
        ],
      }),
    }),
  );
}

/**
 * Stub for GET /api/v1/signals.
 * Only fulfils GET requests — POST (approve/reject) will continue so
 * a separately-registered, more-specific route can handle them.
 */
export async function stubSignals(page: Page): Promise<void> {
  await page.route('**/api/v1/signals**', (route) => {
    if (route.request().method() !== 'GET') {
      // Let more-specific route handlers respond to mutations.
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    }
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        signals: [
          {
            id: 'signal-1',
            symbol: 'AAPL',
            strategy_id: 'ma_crossover_v1',
            signal_type: 'BUY',
            confidence: 0.87,
            entry_price: 185.5,
            stop_loss: 179.04,
            take_profit: 192.89,
            status: 'pending',
            generated_at: '2026-03-06T13:10:00Z',
            created_at: '2026-03-06T13:10:00Z',
          },
        ],
        total: 1,
      }),
    });
  });
}

export async function stubRecommendations(page: Page): Promise<void> {
  await page.route('**/api/v1/recommendations**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ recommendations: [], total: 0 }),
    }),
  );
}

export async function stubPilotStatus(page: Page): Promise<void> {
  await page.route('**/api/v1/trading/pilot-status', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        pilotMode: true,
        canTrade: true,
        paperTrading: true,
        brokerConnected: true,
        marketDataMode: 'delayed',
        readOnly: false,
        reasons: [],
        checkedAt: '2026-03-06T13:15:00Z',
      }),
    }),
  );
}
