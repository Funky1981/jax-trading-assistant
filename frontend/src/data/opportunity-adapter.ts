import type { ApprovalQueueItem, CandidateTrade } from '@/data/approvals-service';
import type { OpportunityConfidenceBand, OpportunityRoute, OpportunitySummary, Signal } from '@/data/types';

interface OpportunitySources {
  signals?: Signal[];
  candidates?: CandidateTrade[];
  approvals?: ApprovalQueueItem[];
}

const BLOCKED_STATUSES = new Set(['blocked', 'rejected', 'expired']);
const APPROVAL_STATUSES = new Set(['awaiting_approval', 'pending_approval', 'pending', 'snoozed']);

function confidenceBand(confidence?: number | null): OpportunityConfidenceBand {
  if (typeof confidence !== 'number') return 'unknown';
  if (confidence >= 0.8) return 'high';
  if (confidence >= 0.6) return 'medium';
  return 'low';
}

function sentence(value?: string | null, fallback = 'Opportunity detected by Jax.'): string {
  const trimmed = value?.trim();
  return trimmed && trimmed.length > 0 ? trimmed : fallback;
}

function routeForCandidate(candidate: CandidateTrade): { route: OpportunityRoute; routeReason: string } {
  const status = typeof candidate.status === 'string' ? candidate.status.toLowerCase() : '';

  if (BLOCKED_STATUSES.has(status)) {
    return {
      route: 'blocked',
      routeReason: candidate.blockReason ?? candidate.blockedReasonCode ?? 'Policy or data quality checks blocked this opportunity.',
    };
  }

  if (APPROVAL_STATUSES.has(status)) {
    return {
      route: 'approval_required',
      routeReason: 'This proposed trade requires approval before execution.',
    };
  }

  return {
    route: 'manual_allowed',
    routeReason: 'This opportunity can be reviewed in the manual trading workflow.',
  };
}

function sentimentFromMetadata(metadata?: Record<string, unknown>): string | undefined {
  const value = metadata?.sentimentSummary ?? metadata?.sentiment_summary;
  return typeof value === 'string' && value.trim().length > 0 ? value : undefined;
}

export function opportunityFromSignal(signal: Signal): OpportunitySummary {
  return {
    id: `signal:${signal.id}`,
    symbol: signal.symbol || 'UNKNOWN',
    signalType: signal.signal_type || 'UNKNOWN',
    confidenceBand: confidenceBand(signal.confidence),
    summary: sentence(signal.reasoning, `${signal.signal_type || 'Signal'} opportunity for ${signal.symbol || 'an unknown symbol'}.`),
    detectedAt: signal.generated_at || signal.created_at,
    expiresAt: signal.expires_at ?? undefined,
    route: 'manual_allowed',
    routeReason: 'Signal is ready for review before creating an order.',
    status: signal.status || 'unknown',
    sourceType: 'signal',
    sourceId: signal.id,
  };
}

export function opportunityFromCandidate(candidate: CandidateTrade): OpportunitySummary {
  const route = routeForCandidate(candidate);

  return {
    id: `candidate:${candidate.id}`,
    symbol: candidate.symbol || 'UNKNOWN',
    signalType: candidate.signalType || 'UNKNOWN',
    confidenceBand: confidenceBand(candidate.confidence),
    summary: sentence(candidate.reasoning, `${candidate.signalType || 'Proposed'} trade setup for ${candidate.symbol || 'an unknown symbol'}.`),
    detectedAt: candidate.detectedAt || candidate.qualifiedAt || candidate.sessionDate,
    expiresAt: candidate.expiresAt,
    route: route.route,
    routeReason: route.routeReason,
    sentimentSummary: sentimentFromMetadata(candidate.metadata),
    status: candidate.status || 'unknown',
    sourceType: 'candidate',
    sourceId: candidate.id,
  };
}

export function opportunityFromApproval(item: ApprovalQueueItem): OpportunitySummary {
  return {
    id: `approval:${item.id}`,
    symbol: item.symbol || 'UNKNOWN',
    signalType: item.signalType || 'UNKNOWN',
    confidenceBand: confidenceBand(item.confidence),
    summary: sentence(item.reasoning, `${item.signalType || 'Proposed'} trade awaiting approval for ${item.symbol || 'an unknown symbol'}.`),
    detectedAt: item.detectedAt,
    expiresAt: item.expiresAt,
    route: 'approval_required',
    routeReason: item.blockReason ?? 'Approval is required before this opportunity can move to execution.',
    sentimentSummary: sentimentFromMetadata(item.metadata),
    status: 'awaiting_approval',
    sourceType: 'approval',
    sourceId: item.id,
  };
}

export function toOpportunitySummaries(sources: OpportunitySources): OpportunitySummary[] {
  return [
    ...(sources.approvals ?? []).map(opportunityFromApproval),
    ...(sources.candidates ?? []).map(opportunityFromCandidate),
    ...(sources.signals ?? []).map(opportunityFromSignal),
  ].sort((left, right) => new Date(right.detectedAt).getTime() - new Date(left.detectedAt).getTime());
}
