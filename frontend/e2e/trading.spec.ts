import { expect, test } from '@playwright/test';

test('loads trading page with market mode badges and core panels', async ({ page }) => {
  await page.route('**/auth/status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false }),
    });
  });

  await page.route('**/health', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy', uptime: 'stubbed' }),
    });
  });

  await page.route('**/config', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ provider: 'ollama', model: 'llama3.2:latest' }),
    });
  });

  await page.route('**/quotes/*', async (route) => {
    const symbol = route.request().url().split('/').pop() ?? 'AAPL';
    const quoteMap: Record<string, number> = {
      SPY: 677.39,
      QQQ: 604.12,
      AAPL: 260.25,
      TSLA: 402.30,
      NVDA: 180.95,
      AMD: 197.88,
      META: 656.01,
      AMZN: 216.84,
    };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        symbol,
        price: quoteMap[symbol] ?? 100,
        change: 1.64,
        change_percent: 0.63,
        volume: 1000000,
        high: 261.1,
        low: 258.4,
      }),
    });
  });

  await page.route('**/positions', async (route) => {
    await route.fulfill({
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
            unrealized_pnl: -14969.05,
            market_value: 547511.15,
          },
        ],
      }),
    });
  });

  await page.route('**/account', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        account_id: 'DU123456',
        net_liquidation: 1041631.08,
        total_cash: 120000,
        buying_power: 300000,
        equity_with_loan: 1041631.08,
        currency: 'USD',
      }),
    });
  });

  await page.route('**/api/v1/strategies', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        strategies: [
          {
            id: 'ma_crossover_v1',
            name: 'MA Crossover V1',
            description: 'Trend-following crossover strategy',
            status: 'active',
            performance: {
              totalPnl: 7732.95,
              winRate: 0.64,
              trades: 42,
              sharpe: 1.42,
            },
            lastSignal: {
              symbol: 'AAPL',
              action: 'buy',
              timestamp: Date.parse('2026-03-06T13:10:00Z'),
              confidence: 0.87,
            },
          },
        ],
      }),
    });
  });

  await page.route('**/api/v1/market/candles**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        symbol: 'AAPL',
        timeframe: '15m',
        requestedTimeframe: '15m',
        degraded: false,
        marketDataMode: 'delayed',
        paperTrading: true,
        candles: [
          { timestamp: '2026-03-06T13:00:00Z', open: 258.61, high: 259.40, low: 258.20, close: 258.89, volume: 1200 },
          { timestamp: '2026-03-06T13:15:00Z', open: 258.89, high: 260.41, low: 258.75, close: 260.25, volume: 1400 },
        ],
      }),
    });
  });

  await page.route('**/api/v1/trades**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        trades: [
          {
            id: 'trade-1',
            symbol: 'AAPL',
            direction: 'buy',
            type: 'market',
            quantity: 10,
            price: 259.9,
            avg_fill_price: 260.25,
            order_status: 'filled',
            filled_qty: 10,
            created_at: '2026-03-06T13:15:00Z',
            updated_at: '2026-03-06T13:16:00Z',
          },
        ],
      }),
    });
  });

  await page.route('**/api/v1/signals**', async (route) => {
    await route.fulfill({
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

  await page.route('**/api/v1/recommendations**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        recommendations: [
          {
            id: 'rec-1',
            signal: { id: 'signal-1' },
            ai_analysis: {
              status: 'completed',
              agent_suggestion: 'BUY',
              confidence: 0.87,
              reasoning: 'Trend and momentum support the long setup.',
            },
          },
        ],
        total: 1,
      }),
    });
  });

  await page.goto('/trading', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /Trading/ })).toBeVisible();
  await expect(page.getByRole('heading', { level: 3, name: 'Price Chart' })).toBeVisible();
  await expect(page.getByText('DELAYED')).toBeVisible();
  await expect(page.getByText('Paper', { exact: true })).toBeVisible();
  await expect(page.locator('p.text-2xl.font-mono.font-bold')).toHaveText('$260.25');
  await expect(page.getByRole('heading', { level: 3, name: 'Trade Blotter' })).toBeVisible();
  await expect(page.getByRole('heading', { level: 3, name: 'Trading Approvals' })).toBeVisible();
  await expect(page.getByText('Trend and momentum support the long setup.')).toBeVisible();
});
