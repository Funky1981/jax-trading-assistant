import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi } from 'vitest';
import { BlotterPage } from './BlotterPage';
import { candidatesService } from '@/data/approvals-service';

vi.mock('@/data/approvals-service', () => ({
  candidatesService: {
    list: vi.fn(),
  },
}));

vi.mock('@/components/dashboard', () => ({
  TradeBlotterPanel: () => <div>Trade Blotter Panel</div>,
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <BlotterPage />
    </QueryClientProvider>
  );
}

describe('BlotterPage', () => {
  it('shows execution chain context above the blotter', async () => {
    vi.mocked(candidatesService.list)
      .mockResolvedValueOnce([
        {
          id: 'candidate-approved-1',
          strategyInstanceId: 'instance-1',
          symbol: 'AAPL',
          signalType: 'BUY',
          status: 'approved',
          sessionDate: '2026-03-19',
          detectedAt: '2026-03-19T10:00:00Z',
          dataProvenance: 'watcher:test',
          latestApproval: {
            id: 'approval-1',
            decision: 'approved',
            approvedBy: 'operator',
            decidedAt: '2026-03-19T10:01:00Z',
          },
          executionInstructionId: 'instruction-1',
        },
      ] as never)
      .mockResolvedValueOnce([
        {
          id: 'candidate-submitted-1',
          strategyInstanceId: 'instance-2',
          symbol: 'MSFT',
          signalType: 'SELL',
          status: 'submitted',
          sessionDate: '2026-03-19',
          detectedAt: '2026-03-19T10:05:00Z',
          submittedAt: '2026-03-19T10:06:00Z',
          dataProvenance: 'watcher:test',
          latestApproval: {
            id: 'approval-2',
            decision: 'approved',
            approvedBy: 'operator',
            decidedAt: '2026-03-19T10:05:30Z',
          },
          executionInstructionId: 'instruction-2',
        },
      ] as never)
      .mockResolvedValueOnce([
        {
          id: 'candidate-filled-1',
          strategyInstanceId: 'instance-3',
          symbol: 'NVDA',
          signalType: 'BUY',
          status: 'filled',
          sessionDate: '2026-03-19',
          detectedAt: '2026-03-19T10:10:00Z',
          submittedAt: '2026-03-19T10:11:00Z',
          filledAt: '2026-03-19T10:12:00Z',
          dataProvenance: 'watcher:test',
          latestApproval: {
            id: 'approval-3',
            decision: 'approved',
            approvedBy: 'operator',
            decidedAt: '2026-03-19T10:10:30Z',
          },
          executionInstructionId: 'instruction-3',
          tradeId: 'trade-3',
        },
      ] as never);

    renderPage();

    expect(await screen.findByText('Paper Execution Chain')).toBeInTheDocument();
    expect(await screen.findByText('Approved')).toBeInTheDocument();
    expect(await screen.findByText('Submitted')).toBeInTheDocument();
    expect(await screen.findByText('Filled')).toBeInTheDocument();
    expect(await screen.findByText('NVDA')).toBeInTheDocument();
    expect(await screen.findByText('Trade Blotter Panel')).toBeInTheDocument();
  });
});
