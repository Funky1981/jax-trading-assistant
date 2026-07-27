import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
    vi.clearAllMocks();
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
        completedCheckpoints: 2,
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
      paperTicketId: 'paper-1',
      entry: 500,
      stop: 495,
      target: 510,
      quantity: 5,
      plannedRisk: 25,
      plannedReward: 50,
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
          hypotheticalPnl: 5,
          maximumFavourableExcursion: 2,
          maximumAdverseExcursion: 1,
          targetTouched: false,
          stopTouched: false,
          status: 'completed',
          dataQualityStatus: 'complete',
          marketDataSource: 'persisted_candles:alpaca',
          createdAt: '2026-07-17T16:46:00Z',
          updatedAt: '2026-07-17T18:00:00Z',
        },
        {
          name: '1d',
          trackingStartedAt: '2026-07-17T16:46:00Z',
          trackingStartSource: 'paper_ticket_created_at',
          dueAt: '2026-07-18T16:46:00Z',
          observationAt: '2026-07-18T17:00:00Z',
          entryPrice: 500,
          checkpointPrice: 510,
          percentageReturn: 2,
          hypotheticalPnl: 50,
          targetTouched: true,
          stopTouched: false,
          status: 'completed',
          dataQualityStatus: 'complete',
          marketDataSource: 'persisted_candles:alpaca',
          createdAt: '2026-07-17T16:46:00Z',
          updatedAt: '2026-07-18T17:00:00Z',
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
          updatedAt: '2026-07-18T17:00:00Z',
        },
      ],
      selectedExecutionCounts: {},
      historicalExecutionCounts: {},
    });
  });

  it('starts with collapsed help and paper details and the latest completed checkpoint only', async () => {
    renderPage();
    expect(
      await screen.findByRole('heading', { name: 'Hypothetical Outcomes' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('How to understand these results').closest('details'),
    ).not.toHaveAttribute('open');
    expect(
      (await screen.findByText('Show paper-plan details')).closest('details'),
    ).not.toHaveAttribute('open');
    expect(screen.getByRole('tablist', { name: 'Outcome checkpoint' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /1 day/i })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('heading', { name: '1 day checkpoint' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '1 hour checkpoint' })).not.toBeInTheDocument();
    expect(screen.getByText('Show checkpoint details').closest('details')).not.toHaveAttribute(
      'open',
    );
  });

  it('changes the one fully displayed checkpoint and treats pending as pending', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByRole('heading', { name: '1 day checkpoint' });
    await user.click(screen.getByRole('tab', { name: /1 week/i }));
    expect(screen.getByRole('heading', { name: '1 week checkpoint' })).toBeInTheDocument();
    expect(screen.getByText('This checkpoint is not due yet.')).toBeInTheDocument();
    expect(
      screen.getByText('Primary values marked unavailable were not persisted for this checkpoint.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '1 day checkpoint' })).not.toBeInTheDocument();
  });

  it('keeps primary hypothetical metrics visible without duplicate badges or mutation controls', async () => {
    renderPage();
    await screen.findByRole('heading', { name: '1 day checkpoint' });
    expect(screen.getAllByText('Hypothetical return').some((node) => node.matches('p'))).toBe(true);
    expect(screen.getAllByText('Hypothetical P&L').some((node) => node.matches('p'))).toBe(true);
    expect(screen.getByText('persisted_candles:alpaca')).toBeVisible();
    const selectedCheckpoint = screen
      .getByRole('heading', { name: '1 day checkpoint' })
      .closest('section');
    expect(within(selectedCheckpoint!).getAllByText('Target touched')).toHaveLength(2);
    expect(screen.getByText(/not trades or realised profit and loss/i)).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /approve|reject|execute|submit/i }),
    ).not.toBeInTheDocument();
  });

  it('has no detectable accessibility violations', async () => {
    const { container } = renderPage();
    await screen.findByRole('heading', { name: '1 day checkpoint' });
    expect((await axe(container)).violations).toHaveLength(0);
  });
});
