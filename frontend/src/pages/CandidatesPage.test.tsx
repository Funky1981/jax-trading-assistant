import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import { operatorEvidenceService } from '@/data/operator-evidence-service';
import { CandidatesPage } from './CandidatesPage';

vi.mock('@/data/operator-evidence-service', () => ({
  operatorEvidenceService: { candidates: vi.fn() },
}));

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <CandidatesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe('CandidatesPage', () => {
  beforeEach(() => {
    vi.mocked(operatorEvidenceService.candidates).mockResolvedValue([
      {
        candidateId: 'candidate-1',
        symbol: 'QQQ',
        setupType: 'sector news momentum',
        candidateStatus: 'approved',
        humanDecision: 'approved',
        decisionProvenance: 'human',
        paperTicketId: 'paper-1',
        paperTicketStatus: 'paper_ticket_created',
        latestOutcomeStatus: 'pending_not_due',
        completedCheckpoints: 2,
        pendingCheckpoints: 1,
        missingCheckpoints: 0,
        ambiguousCheckpoints: 0,
        reason: 'Persisted macro evidence affected the QQQ research outlook.',
        createdAt: '2026-07-17T16:46:00Z',
        expiresAt: '2026-07-20T16:46:00Z',
      },
    ]);
  });

  it('shows the compatible Candidates route as a read-only evidence list', async () => {
    renderPage();
    expect(await screen.findByRole('heading', { name: 'Candidates' })).toBeInTheDocument();
    expect((await screen.findAllByText('Outcomes available')).length).toBeGreaterThan(0);
    expect(screen.getByText('Human approved')).toBeInTheDocument();
    expect(screen.getByText('Persisted paper plan available')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Open Candidate Review/i })).toHaveAttribute(
      'href',
      '/candidates/candidate-1/evidence',
    );
    expect(
      screen.queryByRole('button', { name: /approve|reject|execute|submit/i }),
    ).not.toBeInTheDocument();
  });

  it('has no detectable accessibility violations', async () => {
    const { container } = renderPage();
    await screen.findByRole('link', { name: /Open Candidate Review/i });
    expect((await axe(container)).violations).toHaveLength(0);
  });
});
