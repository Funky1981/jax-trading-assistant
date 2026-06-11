import { expect, test, type Page } from '@playwright/test';

async function installTradingStubs(page: Page) {
  let brokerOrders = [
    {
      order_id: 101,
      symbol: 'AAPL',
      action: 'BUY',
      order_type: 'LMT',
      quantity: 10,
      limit_price: 259.8,
      status: 'Submitted',
      filled_qty: 0,
      avg_fill_price: 0,
      can_cancel: true,
      created_at: '2026-03-06T13:14:00Z',
      updated_at: '2026-03-06T13:15:00Z',
      order_ref: 'manual-entry',
    },
  ];

  const state = {
    bracketRequests: 0,
    cancelRequests: 0,
    protectRequests: 0,
    closeRequests: 0,
  };

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

  await page.route('**/api/v1/system/market-data-status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        connected: true,
        marketDataMode: 'delayed',
        paperTrading: true,
        checkedAt: '2026-03-06T13:15:00Z',
      }),
    });
  });

  await page.route('**/api/v1/trading/pilot-status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        pilotMode: true,
        authRequired: true,
        operatorRole: 'admin',
        allowedRoles: ['admin'],
        operatorAccess: true,
        brokerConnected: true,
        marketDataMode: 'delayed',
        paperTrading: true,
        readOnly: false,
        canTrade: true,
        quoteAuthority: false,
        intradayAuthority: false,
        executionFromChartBlocked: true,
        requiresManualBrokerConfirmation: true,
        reviewAgainstBroker: true,
        rollbackToReadOnly: true,
        etfPhase1Enabled: true,
        etfPolicyVersion: 'phase1-2026-05-13',
        etfPolicyHash: 'stubbed',
        etfEntryWorkflow: 'candidate_approval_only',
        etfReadinessReasons: [
          'ETF entries must use the approval workflow.',
          'Manual ETF entry orders are blocked from the order ticket.',
        ],
        reasons: [],
        checklist: [
          'Verify IB/TWS is connected and paper trading remains enabled.',
          'Confirm symbol, quantity, and exits in IB/TWS before every broker mutation.',
        ],
        checkedAt: '2026-03-06T13:15:00Z',
      }),
    });
  });

  await page.route('**/api/v1/instruments/etfs', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        version: 'phase1-2026-05-13',
        hash: 'stubbed',
        owner: 'trading-risk',
        policy: {
          quote_freshness_seconds: 60,
          max_spread_bps: 10,
        },
        instruments: [
          { symbol: 'SPY', instrument_type: 'etf', eligibility_state: 'approved' },
          { symbol: 'QQQ', instrument_type: 'etf', eligibility_state: 'approved' },
        ],
        checkedAt: '2026-03-06T13:15:00Z',
      }),
    });
  });

  await page.route('**/quotes/*', async (route) => {
    const symbol = route.request().url().split('/').pop() ?? 'AAPL';
    const quoteMap: Record<string, number> = {
      SPY: 677.39,
      QQQ: 604.12,
      AAPL: 260.25,
      TSLA: 402.3,
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

  await page.route('**/positions/*/protect', async (route) => {
    state.protectRequests += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        order_ids: [3001, 3002],
        cancelled_order_ids: [],
        message: 'Submitted protection for SPY',
      }),
    });
  });

  await page.route('**/positions/*/close', async (route) => {
    state.closeRequests += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        order_id: 3003,
        message: 'Close order submitted for SPY',
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

  await page.route('**/orders/bracket', async (route) => {
    state.bracketRequests += 1;
    brokerOrders = [
      {
        order_id: 202,
        symbol: 'AAPL',
        action: 'BUY',
        order_type: 'MKT',
        quantity: 10,
        limit_price: 0,
        status: 'Submitted',
        filled_qty: 0,
        avg_fill_price: 0,
        can_cancel: true,
        created_at: '2026-03-06T13:16:00Z',
        updated_at: '2026-03-06T13:16:00Z',
        order_ref: 'manual-entry',
      },
      ...brokerOrders,
    ];

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        parent_order_id: 202,
        child_order_ids: [203, 204],
        message: 'Bracket order submitted for AAPL',
      }),
    });
  });

  await page.route('**/orders', async (route) => {
    if (route.request().method() === 'POST') {
      brokerOrders = [
        {
          order_id: 205,
          symbol: 'AAPL',
          action: 'BUY',
          order_type: 'MKT',
          quantity: 10,
          limit_price: 0,
          status: 'Submitted',
          filled_qty: 0,
          avg_fill_price: 0,
          can_cancel: true,
          created_at: '2026-03-06T13:17:00Z',
          updated_at: '2026-03-06T13:17:00Z',
          order_ref: 'manual-entry',
        },
        ...brokerOrders,
      ];

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          order_id: 205,
          message: 'Order placed successfully',
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        orders: brokerOrders,
        count: brokerOrders.length,
      }),
    });
  });

  await page.route('**/orders/*', async (route) => {
    if (route.request().method() === 'POST' && route.request().url().endsWith('/orders/bracket')) {
      state.bracketRequests += 1;
      brokerOrders = [
        {
          order_id: 202,
          symbol: 'AAPL',
          action: 'BUY',
          order_type: 'MKT',
          quantity: 10,
          limit_price: 0,
          status: 'Submitted',
          filled_qty: 0,
          avg_fill_price: 0,
          can_cancel: true,
          created_at: '2026-03-06T13:16:00Z',
          updated_at: '2026-03-06T13:16:00Z',
          order_ref: 'manual-entry',
        },
        ...brokerOrders,
      ];

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          parent_order_id: 202,
          child_order_ids: [203, 204],
          message: 'Bracket order submitted for AAPL',
        }),
      });
      return;
    }

    state.cancelRequests += 1;
    brokerOrders = brokerOrders.map((order) =>
      String(order.order_id) === route.request().url().split('/').pop()
        ? { ...order, status: 'Cancelled', can_cancel: false, updated_at: '2026-03-06T13:18:00Z' }
        : order
    );

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        order_id: 101,
        status: 'PendingCancel',
        message: 'Cancel requested for 101',
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
          { timestamp: '2026-03-06T13:00:00Z', open: 258.61, high: 259.4, low: 258.2, close: 258.89, volume: 1200 },
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
            symbol: 'MSFT',
            direction: 'buy',
            type: 'market',
            quantity: 10,
            price: 410.1,
            avg_fill_price: 410.25,
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

  await page.route('**/api/v1/events**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        events: [
          {
            id: 'event-1',
            kind: 'market_news',
            primarySymbol: 'AAPL',
            title: 'Stubbed event',
            eventTime: '2026-03-06T13:10:00Z',
          },
        ],
        total: 1,
      }),
    });
  });

  await page.route('**/api/v1/datasets**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        datasets: [
          {
            datasetId: 'dataset-1',
            name: 'AAPL 15m sample',
            symbol: 'AAPL',
            datasetHash: '241f3c2bcdef',
          },
        ],
        total: 1,
      }),
    });
  });

  return state;
}

