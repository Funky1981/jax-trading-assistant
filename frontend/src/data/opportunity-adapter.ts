import type { ApprovalQueueItem, CandidateTrade } from '@/data/approvals-service';
import type { OpportunityConfidenceBand, OpportunityRoute, OpportunitySummary, SentimentEvidence, Signal } from '@/data/types';

interface OpportunitySources {
  signals?: Signal[];
  candidates?: CandidateTrade[];
  approvals?: ApprovalQueueItem[];
}

const BLOCKED_STATUSES = new Set(['blocked', 'rejected', 'expired']);
const APPROVAL_STATUSES = new Set(['awaiting_approval', 'pending_approval', 'pending', 'snoozed']);
const EXECUTION_STATUSES = new Set(['approved', 'submitted', 'filled', 'cancelled']);

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

function routeForCandidate(candidate: CandidateTrade): { route: OpportunityRoute; routeReason: string; routeReasonCode?: string } {
  const status = typeof candidate.status === 'string' ? candidate.status.toLowerCase() : '';

  if (BLOCKED_STATUSES.has(status)) {
    return {
      route: 'blocked',
      routeReason: candidate.blockReason ?? candidate.blockedReasonCode ?? 'Policy or data quality checks blocked this opportunity.',
      routeReasonCode: candidate.blockedReasonCode,
    };
  }

  if (APPROVAL_STATUSES.has(status)) {
    return {
      route: 'approval_required',
      routeReason: 'This proposed trade requires approval before execution.',
    };
  }

  if (EXECUTION_STATUSES.has(status)) {
    return {
      route: 'execution_ready',
      routeReason:
        candidate.executionInstructionId || candidate.latestApproval
          ? 'This candidate has left the approval queue. Review its approval -> instruction -> broker status in the execution chain.'
          : 'This candidate has left the approval queue. Review its current execution state before taking any action.',
    };
  }

  return {
    route: 'manual_allowed',
    routeReason: 'This opportunity can be reviewed in the manual trading workflow.',
  };
}

function stringArray(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) return undefined;
  const out = value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0);
  return out.length > 0 ? out : undefined;
}

function numberRecord(value: unknown): Record<string, number> | undefined {
  if (!value || typeof value !== 'object') return undefined;
  const entries = Object.entries(value as Record<string, unknown>)
    .filter((entry): entry is [string, number] => typeof entry[1] === 'number' && Number.isFinite(entry[1]));
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}

function sentimentFromMetadata(metadata?: Record<string, unknown>): SentimentEvidence | undefined {
  const raw = metadata?.sentiment ?? metadata?.sentiment_summary_structured ?? metadata?.sentimentSummaryStructured;
  const legacySummary = metadata?.sentimentSummary ?? metadata?.sentiment_summary;
  if (raw && typeof raw === 'object') {
    const value = raw as Record<string, unknown>;
    const label = typeof value.label === 'string' ? value.label : 'mixed';
    const state = typeof value.state === 'string' ? value.state : 'available';
    return {
      score: typeof value.score === 'number' ? value.score : undefined,
      label: label === 'positive' || label === 'negative' || label === 'mixed' || label === 'unavailable' ? label : 'mixed',
      confidence: typeof value.confidence === 'number' ? value.confidence : undefined,
      window: typeof value.window === 'string' ? value.window : typeof value.timeWindow === 'string' ? value.timeWindow : undefined,
      sourceCount:
        typeof value.sourceCount === 'number'
          ? value.sourceCount
          : typeof value.source_count === 'number'
            ? value.source_count
            : undefined,
      sourceGroups: numberRecord(value.sourceGroups ?? value.source_groups),
      priceAgreement:
        value.priceAgreement === 'agreeing' ||
        value.priceAgreement === 'diverging' ||
        value.priceAgreement === 'neutral' ||
        value.priceAgreement === 'unknown'
          ? value.priceAgreement
          : value.price_agreement === 'agreeing' ||
              value.price_agreement === 'diverging' ||
              value.price_agreement === 'neutral' ||
              value.price_agreement === 'unknown'
            ? value.price_agreement
            : undefined,
      topDrivers: stringArray(value.topDrivers ?? value.top_drivers),
      limitations: stringArray(value.limitations),
      state:
        state === 'available' ||
        state === 'disabled' ||
        state === 'missing' ||
        state === 'sparse' ||
        state === 'low_confidence' ||
        state === 'degraded' ||
        state === 'error'
          ? state
          : 'available',
      summary: typeof value.summary === 'string' ? value.summary : typeof legacySummary === 'string' ? legacySummary : undefined,
      snapshotAt:
        typeof value.snapshotAt === 'string'
          ? value.snapshotAt
          : typeof value.snapshot_at === 'string'
            ? value.snapshot_at
            : undefined,
      intendedUse:
        typeof value.intendedUse === 'string'
          ? value.intendedUse
          : 'Use sentiment as supporting evidence beside strategy, price, policy, and risk.',
    };
  }
  if (typeof legacySummary === 'string' && legacySummary.trim().length > 0) {
    return {
      label: 'mixed',
      state: 'available',
      summary: legacySummary,
      intendedUse: 'Use sentiment as supporting evidence beside strategy, price, policy, and risk.',
      limitations: ['Legacy sentiment summary does not include source counts or confidence.'],
    };
  }
  return undefined;
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
    routeReasonCode: route.routeReasonCode,
    sentimentSummary: sentimentFromMetadata(candidate.metadata)?.summary,
    sentiment: candidate.sentiment ?? sentimentFromMetadata(candidate.metadata),
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
    sentimentSummary: sentimentFromMetadata(item.metadata)?.summary,
    sentiment: item.sentiment ?? sentimentFromMetadata(item.metadata),
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
