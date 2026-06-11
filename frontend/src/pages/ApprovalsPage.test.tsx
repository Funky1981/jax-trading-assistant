import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { ApprovalsPage } from './ApprovalsPage';
import { approvalsService, candidatesService } from '@/data/approvals-service';
import { emitAnalyticsEvent } from '@/lib/analytics';

vi.mock('@/data/approvals-service', () => ({
  approvalsService: {
    getQueue: vi.fn(),
    approve: vi.fn(),
    reject: vi.fn(),
    snooze: vi.fn(),
    reanalyze: vi.fn(),
  },
  candidatesService: {
    list: vi.fn(),
    refresh: vi.fn(),
  },
}));

vi.mock('@/lib/analytics', () => ({
  emitAnalyticsEvent: vi.fn(),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ApprovalsPage />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('ApprovalsPage', () => {
  it('shows approval queue, execution activity, and blocked candidates', async () => {
    vi.mocked(emitAnalyticsEvent).mockClear();
    vi.mocked(approvalsService.getQueue).mockResolvedValue([
      {
        id: 'candidate-awaiting-1',
        symbol: 'AAPL',
        signalType: 'BUY',
        confidence: 0.82,
        detectedAt: '2026-03-19T13:00:00Z',
        instanceName: 'Opening Range Breakout',
        metadata: {
          etfPolicy: {
            allowed: true,
            reasonCode: 'allowed',
            reason: 'SPY is approved for ETF phase-1 paper trading.',
            catalogVersion: 'phase1-2026-05-13',
          },
        },
      },
    ]);
    vi.mocked(candidatesService.list)
      .mockResolvedValueOnce([
        {
          id: 'candidate-blocked-1',
          strategyInstanceId: 'instance-1',
          signalId: 'signal-1',
          artifactId: 'artifact-1',
          strategyId: 'orb',
          symbol: 'MSFT',
          signalType: 'SELL',
          status: 'blocked',
          blockedReasonCode: 'low_confidence',
          blockReason: 'Confidence was below threshold.',
          sessionDate: '2026-03-19',
          detectedAt: '2026-03-19T12:45:00Z',
          blockedAt: '2026-03-19T12:46:00Z',
          dataProvenance: 'watcher:test',
        },
      ] as never)
      .mockResolvedValueOnce([
        {
          id: 'candidate-filled-1',
          strategyInstanceId: 'instance-2',
          symbol: 'NVDA',
          signalType: 'BUY',
          status: 'filled',
          sessionDate: '2026-03-19',
          detectedAt: '2026-03-19T12:00:00Z',
          submittedAt: '2026-03-19T12:01:00Z',
          filledAt: '2026-03-19T12:02:00Z',
          dataProvenance: 'watcher:test',
          latestApproval: {
            id: 'approval-1',
            decision: 'approved',
            approvedBy: 'operator',
            decidedAt: '2026-03-19T12:00:30Z',
          },
          executionInstructionId: 'instruction-1',
          tradeId: 'trade-1',
        },
      ] as never)
      .mockResolvedValueOnce([] as never)
      .mockResolvedValueOnce([] as never);

    renderPage();

    expect(await screen.findByText('Approval Queue')).toBeInTheDocument();
    expect(await screen.findByText('AAPL')).toBeInTheDocument();
    expect(await screen.findByText('ETF eligible')).toBeInTheDocument();
    expect(await screen.findByText('allowed')).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /Approve for paper order/i })).toBeInTheDocument();
    expect(await screen.findByText('Recent Execution Activity')).toBeInTheDocument();
    expect(await screen.findByText('NVDA')).toBeInTheDocument();
    expect(await screen.findByText('Recently Blocked')).toBeInTheDocument();
    expect(await screen.findByText('Confidence was below threshold.')).toBeInTheDocument();
    expect(await screen.findByText('low_confidence: 1')).toBeInTheDocument();
  });

  it('wires blocked refresh action to candidate refresh and mobile queue producer path', async () => {
    vi.mocked(emitAnalyticsEvent).mockClear();
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);
    vi.mocked(candidatesService.list)
      .mockResolvedValue([] as never)
      .mockResolvedValueOnce([
        {
          id: 'candidate-blocked-2',
          strategyInstanceId: 'instance-3',
          signalId: 'signal-3',
          artifactId: 'artifact-3',
          strategyId: 'etf_orb',
          symbol: 'SPY',
          signalType: 'BUY',
          status: 'blocked',
          blockedReasonCode: 'priced_in_high',
          blockReason: 'Move appears mostly priced in.',
          sessionDate: '2026-03-20',
          detectedAt: '2026-03-20T10:00:00Z',
          blockedAt: '2026-03-20T10:01:00Z',
          dataProvenance: 'watcher:test',
        },
      ] as never)
      .mockResolvedValueOnce([] as never)
      .mockResolvedValueOnce([] as never)
      .mockResolvedValueOnce([] as never);
    vi.mocked(candidatesService.refresh).mockResolvedValue({
      id: 'candidate-blocked-2',
      strategyInstanceId: 'instance-3',
      symbol: 'SPY',
      signalType: 'BUY',
      status: 'awaiting_approval',
      sessionDate: '2026-03-20',
      detectedAt: '2026-03-20T10:00:00Z',
      dataProvenance: 'watcher:test',
    } as never);

    renderPage();

    const evidenceLink = await screen.findByRole('link', { name: 'Evidence' });
    expect(evidenceLink).toHaveAttribute('href', '/candidates/candidate-blocked-2/evidence');

    fireEvent.click(evidenceLink);
    expect(vi.mocked(emitAnalyticsEvent)).toHaveBeenCalledWith(
      'approval_sentiment_evidence_viewed',
      expect.objectContaining({ source_surface: 'approvals', candidate_id: 'candidate-blocked-2' })
    );

    fireEvent.click(screen.getByRole('button', { name: /Re-qualify & Queue Mobile/i }));

    await waitFor(() => {
      expect(candidatesService.refresh).toHaveBeenCalledWith('candidate-blocked-2');
    });
  });
});