test('shows pilot trade gate on the system page', async ({ page }) => {
  await installTradingStubs(page);

  await page.goto('/system', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /System/ })).toBeVisible();
  await expect(page.getByText('Pilot Trading Status')).toBeVisible();
  await expect(page.getByText('Trade Enabled')).toBeVisible();
  await expect(page.getByText('Connected')).toBeVisible();
  await expect(page.getByText('Required')).toBeVisible();
  await expect(page.getByText('admin', { exact: true })).toBeVisible();
});

test('loads trading page with operator workflow and management controls', async ({ page }) => {
  await installTradingStubs(page);

  await page.goto('/trading', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { level: 1, name: /Trading/ })).toBeVisible();
  await expect(page.getByText('How to Use This Screen')).toBeVisible();
  await expect(page.getByText('Delayed', { exact: true }).first()).toBeVisible();
  await expect(page.getByText('Paper', { exact: true })).toBeVisible();
  await expect(page.getByText('Watchlist prices are non-authoritative during the pilot.')).toHaveCount(0);
  await expect(page.getByText('Intraday chart data is non-authoritative during the pilot. Use IB/TWS as the execution source of truth.')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Protect' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Close' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
});

test('submits protected entries and manages orders and positions', async ({ page }) => {
  const state = await installTradingStubs(page);

  await page.goto('/trading', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#order-ticket-symbol')).toBeVisible();

  await page.locator('#order-ticket-symbol').fill('AAPL');
  await page.locator('#order-ticket-quantity').fill('10');
  await page.locator('#order-ticket-stop-loss').fill('195');
  await page.locator('#order-ticket-take-profit').fill('270');
  await page.getByRole('button', { name: 'Submit BUY Bracket' }).click();
  await page.getByRole('checkbox', { name: /I confirmed the symbol, size, and market context in IB\/TWS/i }).check();
  await page.getByRole('button', { name: 'Submit Broker Order' }).click();

  await expect(page.getByText('Bracket order submitted for AAPL')).toBeVisible();
  await expect.poll(() => state.bracketRequests).toBe(1);

  await page.getByRole('button', { name: 'Cancel' }).first().click();
  await page.getByRole('checkbox', { name: /I confirmed in IB\/TWS that this working order should be cancelled/i }).check();
  await page.getByRole('button', { name: 'Submit Cancel' }).click();
  await expect.poll(() => state.cancelRequests).toBe(1);

  await page.getByRole('button', { name: 'Protect' }).first().click();
  const protectDialog = page.getByRole('dialog');
  await protectDialog.getByLabel('Stop Loss').fill('650');
  await protectDialog.getByRole('checkbox', { name: /I confirmed these protective levels in IB\/TWS/i }).check();
  await protectDialog.getByRole('button', { name: 'Submit Protection' }).click();
  await expect.poll(() => state.protectRequests).toBe(1);

  await page.getByRole('button', { name: 'Close' }).first().click();
  const closeDialog = page.getByRole('dialog');
  await closeDialog.getByRole('checkbox', { name: /I confirmed this exit in IB\/TWS/i }).check();
  await closeDialog.getByRole('button', { name: 'Submit Close' }).click();
  await expect.poll(() => state.closeRequests).toBe(1);
});

test('reroutes ETF manual entries to approval flow while leaving exposure actions available', async ({ page }) => {
  const state = await installTradingStubs(page);

  await page.goto('/trading', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#order-ticket-symbol')).toBeVisible();

  await page.locator('#order-ticket-symbol').fill('SPY');
  await page.locator('#order-ticket-quantity').fill('10');

  await expect(page.getByText(/Approval required for this ETF/i)).toBeVisible();
  await expect(page.getByRole('link', { name: 'Open approval flow' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Submit BUY Order' })).toBeDisabled();

  await page.getByRole('button', { name: 'Protect' }).first().click();
  const protectDialog = page.getByRole('dialog');
  await protectDialog.getByLabel('Stop Loss').fill('650');
  await protectDialog.getByRole('checkbox', { name: /I confirmed these protective levels in IB\/TWS/i }).check();
  await protectDialog.getByRole('button', { name: 'Submit Protection' }).click();
  await expect.poll(() => state.protectRequests).toBe(1);
});

test('approval paper order flow from promoted news idea', async ({ page }) => {
  await installTradingStubs(page);
  let approved = false;
  let approveRequests = 0;

  const queueCandidate = {
    id: 'candidate-news-approval-1',
    symbol: 'QQQ',
    signalType: 'BUY',
    confidence: 0.72,
    entryPrice: 500,
    stopLoss: 490,
    takeProfit: 520,
    reasoning: 'World Monitor highlighted QQQ after a rates-sensitive headline.',
    detectedAt: '2026-06-11T13:00:00Z',
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
        sourceEventId: 'wm-e2e-approval-1',
        headline: 'Rates-sensitive headline supports QQQ review.',
      },
    },
  };

  const approvedCandidate = {
    id: queueCandidate.id,
    strategyInstanceId: 'instance-etf-news',
    signalId: 'signal-news-approval-1',
    strategyId: 'etf_news_sector_momentum_v1',
    symbol: 'QQQ',
    signalType: 'BUY',
    status: 'approved',
    entryPrice: 500,
    stopLoss: 490,
    takeProfit: 520,
    confidence: 0.72,
    reasoning: queueCandidate.reasoning,
    sessionDate: '2026-06-11',
    detectedAt: '2026-06-11T13:00:00Z',
    dataProvenance: 'world-monitor',
    latestApproval: {
      id: 'approval-news-1',
      decision: 'approved',
      approvedBy: 'admin',
      decidedAt: '2026-06-11T13:05:00Z',
    },
    executionInstructionId: 'instruction-news-1',
  };

  await page.route('**/api/v1/research/events/world-monitor', async (route) => {
    await route.fulfill({
      status: 202,
      contentType: 'application/json',
      body: JSON.stringify({
        inbox_id: 'inbox-news-1',
        event_id: 'event-news-1',
        status: 'new',
        duplicate: false,
      }),
    });
  });

  await page.route('**/api/v1/research/events/world-monitor/promote', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        promoted: [
          {
            inboxId: 'inbox-news-1',
            eventId: 'event-news-1',
            signalId: 'signal-news-approval-1',
            candidateId: queueCandidate.id,
            symbol: 'QQQ',
            route: 'approval_required',
          },
        ],
        skipped: 0,
      }),
    });
  });

  await page.route('**/api/v1/approvals/queue**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(approved ? [] : [queueCandidate]),
    });
  });

  await page.route('**/api/v1/approvals/*/approve', async (route) => {
    approved = true;
    approveRequests += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        candidateId: queueCandidate.id,
        latestApproval: approvedCandidate.latestApproval,
        execution: {
          id: 'instruction-news-1',
          approvalId: 'approval-news-1',
          candidateId: queueCandidate.id,
          symbol: 'QQQ',
          signalType: 'BUY',
          entryPrice: 500,
          stopLoss: 490,
          takeProfit: 520,
          status: 'pending',
          createdAt: '2026-06-11T13:05:01Z',
          updatedAt: '2026-06-11T13:05:01Z',
        },
      }),
    });
  });

  await page.route('**/api/v1/candidates**', async (route) => {
    const url = new URL(route.request().url());
    const status = url.searchParams.get('status');
    const rows = approved && status === 'approved' ? [approvedCandidate] : [];
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(rows),
    });
  });

  await page.goto('/etf/approvals', { waitUntil: 'domcontentloaded' });

  await expect(page.getByRole('heading', { name: 'Approval Queue' })).toBeVisible();
  await expect(page.getByText('QQQ', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Show reasoning' })).toBeVisible();
  await page.getByRole('button', { name: /Approve for paper order/i }).click();
  await expect(page.getByText('Create paper instruction?')).toBeVisible();
  await page.getByRole('button', { name: /Yes, create paper order/i }).click();

  await expect.poll(() => approveRequests).toBe(1);
  await expect(page.getByText('Decision recorded: approve. Execution: pending')).toBeVisible();
  await expect(page.getByText('instruct...')).toBeVisible();
});
