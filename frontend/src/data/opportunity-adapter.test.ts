import { describe, expect, it } from 'vitest';
import { opportunityFromApproval, opportunityFromCandidate, opportunityFromSignal, toOpportunitySummaries } from './opportunity-adapter';
import type { ApprovalQueueItem, CandidateTrade } from './approvals-service';
import type { Signal } from './types';

describe('opportunity adapter', () => {
  it('normalizes signals into manual review opportunities', () => {
    const signal: Signal = {
      id: 'signal-1',
      symbol: 'AAPL',
      strategy_id: 'breakout',
      signal_type: 'BUY',
      confidence: 0.87,
      reasoning: 'Momentum and volume are aligned.',
      generated_at: '2026-05-22T09:30:00Z',
      expires_at: '2026-05-22T10:30:00Z',
      status: 'pending',
      created_at: '2026-05-22T09:29:00Z',
    };

    expect(opportunityFromSignal(signal)).toMatchObject({
      id: 'signal:signal-1',
      symbol: 'AAPL',
      confidenceBand: 'high',
      route: 'manual_allowed',
      routeReason: 'Signal is ready for review before creating an order.',
      status: 'pending',
    });
  });

  it('preserves blocked candidate route and policy reason', () => {
    const candidate: CandidateTrade = {
      id: 'candidate-1',
      strategyInstanceId: 'instance-1',
      symbol: 'SPY',
      signalType: 'BUY',
      status: 'blocked',
      confidence: 0.72,
      reasoning: 'Setup is positive but policy blocked entry.',
      blockReason: 'ETF entries require approval queue routing.',
      sessionDate: '2026-05-22',
      detectedAt: '2026-05-22T09:35:00Z',
      dataProvenance: 'paper',
    };

    expect(opportunityFromCandidate(candidate)).toMatchObject({
      id: 'candidate:candidate-1',
      symbol: 'SPY',
      confidenceBand: 'medium',
      route: 'blocked',
      routeReason: 'ETF entries require approval queue routing.',
      sourceType: 'candidate',
    });
  });

  it('normalizes approval queue items as approval-required opportunities', () => {
    const approval: ApprovalQueueItem = {
      id: 'candidate-2',
      symbol: 'QQQ',
      signalType: 'SELL',
      confidence: 0.59,
      reasoning: 'Risk-off rotation detected.',
      detectedAt: '2026-05-22T09:40:00Z',
      expiresAt: '2026-05-22T10:00:00Z',
      instanceName: 'ETF guardrail',
    };

    expect(opportunityFromApproval(approval)).toMatchObject({
      id: 'approval:candidate-2',
      confidenceBand: 'low',
      route: 'approval_required',
      status: 'awaiting_approval',
    });
  });

  it('returns a deterministic newest-first unified feed', () => {
    const signal = {
      id: 'signal-old',
      symbol: 'MSFT',
      strategy_id: 'breakout',
      signal_type: 'BUY',
      confidence: 0.9,
      generated_at: '2026-05-22T09:00:00Z',
      status: 'pending',
      created_at: '2026-05-22T09:00:00Z',
    } satisfies Signal;
    const approval = {
      id: 'candidate-new',
      symbol: 'IWM',
      signalType: 'BUY',
      detectedAt: '2026-05-22T09:45:00Z',
      instanceName: 'ETF guardrail',
    } satisfies ApprovalQueueItem;

    expect(toOpportunitySummaries({ signals: [signal], approvals: [approval] }).map((item) => item.id)).toEqual([
      'approval:candidate-new',
      'signal:signal-old',
    ]);
  });
});
