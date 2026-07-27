import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import { candidatesService } from '@/data/approvals-service';
import { operatorEvidenceService } from '@/data/operator-evidence-service';
import { CandidateEvidencePage } from './CandidateEvidencePage';

vi.mock('@/data/approvals-service', async () => ({
  ...(await vi.importActual('@/data/approvals-service')),
  candidatesService: { get: vi.fn() },
}));
vi.mock('@/data/operator-evidence-service', () => ({
  operatorEvidenceService: { candidate: vi.fn() },
}));

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={['/candidates/candidate-1/evidence']}>
      <QueryClientProvider client={client}>
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
      strategyId: 'etf_news_sector_momentum_v1',
      symbol: 'SPY',
      signalType: 'BUY',
      status: 'awaiting_approval',
      entryPrice: 530,
      stopLoss: 527.5,
      takeProfit: 536,
      confidence: 0.84,
      reasoning: 'Softer inflation news supports a tactical SPY paper long.',
      sessionDate: '2026-06-12',
      detectedAt: '2026-06-12T10:30:00Z',
      expiresAt: '2026-06-12T11:00:00Z',
      dataProvenance: 'world-monitor',
      metadata: {
        worldMonitor: {
          source: 'world-monitor',
          sourceEventId: 'wm-1',
          headline: 'Inflation cools more than expected',
          summary: 'Treasury yields moved lower.',
          sourceURLs: ['https://example.com/inflation'],
          assetThemes: ['rates', 'growth'],
          mappingReason: 'Broad equities react to lower-rate surprises.',
        },
        chartConfirmation: {
          confirmed: true,
          reason: 'SPY held above its moving average.',
          lastClose: 531.25,
          sma20: 528.1,
          fiveCandleChangePct: 0.012,
        },
      },
      sentiment: {
        label: 'positive',
        state: 'available',
        score: 0.68,
        confidence: 0.74,
        summary: 'News tone is positive.',
        limitations: ['Only two trusted sources were available.'],
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
      leverage: 1.2,
      notional: 21200,
      checkpoints: [
        {
          name: '1h',
          trackingStartedAt: '2026-06-12T10:40:00Z',
          trackingStartSource: 'approval',
          dueAt: '2026-06-12T11:40:00Z',
          entryPrice: 530,
          status: 'completed',
          dataQualityStatus: 'complete',
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

  it('opens on Overview only and keeps the top safety statement visible', async () => {
    renderPage();
    expect(await screen.findByRole('heading', { name: 'Candidate Review' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Overview' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByText('Review only — no order or fill exists.')).toBeInTheDocument();
    expect(screen.getByText('Why Jax created it')).toBeInTheDocument();
    expect(screen.queryByText('Inflation cools more than expected')).not.toBeInTheDocument();
    expect(
      screen.queryByText('HYPOTHETICAL PAPER PLAN — NOT AN ORDER OR FILL'),
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Selected-journey execution counts')).not.toBeInTheDocument();
  });

  it('supports keyboard tab navigation and truthfully groups missing evidence', async () => {
    const user = userEvent.setup();
    const base = await candidatesService.get('candidate-1');
    vi.mocked(candidatesService.get).mockResolvedValue({
      ...base,
      sentiment: undefined,
      metadata: { ...base.metadata, chartConfirmation: undefined },
    });
    renderPage();
    const overview = await screen.findByRole('tab', { name: 'Overview' });
    overview.focus();
    await user.keyboard('{ArrowRight}');
    expect(screen.getByRole('tab', { name: 'Evidence' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByText('Inflation cools more than expected')).toBeInTheDocument();
    expect(
      screen.getByText(
        /Additional evidence not recorded: chart confirmation and sentiment analysis/i,
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('Review only — no order or fill exists.')).toBeInTheDocument();
  });

  it('shows primary plan values and keeps secondary assumptions collapsed', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('tab', { name: 'Paper Plan' }));
    expect(screen.getByText('$530.00')).toBeInTheDocument();
    expect(screen.getByText('$125.00')).toBeInTheDocument();
    expect(screen.getByText('2.40R')).toBeInTheDocument();
    const disclosure = screen.getByText('Show all paper-plan details');
    expect(disclosure.closest('details')).not.toHaveAttribute('open');
    expect(screen.getByText('$21,200.00')).not.toBeVisible();
  });

  it('keeps Outcomes compact and exposes audit IDs and counts only after selection', async () => {
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole('tab', { name: 'Outcomes' }));
    expect(screen.getByRole('link', { name: 'Open Hypothetical Outcomes' })).toHaveAttribute(
      'href',
      '/outcomes',
    );
    expect(screen.queryByText('approval-1')).not.toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: 'Audit' }));
    expect(screen.getByText('approval-1')).toBeInTheDocument();
    expect(screen.getByText('wm-1')).toBeInTheDocument();
    expect(screen.getByText('Selected-journey execution counts')).toBeInTheDocument();
    expect(screen.getByText('Show raw metadata').closest('details')).not.toHaveAttribute('open');
    expect(
      screen.queryByRole('button', { name: /approve|reject|execute|submit/i }),
    ).not.toBeInTheDocument();
  });

  it('has no detectable accessibility violations', async () => {
    const { container } = renderPage();
    await screen.findByRole('tab', { name: 'Overview' });
    expect((await axe(container)).violations).toHaveLength(0);
  });
});
