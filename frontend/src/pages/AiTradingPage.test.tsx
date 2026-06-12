import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AiTradingPage } from './AiTradingPage';
import { approvalsService, candidatesService } from '@/data/approvals-service';
import { aiService } from '@/data/ai-service';
import { signalsService } from '@/data/signals-service';
import { BeginnerUXProvider } from '@/context/BeginnerUXContext';

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
    getWorldMonitorStatus: vi.fn(),
    updateScanner: vi.fn(),
    promoteSuggestion: vi.fn(),
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
  vi.mocked(aiService.getWorldMonitorStatus).mockResolvedValue({
    connected: true,
    lastReceivedAt: '2026-05-22T09:41:00Z',
    lastSourceEventId: 'world-monitor-event-1',
    lastStatus: 'candidate_created',
    lastHeadline: 'Softer inflation supports growth ETF review',
    lastSymbols: ['QQQ'],
    lastCandidateId: 'candidate-approval',
    counts: {
      total: 3,
      pending: 0,
      candidatesCreated: 2,
      rejected: 1,
      ignored: 0,
    },
    checkedAt: '2026-05-22T09:42:00Z',
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
        <BeginnerUXProvider>
          <AiTradingPage />
        </BeginnerUXProvider>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('AiTradingPage', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    vi.restoreAllMocks();
  });

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
        reasoning: 'News is relevant but chart confirmation is missing.',
        blockReason: 'Needs chart confirmation: only 0 recent candles are available for SPY; at least 20 are required.',
        blockedReasonCode: 'no_chart_confirmation',
        sessionDate: '2026-05-22',
        detectedAt: '2026-05-22T09:35:00Z',
        dataProvenance: 'paper',
      },
      {
        id: 'candidate-approved',
        strategyInstanceId: 'instance-2',
        symbol: 'IWM',
        signalType: 'BUY',
        status: 'approved',
        confidence: 0.82,
        reasoning: 'Approved after evidence review and waiting in the execution chain.',
        sessionDate: '2026-05-22',
        detectedAt: '2026-05-22T09:37:00Z',
        dataProvenance: 'world-monitor',
        latestApproval: {
          id: 'approval-1',
          decision: 'approved',
          approvedBy: 'operator',
          decidedAt: '2026-05-22T09:42:00Z',
        },
        executionInstructionId: 'instruction-1',
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

    expect(await screen.findByRole('heading', { name: 'Find Trade Ideas' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Start Here' })).toBeInTheDocument();
    expect(await screen.findByText('Softer inflation supports growth ETF review')).toBeInTheDocument();
    expect(screen.getByText('Receiving news')).toBeInTheDocument();
    expect(await screen.findByText('QQQ')).toBeInTheDocument();
    expect(screen.getByText('SPY')).toBeInTheDocument();
    expect(screen.getByText('IWM')).toBeInTheDocument();
    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.getByText('Needs chart confirmation')).toBeInTheDocument();
    expect(screen.getByText('Execution chain')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Send to approval/i })).toHaveAttribute('href', '/etf/approvals');
    expect(screen.getByRole('link', { name: /View execution chain/i })).toHaveAttribute('href', '/approvals');
    expect(screen.getByRole('link', { name: /Review chart evidence/i })).toHaveAttribute(
      'href',
      '/candidates/candidate-blocked/evidence'
    );
    expect(screen.getByRole('link', { name: /Review order/i })).toHaveAttribute('href', '/manual-trading');
    expect(screen.getAllByRole('button', { name: 'Watch' }).length).toBeGreaterThan(0);
    expect(screen.getAllByRole('button', { name: 'Dismiss' }).length).toBeGreaterThan(0);
  });

  it('lets operators watch and dismiss opportunities locally', async () => {
    const user = userEvent.setup();
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
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    expect(await screen.findByText('AAPL')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Watch' }));
    expect(screen.getByRole('button', { name: 'Watching' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(screen.queryByText('AAPL')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Show dismissed/i })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /Show dismissed/i }));
    expect(await screen.findByText('AAPL')).toBeInTheDocument();
  });

  it('renders scanner and sentiment settings from API overview', async () => {
    window.localStorage.setItem('beginner-mode', 'technical');
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

  it('renders a compact scanner control in simple mode', async () => {
    mockOverview();
    vi.mocked(signalsService.list).mockResolvedValue({ signals: [], total: 0, limit: 12, offset: 0 });
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);

    renderPage();

    expect(await screen.findByRole('heading', { name: 'Scanner' })).toBeInTheDocument();
    expect(screen.getByText(/Jax is watching SPY, QQQ, IWM for new ideas/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Pause scanner/i })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Scanner settings' })).not.toBeInTheDocument();
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

  it('promotes a BUY suggestion into the approval queue', async () => {
    const user = userEvent.setup();
    mockOverview();
    vi.mocked(signalsService.list).mockResolvedValue({ signals: [], total: 0, limit: 12, offset: 0 });
    vi.mocked(candidatesService.list).mockResolvedValue([]);
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);
    vi.mocked(aiService.promoteSuggestion).mockResolvedValue({
      candidateId: 'candidate-ai-1',
      signalId: 'signal-ai-1',
      route: 'approval_required',
      status: 'awaiting_approval',
    });
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({
        symbol: 'SPY',
        action: 'BUY',
        confidence: 64,
        reasoning: 'Agent0 found a paper-only setup.',
        risk: { risk_level: 'medium' },
        generated_at: '2026-06-11T10:00:00Z',
      }),
    } as Response);

    renderPage();

    await user.click(await screen.findByRole('button', { name: 'Ask' }));
    expect(await screen.findByText('Agent0 found a paper-only setup.')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Send to approval queue' }));

    await waitFor(() =>
      expect(aiService.promoteSuggestion).toHaveBeenCalledWith({
        symbol: 'SPY',
        action: 'BUY',
        confidence: 0.64,
        reasoning: 'Agent0 found a paper-only setup.',
        risk: 'medium',
        source: 'agent0_manual_review',
      })
    );
    expect(await screen.findByText(/Sent to Approvals as candidate candidate-ai-1/i)).toBeInTheDocument();
  });
});
