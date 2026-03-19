import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { ApprovalsPage } from './ApprovalsPage';
import { approvalsService, candidatesService } from '@/data/approvals-service';

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
  },
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <ApprovalsPage />
    </QueryClientProvider>
  );
}

describe('ApprovalsPage', () => {
  it('shows approval queue, execution activity, and blocked candidates', async () => {
    vi.mocked(approvalsService.getQueue).mockResolvedValue([
      {
        id: 'candidate-awaiting-1',
        symbol: 'AAPL',
        signalType: 'BUY',
        confidence: 0.82,
        detectedAt: '2026-03-19T13:00:00Z',
        instanceName: 'Opening Range Breakout',
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
    expect(await screen.findByText('Recent Execution Activity')).toBeInTheDocument();
    expect(await screen.findByText('NVDA')).toBeInTheDocument();
    expect(await screen.findByText('Recently Blocked')).toBeInTheDocument();
    expect(await screen.findByText('Confidence was below threshold.')).toBeInTheDocument();
    expect(await screen.findByText('low_confidence: 1')).toBeInTheDocument();
  });
});
