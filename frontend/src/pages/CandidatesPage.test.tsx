import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import {
  operatorEvidenceService,
  type OperatorCandidateSummary,
} from '@/data/operator-evidence-service';
import { CandidatesPage } from './CandidatesPage';

vi.mock('@/data/operator-evidence-service', () => ({
  operatorEvidenceService: { candidates: vi.fn() },
}));

function candidate(index: number): OperatorCandidateSummary {
  return {
    candidateId: `candidate-${index}`,
    symbol: index === 25 ? 'QQQ' : `SYM${index}`,
    setupType: 'sector news momentum',
    candidateStatus: index % 2 ? 'awaiting_approval' : 'approved',
    humanDecision: index % 2 ? '' : 'approved',
    decisionProvenance: index % 2 ? 'none' : 'human',
    paperTicketId: index % 3 === 0 ? `paper-${index}` : undefined,
    paperTicketStatus: index % 3 === 0 ? 'paper_ticket_created' : undefined,
    latestOutcomeStatus: index % 5 === 0 ? 'completed' : undefined,
    completedCheckpoints: index % 5 === 0 ? 1 : 0,
    pendingCheckpoints: 0,
    missingCheckpoints: 0,
    ambiguousCheckpoints: 0,
    reason: `Persisted reason ${index} that remains deliberately concise in the candidate list.`,
    createdAt: '2026-07-17T16:46:00Z',
  };
}

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
  beforeEach(() =>
    vi
      .mocked(operatorEvidenceService.candidates)
      .mockResolvedValue(Array.from({ length: 25 }, (_, i) => candidate(i + 1))),
  );

  it('renders only ten accessible candidate links by default and paginates safely', async () => {
    const user = userEvent.setup();
    renderPage();
    expect(await screen.findAllByRole('link', { name: /Open Candidate Review/i })).toHaveLength(10);
    expect(screen.getByText('1–10 of 25')).toBeInTheDocument();
    const previous = screen.getByRole('button', { name: /Previous candidate page/i });
    const next = screen.getByRole('button', { name: /Next candidate page/i });
    expect(previous).toBeDisabled();
    await user.click(next);
    expect(screen.getByText('11–20 of 25')).toBeInTheDocument();
    await user.click(previous);
    expect(screen.getByText('1–10 of 25')).toBeInTheDocument();
  });

  it('supports the twenty-item page size', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    await user.selectOptions(screen.getByRole('combobox', { name: /Candidates per page/i }), '20');
    expect(screen.getAllByRole('link', { name: /Open Candidate Review/i })).toHaveLength(20);
    expect(screen.getByText('1–20 of 25')).toBeInTheDocument();
  });

  it('searches by symbol, filters by state, and resets an out-of-range page', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    await user.click(screen.getByRole('button', { name: /Next candidate page/i }));
    await user.type(screen.getByRole('textbox', { name: /Search by symbol/i }), 'QQQ');
    expect(screen.getByText('1–1 of 1')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'QQQ' })).toBeInTheDocument();
    await user.clear(screen.getByRole('textbox', { name: /Search by symbol/i }));
    await user.click(screen.getByRole('button', { name: 'Needs review' }));
    expect(screen.getByRole('button', { name: 'Needs review' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.getByText('1–10 of 13')).toBeInTheDocument();
  });

  it('has no mutation controls or detectable accessibility violations', async () => {
    const { container } = renderPage();
    const links = await screen.findAllByRole('link', { name: /Open Candidate Review/i });
    expect(links[0]).toHaveAttribute('href', '/candidates/candidate-1/evidence');
    expect(
      within(container).queryByRole('button', {
        name: /^(approve|reject|execute|submit)$/i,
      }),
    ).not.toBeInTheDocument();
    expect((await axe(container)).violations).toHaveLength(0);
  });
});
