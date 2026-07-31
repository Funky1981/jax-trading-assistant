import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { HomePage } from './HomePage';
import { UserGuidePage } from './UserGuidePage';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';

vi.mock('@/hooks/useOperatorEvidenceOverview', () => ({ useOperatorEvidenceOverview: vi.fn() }));
vi.mock('@/lib/analytics', () => ({ emitAnalyticsEvent: vi.fn() }));

const safeData = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
  genuineEvents: 6,
  syntheticEvents: 2,
  rejectedEvents: 3,
  deduplicatedEvents: 0,
  candidates: 2,
  approvals: 1,
  paperTickets: 1,
  pendingCheckpoints: 1,
  completedCheckpoints: 2,
  missingDataCheckpoints: 0,
  ambiguousCheckpoints: 0,
  checkedAt: '2026-07-21T10:00:00Z',
};

function result(data: typeof safeData | undefined, overrides = {}) {
  return { data, isPending: false, isError: false, ...overrides } as ReturnType<
    typeof useOperatorEvidenceOverview
  >;
}

beforeEach(() => vi.mocked(useOperatorEvidenceOverview).mockReturnValue(result(safeData)));

describe('Beginner Phase 1 pages', () => {
  it('uses API-shaped safety and links Home cards without mutation controls', () => {
    render(
      <MemoryRouter>
        <HomePage />
      </MemoryRouter>,
    );
    expect(screen.getByRole('heading', { name: 'Jax overview' })).toBeInTheDocument();
    expect(screen.getAllByRole('status').some((node) => node.textContent?.includes('Paper-safe mode is on'))).toBe(true);
    expect(screen.getAllByRole('status').some((node) => node.textContent?.includes('Jax cannot place a real trade'))).toBe(true);
    expect(screen.getAllByRole('link', { name: 'Open Evidence Inbox' })[0]).toHaveAttribute(
      'href',
      '/monitor/inbox',
    );
    expect(screen.getByText('6')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /approve|order|execute/i }),
    ).not.toBeInTheDocument();
  });

  it.each([[{ ...safeData, allowLiveTrading: true }], [undefined]])(
    'warns when safety is unsafe or unknown',
    (data) => {
      vi.mocked(useOperatorEvidenceOverview).mockReturnValue(
        result(data as typeof safeData | undefined),
      );
      render(
        <MemoryRouter>
          <HomePage />
        </MemoryRouter>,
      );
      expect(screen.getByRole('status')).toHaveTextContent('cannot confirm a paper-safe state');
    },
  );

  it('explains capabilities, truthful workflow statuses, links and collapsed technical detail', () => {
    render(
      <MemoryRouter>
        <UserGuidePage />
      </MemoryRouter>,
    );
    expect(screen.getByRole('heading', { name: 'What Jax does today' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'What Jax cannot do' })).toBeInTheDocument();
    expect(screen.getByText(/Jax cannot trade live/i)).toBeInTheDocument();
    expect(screen.getByText('Review new evidence').closest('li')).toHaveTextContent('Ready');
    const destinations = screen.getByRole('navigation', { name: 'Guide destinations' });
    expect(within(destinations).getByRole('link', { name: 'Outcomes' })).toHaveAttribute(
      'href',
      '/outcomes',
    );
    expect(screen.getByText('Technical detail').closest('details')).not.toHaveAttribute('open');
  });

  it('does not invent completion when overview data is unavailable', () => {
    vi.mocked(useOperatorEvidenceOverview).mockReturnValue(result(undefined, { isError: true }));
    render(
      <MemoryRouter>
        <UserGuidePage />
      </MemoryRouter>,
    );
    expect(screen.queryByText('Done')).not.toBeInTheDocument();
    expect(screen.getAllByText('No data yet')).toHaveLength(3);
  });
});
