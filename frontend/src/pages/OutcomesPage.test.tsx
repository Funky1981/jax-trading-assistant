import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import { operatorEvidenceService } from '@/data/operator-evidence-service';
import { OutcomesPage } from './OutcomesPage';

vi.mock('@/data/operator-evidence-service', () => ({
  operatorEvidenceService: { candidates: vi.fn(), candidate: vi.fn() },
}));

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <OutcomesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe('OutcomesPage', () => {
  beforeEach(() => {
    vi.mocked(operatorEvidenceService.candidates).mockResolvedValue([
      {
        candidateId: 'candidate-1',
        symbol: 'QQQ',
        setupType: 'sector news momentum',
        candidateStatus: 'expired',
        humanDecision: 'approved',
        decisionProvenance: 'human',
        paperTicketId: 'paper-1',
        paperTicketStatus: 'paper_ticket_created',
        latestOutcomeStatus: 'pending_not_due',
        completedCheckpoints: 1,
        pendingCheckpoints: 1,
        missingCheckpoints: 0,
        ambiguousCheckpoints: 0,
        reason: 'Persisted evidence.',
        createdAt: '2026-07-17T16:46:00Z',
      },
    ]);
    vi.mocked(operatorEvidenceService.candidate).mockResolvedValue({
      evidenceStatus: 'sufficient',
      gateStatus: 'passed',
      riskStatus: 'passed',
      checkpoints: [
        {
          name: '1h',
          trackingStartedAt: '2026-07-17T16:46:00Z',
          trackingStartSource: 'paper_ticket_created_at',
          dueAt: '2026-07-17T17:46:00Z',
          observationAt: '2026-07-17T18:00:00Z',
          entryPrice: 500,
          checkpointPrice: 501,
          percentageReturn: 0.2,
          hypotheticalPnl: 10,
          maximumFavourableExcursion: 2,
          maximumAdverseExcursion: 1,
          targetTouched: true,
          stopTouched: false,
          firstTargetTouchAt: '2026-07-17T17:30:00Z',
          status: 'completed',
          dataQualityStatus: 'complete',
          marketDataSource: 'persisted_candles:alpaca',
          createdAt: '2026-07-17T16:46:00Z',
          updatedAt: '2026-07-17T18:00:00Z',
        },
        {
          name: '1w',
          trackingStartedAt: '2026-07-17T16:46:00Z',
          trackingStartSource: 'paper_ticket_created_at',
          dueAt: '2026-07-24T16:46:00Z',
          entryPrice: 500,
          targetTouched: false,
          stopTouched: false,
          status: 'pending_not_due',
          dataQualityStatus: 'not_due',
          createdAt: '2026-07-17T16:46:00Z',
          updatedAt: '2026-07-17T18:00:00Z',
        },
      ],
      selectedExecutionCounts: {},
      historicalExecutionCounts: {},
    });
  });

  it('shows persisted checkpoint states with visible hypothetical explanations', async () => {
    renderPage();
    expect(
      await screen.findByRole('heading', { name: 'Hypothetical Outcomes' }),
    ).toBeInTheDocument();
    expect(await screen.findByRole('heading', { name: '1 hour' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '1 week' })).toBeInTheDocument();
    expect(screen.getAllByText('PENDING — NOT DUE').length).toBeGreaterThan(0);
    expect(screen.getByText('persisted_candles:alpaca')).toBeInTheDocument();
    expect(screen.getAllByText('HYPOTHETICAL — NOT A FILL').length).toBeGreaterThan(0);
    expect(
      screen.getByText(/No order, fill, position or realised profit exists/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /approve|reject|execute|submit/i }),
    ).not.toBeInTheDocument();
  });

  it('has no detectable accessibility violations', async () => {
    const { container } = renderPage();
    await screen.findByRole('heading', { name: '1 hour' });
    expect((await axe(container)).violations).toHaveLength(0);
  });
});
