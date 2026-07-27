import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { OperatorEvidenceOverview } from '@/data/operator-evidence-service';
import { SystemPage } from './SystemPage';

const overview = vi.fn();
const candidate = vi.fn();
vi.mock('@/hooks/useOperatorEvidenceOverview', () => ({
  useOperatorEvidenceOverview: () => overview(),
}));
vi.mock('@/data/operator-evidence-service', () => ({
  operatorEvidenceService: { candidate: (...args: unknown[]) => candidate(...args) },
}));
vi.mock('@/data/events-service', () => ({
  eventsService: { list: vi.fn().mockResolvedValue({ events: [] }) },
}));
vi.mock('@/data/datasets-service', () => ({
  datasetsService: { list: vi.fn().mockResolvedValue({ datasets: [] }) },
}));
vi.mock('@/components/dashboard', () => ({
  HealthPanel: () => <div>Health detail</div>,
  MetricsPanel: () => <div>Metrics detail</div>,
  MemoryBrowserPanel: () => <div>Memory detail</div>,
}));

const data = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
  approvals: 1,
  paperTickets: 1,
  historicalExecutionInstructions: 1,
  historicalOrderIntents: 0,
  historicalBrokerOrders: 0,
  historicalTrades: 0,
  historicalFills: 0,
  checkedAt: '2026-07-27T12:00:00Z',
} as OperatorEvidenceOverview;

function renderPage(path = '/system') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <SystemPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe('System Safety', () => {
  beforeEach(() => {
    overview.mockReturnValue({ data, isPending: false, isError: false });
    candidate.mockReset();
  });

  it('renders the safe state, six cards, separate history and collapsed diagnostics', () => {
    renderPage();
    expect(screen.getByRole('heading', { level: 1, name: 'System Safety' })).toBeInTheDocument();
    expect(
      screen.getByText(
        'Paper-safe mode is on. Live trading, execution and broker activity are disabled.',
      ),
    ).toBeInTheDocument();
    expect(screen.getAllByText('Safe')).toHaveLength(6);
    expect(
      screen.getByText(
        'No specific candidate journey is selected. Global historical records are shown separately below.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Historical records' })).toBeInTheDocument();
    expect(screen.getByText('Technical diagnostics').closest('details')).not.toHaveAttribute(
      'open',
    );
    expect(
      screen.queryByRole('button', { name: /enable|start|delete|clear|restart/i }),
    ).not.toBeInTheDocument();
  });

  it('keeps selected-journey counts separate from the unrelated historical instruction', async () => {
    candidate.mockResolvedValue({
      selectedExecutionCounts: {
        executionInstructions: 0,
        orderIntents: 0,
        brokerOrders: 0,
        trades: 0,
        fills: 0,
      },
    });
    renderPage('/system?candidateId=qqq-candidate');
    expect(
      await screen.findByText('This journey created no execution records.'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'These are database-wide historical records and may be unrelated to the candidate or evidence item currently being reviewed.',
      ),
    ).toBeInTheDocument();
  });

  it('expands technical diagnostics by keyboard activation', () => {
    renderPage();
    const summary = screen.getByText('Technical diagnostics');
    fireEvent.click(summary);
    expect(summary.closest('details')).toHaveAttribute('open');
    expect(screen.getByText('ALLOW_LIVE_TRADING')).toBeInTheDocument();
  });
});
