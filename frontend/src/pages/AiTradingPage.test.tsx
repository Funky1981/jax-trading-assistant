import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AiTradingPage } from './AiTradingPage';
import { approvalsService, candidatesService } from '@/data/approvals-service';
import { aiService } from '@/data/ai-service';
import { signalsService } from '@/data/signals-service';

vi.mock('@/data/signals-service', () => ({
  signalsService: {
    list: vi.fn(),
  },
}));

vi.mock('@/data/approvals-service', () => ({
  approvalsService: {
    getQueue: vi.fn(),
  },
  candidatesService: {
    list: vi.fn(),
  },
}));

vi.mock('@/data/ai-service', () => ({
  aiService: {
    getOverview: vi.fn(),
    updateScanner: vi.fn(),
  },
}));

function mockOverview() {
  vi.mocked(aiService.getOverview).mockResolvedValue({
    checkedAt: '2026-05-22T10:00:00Z',
    scanner: {
      enabled: true,
      assetScope: 'etf',
      symbols: ['SPY', 'QQQ', 'IWM'],
      universePreset: 'etf-core',
      intervalSeconds: 300,
      minimumConfidence: 0.7,
      sentiment: {
        enabled: false,
        sourceScope: 'news',
        window: '24h',
        threshold: 0.6,
        minimumSourceCount: 3,
        sourceTrustWeightingMode: 'equal',
        mode: 'filter',
      },
      status: 'ready',
      channels: {
        inApp: true,
        desktopWeb: false,
        mobilePush: false,
      },
      policy: {
        manualRouteEnabled: true,
        approvalRouteEnabled: true,
        requiresHumanApproval: true,
      },
    },
    opportunityCounts: {
      signalsPending: 1,
      candidates: 1,
      approvals: 1,
    },
    policySummary: {
      requiresHumanApproval: true,
      manualRouteEnabled: true,
      approvalRouteEnabled: true,
    },
    channelSummary: {
      inApp: true,
      desktopWeb: false,
      mobilePush: false,
    },
  });
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <AiTradingPage />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('AiTradingPage', () => {
  it('renders scanner state and unified Opportunity feed with route-aware actions', async () => {
    mockOverview();
    vi.mocked(signalsService.list).mockResolvedValue({
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
    });
    vi.mocked(candidatesService.list).mockResolvedValue([
      {
        id: 'candidate-blocked',
        strategyInstanceId: 'instance-1',
        symbol: 'SPY',
        signalType: 'BUY',
        status: 'blocked',
        confidence: 0.71,
        reasoning: 'ETF setup is positive but blocked by policy.',
        blockReason: 'ETF entries require approval queue routing.',
        sessionDate: '2026-05-22',
        detectedAt: '2026-05-22T09:35:00Z',
        dataProvenance: 'paper',
      },
    ]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([
      {
        id: 'candidate-approval',
        symbol: 'QQQ',
        signalType: 'SELL',
        confidence: 0.67,
        reasoning: 'Risk-off rotation detected.',
        detectedAt: '2026-05-22T09:40:00Z',
        instanceName: 'ETF guardrail',
      },
    ]);

    renderPage();

    expect(await screen.findByRole('heading', { name: 'AI Trading' })).toBeInTheDocument();
    expect(await screen.findByText('QQQ')).toBeInTheDocument();
    expect(screen.getByText('SPY')).toBeInTheDocument();
    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Send to approval/i })).toHaveAttribute('href', '/etf/approvals');
    expect(screen.getByRole('link', { name: /Open blocked-state guidance/i })).toHaveAttribute(
      'href',
      '/candidates/candidate-blocked/evidence'
    );
    expect(screen.getByRole('link', { name: /Review order/i })).toHaveAttribute('href', '/manual-trading');
    expect(screen.getAllByRole('button', { name: 'Watch' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('button', { name: 'Dismiss' }).length).toBeGreaterThan(0);
  });

  it('renders scanner and sentiment settings from API overview', async () => {
    mockOverview();
    vi.mocked(signalsService.list).mockResolvedValue({ signals: [], total: 0, limit: 12, offset: 0 });
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    expect(await screen.findByRole('heading', { name: 'Scanner settings' })).toBeInTheDocument();
    expect(screen.getByText('Watching')).toBeInTheDocument();
    expect(screen.getByText('Connected')).toBeInTheDocument();
    expect(screen.getByLabelText('Symbols')).toHaveDisplayValue('SPY, QQQ, IWM');
    expect(screen.getByLabelText('Minimum confidence')).toHaveDisplayValue('70%');
    expect(screen.getByLabelText('Sentiment source scope')).toHaveDisplayValue('news');
    expect(screen.getByLabelText('Sentiment time window')).toHaveDisplayValue('24h');
    expect(screen.getByLabelText('Minimum sentiment threshold')).toHaveDisplayValue('60%');
    expect(screen.getByLabelText('Source trust weighting')).toHaveDisplayValue('Equal source weighting');
    expect(screen.getByLabelText('Sentiment mode')).toHaveDisplayValue('Filter');
    expect(screen.getByText(/persisted and connected to the AI scanner API/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Pause scanner/i })).toBeInTheDocument();
  });

  it('renders an explicit empty state', async () => {
    mockOverview();
    vi.mocked(signalsService.list).mockResolvedValue({ signals: [], total: 0, limit: 12, offset: 0 });
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    expect(await screen.findByText(/No Opportunities are available right now/i)).toBeInTheDocument();
  });

  it('renders an explicit partial error state', async () => {
    mockOverview();
    vi.mocked(signalsService.list).mockRejectedValue(new Error('offline'));
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    await waitFor(() => expect(screen.getByText(/Failed to load one or more Opportunity sources/i)).toBeInTheDocument());
  });
});
