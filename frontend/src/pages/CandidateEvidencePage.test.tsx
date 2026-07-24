import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { candidatesService } from '@/data/approvals-service';
import { CandidateEvidencePage } from './CandidateEvidencePage';
import { operatorEvidenceService } from '@/data/operator-evidence-service';

vi.mock('@/data/approvals-service', async () => {
  const actual = await vi.importActual<typeof import('@/data/approvals-service')>(
    '@/data/approvals-service',
  );
  return {
    ...actual,
    candidatesService: {
      get: vi.fn(),
    },
  };
});
vi.mock('@/data/operator-evidence-service', () => ({
  operatorEvidenceService: { candidate: vi.fn() },
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <MemoryRouter initialEntries={['/candidates/candidate-1/evidence']}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route path="/candidates/:candidateId/evidence" element={<CandidateEvidencePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe('CandidateEvidencePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(candidatesService.get).mockResolvedValue({
      id: 'candidate-1',
      strategyInstanceId: 'instance-1',
      symbol: 'SPY',
      signalType: 'BUY',
      status: 'awaiting_approval',
      entryPrice: 530,
      stopLoss: 527.5,
      takeProfit: 536,
      confidence: 0.84,
      reasoning:
        'Softer inflation news supports a tactical SPY paper long while price holds above trend.',
      sessionDate: '2026-06-12',
      detectedAt: '2026-06-12T10:30:00Z',
      expiresAt: '2026-06-12T11:00:00Z',
      dataProvenance: 'world-monitor',
      metadata: {
        worldMonitor: {
          source: 'world-monitor',
          sourceEventId: 'wm-1',
          headline: 'Inflation cools more than expected',
          summary: 'Treasury yields moved lower after the inflation print.',
          eventType: 'macro_rates',
          sourceURLs: ['https://example.com/inflation', 'https://example.com/yields'],
          sourceCount: 2,
          assetThemes: ['rates', 'growth'],
          confidenceReasons: ['trusted macro source', 'mapped to SPY'],
          mappingReason:
            'SPY was selected because broad US equities often react to lower-rate surprises.',
          route: 'approval_required',
        },
        chartConfirmation: {
          confirmed: true,
          reasonCode: 'above_sma20',
          reason:
            'SPY held above the 20-period moving average and the last five candles were positive.',
          candleCount: 30,
          lastClose: 531.25,
          sma20: 528.1,
          fiveCandleChangePct: 0.012,
          checkedAt: '2026-06-12T10:31:00Z',
        },
      },
      sentiment: {
        label: 'positive',
        state: 'available',
        score: 0.68,
        confidence: 0.74,
        window: '24h',
        sourceCount: 2,
        priceAgreement: 'agreeing',
        topDrivers: ['Lower yields supported equity risk appetite.'],
        limitations: ['Only two trusted sources were available.'],
        summary: 'News tone is positive for broad US equity exposure.',
        snapshotAt: '2026-06-12T10:32:00Z',
      },
    });
    vi.mocked(operatorEvidenceService.candidate).mockResolvedValue({
      evidenceScore: 0.81,
      evidenceStatus: 'sufficient',
      gateStatus: 'passed',
      riskStatus: 'passed',
      approvalId: 'approval-1',
      approvalDecision: 'approved',
      approvedBy: 'operator',
      approvalReason: 'Paper evidence accepted',
      approvalAt: '2026-06-12T10:40:00Z',
      paperTicketId: 'paper-1',
      paperTicketStatus: 'paper_ticket_created',
      entry: 530,
      stop: 527.5,
      target: 536,
      quantity: 40,
      plannedRisk: 125,
      plannedReward: 240,
      rewardRisk: 2.4,
      notional: 21200,
      checkpoints: [
        {
          name: '1h',
          trackingStartedAt: '2026-06-12T10:40:00Z',
          trackingStartSource: 'approval',
          dueAt: '2026-06-12T11:40:00Z',
          entryPrice: 530,
          status: 'pending_not_due',
          dataQualityStatus: 'not_due',
          targetTouched: false,
          stopTouched: false,
          createdAt: '2026-06-12T10:40:00Z',
          updatedAt: '2026-06-12T10:40:00Z',
        },
      ],
      selectedExecutionCounts: {
        executionInstructions: 0,
        orderIntents: 0,
        brokerOrders: 0,
        trades: 0,
        fills: 0,
      },
      historicalExecutionCounts: { executionInstructions: 3 },
    });
  });

  it('shows the complete trade evidence needed before approval or manual review', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { name: 'Candidate Review' })).toBeInTheDocument();
    expect(
      screen.getByText('Review only — this page cannot place an order or create a fill.'),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Softer inflation news supports a tactical SPY paper long/i),
    ).toBeInTheDocument();
    expect(screen.getAllByText(/SPY was selected because broad US equities/i).length).toBeGreaterThan(0);

    expect(screen.getByRole('heading', { name: /Why Jax considered it/i })).toBeInTheDocument();
    expect(screen.getByText('Inflation cools more than expected')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /example.com\/inflation/i })).toHaveAttribute(
      'href',
      'https://example.com/inflation',
    );
    expect(screen.getByRole('link', { name: /example.com\/yields/i })).toHaveAttribute(
      'href',
      'https://example.com/yields',
    );

    expect(
      screen.getByRole('heading', { name: /What the charts are saying/i }),
    ).toBeInTheDocument();
    expect(screen.getByText('Chart confirmed')).toBeInTheDocument();
    expect(screen.getAllByText(/held above the 20-period moving average/i).length).toBeGreaterThan(0);
    expect(screen.getByText('$531.25')).toBeInTheDocument();

    expect(screen.getByRole('heading', { name: 'Sentiment and source' })).toBeInTheDocument();
    expect(
      screen.getByText('News tone is positive for broad US equity exposure.'),
    ).toBeInTheDocument();
    expect(screen.getByText('Lower yields supported equity risk appetite.')).toBeInTheDocument();
    expect(screen.getByText('Only two trusted sources were available.')).toBeInTheDocument();

    expect(
      screen.getByRole('heading', { name: 'Persisted paper sizing evidence' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Hypothetical paper plan' })).toBeInTheDocument();
    expect(screen.getByText('No persisted sizing evidence available')).toBeInTheDocument();
    expect(screen.getByText('$21,200.00')).toBeInTheDocument();
    expect(screen.getByText('$125.00')).toBeInTheDocument();
    expect(screen.getAllByText('2.40R').length).toBeGreaterThan(0);
    expect(screen.getAllByText('PENDING — NOT DUE').length).toBeGreaterThan(0);
    expect(screen.getByText('NO FILL OCCURRED')).toBeInTheDocument();
    expect(screen.getAllByText('0').length).toBeGreaterThanOrEqual(5);
    expect(screen.getByText('Audit details')).toBeInTheDocument();
    expect(
      screen.getByText(/This candidate journey created no execution records/i),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /approve|reject|execute|submit/i }),
    ).not.toBeInTheDocument();
  });

  it('does not invent sizing when no persisted sizing exists', async () => {
    const candidate = await candidatesService.get('candidate-1');
    vi.mocked(candidatesService.get).mockResolvedValue({
      ...candidate,
      metadata: { ...candidate.metadata, sizing: undefined },
    });
    renderPage();
    expect(await screen.findByText('No persisted sizing evidence available')).toBeInTheDocument();
  });
});
