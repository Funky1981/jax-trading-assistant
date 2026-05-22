import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AiTradingPage } from './AiTradingPage';
import { approvalsService, candidatesService } from '@/data/approvals-service';
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

  it('renders an explicit empty state', async () => {
    vi.mocked(signalsService.list).mockResolvedValue({ signals: [], total: 0, limit: 12, offset: 0 });
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    expect(await screen.findByText(/No Opportunities are available right now/i)).toBeInTheDocument();
  });

  it('renders an explicit partial error state', async () => {
    vi.mocked(signalsService.list).mockRejectedValue(new Error('offline'));
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    await waitFor(() => expect(screen.getByText(/Failed to load one or more Opportunity sources/i)).toBeInTheDocument());
  });
});
