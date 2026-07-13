import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { ApprovalsPage } from './ApprovalsPage';
import { approvalsService, candidatesService } from '@/data/approvals-service';
import { emitAnalyticsEvent } from '@/lib/analytics';

vi.mock('@/data/approvals-service', () => ({
  approvalsService: {
    getQueue: vi.fn(),
    approve: vi.fn(),
    reject: vi.fn(),
    snooze: vi.fn(),
    reanalyze: vi.fn(),
    getPaperTicketQueue: vi.fn(),
    markPaperTicketReviewed: vi.fn(),
    cancelPaperTicket: vi.fn(),
    addPaperTicketNote: vi.fn(),
  },
  candidatesService: {
    list: vi.fn(),
    refresh: vi.fn(),
  },
}));

vi.mock('@/lib/analytics', () => ({
  emitAnalyticsEvent: vi.fn(),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ApprovalsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe('ApprovalsPage', () => {
  it('shows approval queue, execution activity, and blocked candidates', async () => {
    vi.mocked(emitAnalyticsEvent).mockClear();
    vi.mocked(approvalsService.getQueue).mockResolvedValue([
      {
        id: 'candidate-awaiting-1',
        symbol: 'AAPL',
        signalType: 'BUY',
        confidence: 0.82,
        detectedAt: '2026-03-19T13:00:00Z',
        instanceName: 'Opening Range Breakout',
        metadata: {
          etfPolicy: {
            allowed: true,
            reasonCode: 'allowed',
            reason: 'SPY is approved for ETF phase-1 paper trading.',
            catalogVersion: 'phase1-2026-05-13',
          },
        },
      },
    ]);
    vi.mocked(approvalsService.getPaperTicketQueue).mockResolvedValue([]);
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
    expect(await screen.findByText('ETF eligible')).toBeInTheDocument();
    expect(await screen.findByText('allowed')).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: /Approve for paper order/i }),
    ).toBeInTheDocument();
    expect(await screen.findByText('Recent Execution Activity')).toBeInTheDocument();
    expect(await screen.findByText('NVDA')).toBeInTheDocument();
    expect(await screen.findByText('Recently Blocked')).toBeInTheDocument();
    expect(await screen.findByText('Confidence was below threshold.')).toBeInTheDocument();
    expect(await screen.findByText('low_confidence: 1')).toBeInTheDocument();
  });

  it('wires blocked refresh action to candidate refresh and mobile queue producer path', async () => {
    vi.mocked(emitAnalyticsEvent).mockClear();
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);
    vi.mocked(approvalsService.getPaperTicketQueue).mockResolvedValue([]);
    vi.mocked(candidatesService.list)
      .mockResolvedValue([] as never)
      .mockResolvedValueOnce([
        {
          id: 'candidate-blocked-2',
          strategyInstanceId: 'instance-3',
          signalId: 'signal-3',
          artifactId: 'artifact-3',
          strategyId: 'etf_orb',
          symbol: 'SPY',
          signalType: 'BUY',
          status: 'blocked',
          blockedReasonCode: 'priced_in_high',
          blockReason: 'Move appears mostly priced in.',
          sessionDate: '2026-03-20',
          detectedAt: '2026-03-20T10:00:00Z',
          blockedAt: '2026-03-20T10:01:00Z',
          dataProvenance: 'watcher:test',
        },
      ] as never)
      .mockResolvedValueOnce([] as never)
      .mockResolvedValueOnce([] as never)
      .mockResolvedValueOnce([] as never);
    vi.mocked(candidatesService.refresh).mockResolvedValue({
      id: 'candidate-blocked-2',
      strategyInstanceId: 'instance-3',
      symbol: 'SPY',
      signalType: 'BUY',
      status: 'awaiting_approval',
      sessionDate: '2026-03-20',
      detectedAt: '2026-03-20T10:00:00Z',
      dataProvenance: 'watcher:test',
    } as never);

    renderPage();

    const evidenceLink = await screen.findByRole('link', { name: 'Evidence' });
    expect(evidenceLink).toHaveAttribute('href', '/candidates/candidate-blocked-2/evidence');

    fireEvent.click(evidenceLink);
    expect(vi.mocked(emitAnalyticsEvent)).toHaveBeenCalledWith(
      'approval_sentiment_evidence_viewed',
      expect.objectContaining({ source_surface: 'approvals', candidate_id: 'candidate-blocked-2' }),
    );

    fireEvent.click(screen.getByRole('button', { name: /Re-qualify & Queue Mobile/i }));

    await waitFor(() => {
      expect(candidatesService.refresh).toHaveBeenCalledWith('candidate-blocked-2');
    });
  });

  it('shows readable paper ticket review cards with safe review actions only', async () => {
    vi.mocked(emitAnalyticsEvent).mockClear();
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);
    vi.mocked(approvalsService.getPaperTicketQueue).mockResolvedValue([
      {
        paperTicketId: 'pt_candidate_1',
        candidateId: 'candidate-paper-1',
        createdAt: '2026-07-13T08:00:00Z',
        updatedAt: '2026-07-13T08:05:00Z',
        status: 'paper_ticket_created',
        symbol: 'SPY',
        direction: 'long',
        setupType: 'pullback_continuation',
        catalystSummary: 'Broad-market ETF holding above support.',
        entryPrice: 101.25,
        stopLossPrice: 98.75,
        targetPrice: 107,
        positionSize: 4,
        maxNormalLoss: 10,
        maxSlippageAdjustedLoss: 12,
        rewardRiskRatio: 2.3,
        evidenceStatus: 'sufficient',
        gateStatus: 'ready_for_risk_review',
        riskStatus: 'ready_for_approval_review',
        approvalStatus: 'paper_ticket_ready',
        paperOnly: true,
        rejectReasons: ['risk_review_pending'],
        warningReasons: ['spread_check_needed'],
        reviewNotes: 'Review again after the open.',
      },
      {
        paperTicketId: 'pt_reviewed',
        candidateId: 'candidate-reviewed',
        createdAt: '2026-07-13T07:00:00Z',
        updatedAt: '2026-07-13T07:15:00Z',
        status: 'paper_ticket_reviewed',
        symbol: 'QQQ',
        direction: 'short',
        setupType: 'failed_breakout',
        catalystSummary: 'Weak tech follow-through.',
        entryPrice: 402,
        stopLossPrice: 406,
        targetPrice: 392,
        positionSize: 2,
        maxNormalLoss: 8,
        maxSlippageAdjustedLoss: 9.5,
        rewardRiskRatio: 2.5,
        evidenceStatus: 'sufficient',
        gateStatus: 'passed',
        riskStatus: 'ready_for_approval_review',
        approvalStatus: 'paper_ticket_ready',
        paperOnly: true,
      },
      {
        paperTicketId: 'pt_cancelled',
        candidateId: 'candidate-cancelled',
        createdAt: '2026-07-13T06:00:00Z',
        updatedAt: '2026-07-13T06:20:00Z',
        status: 'paper_ticket_cancelled',
        symbol: 'IWM',
        direction: 'long',
        setupType: 'range_reclaim',
        catalystSummary: 'Small caps lost confirmation.',
        entryPrice: 210,
        stopLossPrice: 207,
        targetPrice: 216,
        positionSize: 3,
        maxNormalLoss: 9,
        maxSlippageAdjustedLoss: 11,
        rewardRiskRatio: 2,
        evidenceStatus: 'insufficient',
        gateStatus: 'blocked',
        riskStatus: 'cancelled',
        approvalStatus: 'paper_ticket_cancelled',
        paperOnly: true,
      },
    ]);
    vi.mocked(approvalsService.markPaperTicketReviewed).mockResolvedValue({
      paperTicketId: 'pt_candidate_1',
      candidateId: 'candidate-paper-1',
      createdAt: '2026-07-13T08:00:00Z',
      updatedAt: '2026-07-13T08:06:00Z',
      status: 'paper_ticket_reviewed',
      symbol: 'SPY',
      direction: 'long',
      setupType: 'pullback_continuation',
      catalystSummary: 'Broad-market ETF holding above support.',
      entryPrice: 101.25,
      stopLossPrice: 98.75,
      targetPrice: 107,
      positionSize: 4,
      maxNormalLoss: 10,
      maxSlippageAdjustedLoss: 12,
      rewardRiskRatio: 2.3,
      evidenceStatus: 'sufficient',
      gateStatus: 'ready_for_risk_review',
      riskStatus: 'ready_for_approval_review',
      approvalStatus: 'paper_ticket_ready',
      paperOnly: true,
    });
    vi.mocked(approvalsService.addPaperTicketNote).mockResolvedValue({
      paperTicketId: 'pt_candidate_1',
      candidateId: 'candidate-paper-1',
      createdAt: '2026-07-13T08:00:00Z',
      updatedAt: '2026-07-13T08:07:00Z',
      status: 'paper_ticket_created',
      symbol: 'SPY',
      direction: 'long',
      setupType: 'pullback_continuation',
      catalystSummary: 'Broad-market ETF holding above support.',
      entryPrice: 101.25,
      stopLossPrice: 98.75,
      targetPrice: 107,
      positionSize: 4,
      maxNormalLoss: 10,
      maxSlippageAdjustedLoss: 12,
      rewardRiskRatio: 2.3,
      evidenceStatus: 'sufficient',
      gateStatus: 'ready_for_risk_review',
      riskStatus: 'ready_for_approval_review',
      approvalStatus: 'paper_ticket_ready',
      paperOnly: true,
      reviewNotes: 'Review again after the open.\nBeginner-safe note.',
    });
    vi.mocked(candidatesService.list).mockResolvedValue([] as never);

    renderPage();

    expect(await screen.findByText('Paper Ticket Review Queue')).toBeInTheDocument();
    expect(await screen.findAllByText('Paper review only')).toHaveLength(3);
    expect(screen.getAllByText('Trade idea')).toHaveLength(3);
    expect(screen.getAllByText('Why this exists')).toHaveLength(3);
    expect(screen.getAllByText('Evidence')).toHaveLength(3);
    expect(screen.getAllByText('Risk summary')).toHaveLength(3);
    expect(screen.getAllByText('Review actions')).toHaveLength(3);
    expect(screen.getAllByText('Notes')).toHaveLength(3);
    expect(screen.getByText('SPY')).toBeInTheDocument();
    expect(screen.getAllByText('long').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('pullback continuation')).toBeInTheDocument();
    expect(screen.getByText('Broad-market ETF holding above support.')).toBeInTheDocument();
    expect(screen.getByText('$101.25 / $98.75 / $107.00')).toBeInTheDocument();
    expect(screen.getByText('$10.00')).toBeInTheDocument();
    expect(screen.getByText('$12.00')).toBeInTheDocument();
    expect(screen.getByText('2.30')).toBeInTheDocument();
    expect(screen.getByText('spread check needed')).toBeInTheDocument();
    expect(screen.getByText('risk review pending')).toBeInTheDocument();
    expect(screen.getByText('Review again after the open.')).toBeInTheDocument();
    expect(screen.getAllByText('paper ticket reviewed').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('paper ticket cancelled').length).toBeGreaterThanOrEqual(1);

    expect(screen.getAllByRole('button', { name: /Mark reviewed/i })).toHaveLength(3);
    expect(screen.getAllByRole('button', { name: /Cancel paper ticket/i })).toHaveLength(3);
    expect(screen.getAllByRole('button', { name: /Add internal note/i })).toHaveLength(3);
    for (const forbidden of [
      /execute/i,
      /place order/i,
      /broker/i,
      /live/i,
      /leverage/i,
      /trade now/i,
      /auto trade/i,
    ]) {
      expect(screen.queryByRole('button', { name: forbidden })).not.toBeInTheDocument();
    }
    for (const forbiddenField of [
      /brokerExecutionAllowed/i,
      /executionInstructionCreated/i,
      /liveTradingAllowed/i,
      /leverageAllowed/i,
      /paperOnly/i,
    ]) {
      expect(screen.queryByText(forbiddenField)).not.toBeInTheDocument();
    }

    fireEvent.change(
      screen.getAllByPlaceholderText('Add an internal note for this paper review')[0],
      {
        target: { value: 'Beginner-safe note.' },
      },
    );
    fireEvent.click(screen.getAllByRole('button', { name: /Add internal note/i })[0]);
    await waitFor(() => {
      expect(approvalsService.addPaperTicketNote).toHaveBeenCalledWith(
        'pt_candidate_1',
        'Beginner-safe note.',
      );
    });

    fireEvent.click(screen.getAllByRole('button', { name: /Mark reviewed/i })[0]);
    await waitFor(() => {
      expect(approvalsService.markPaperTicketReviewed).toHaveBeenCalledWith(
        'pt_candidate_1',
        'marked reviewed from approvals page',
      );
    });
  });

  it('shows a friendly empty state when no paper tickets need review', async () => {
    vi.mocked(approvalsService.getQueue).mockResolvedValue([]);
    vi.mocked(approvalsService.getPaperTicketQueue).mockResolvedValue([]);
    vi.mocked(candidatesService.list).mockResolvedValue([] as never);

    renderPage();

    expect(await screen.findByText('No paper tickets need review.')).toBeInTheDocument();
    expect(
      screen.getByText(
        'New paper review cards will appear here after a candidate passes evidence and risk checks.',
      ),
    ).toBeInTheDocument();
  });
});
