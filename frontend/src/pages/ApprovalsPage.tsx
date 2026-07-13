import { type ReactNode, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle, Clock, MessageSquare, RefreshCw, XCircle } from 'lucide-react';
import {
  approvalsService,
  candidatesService,
  type ApprovalQueueItem,
  type CandidateApprovalDetail,
  type CandidateTrade,
  type PaperTicketReview,
} from '@/data/approvals-service';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { SentimentEvidencePanel } from '@/components/trading/SentimentEvidencePanel';
import { emitAnalyticsEvent } from '@/lib/analytics';

function fmtDate(raw?: string | null) {
  if (!raw) return '-';
  return new Date(raw).toLocaleString();
}

function ConfidenceBadge({ value }: { value?: number }) {
  if (value == null) return <span className="text-muted-foreground">-</span>;
  const pct = Math.round(value * 100);
  const variant = pct >= 70 ? 'default' : pct >= 50 ? 'secondary' : 'destructive';
  return <Badge variant={variant}>{pct}%</Badge>;
}

function SignalBadge({ type }: { type: string }) {
  return (
    <Badge variant={type === 'BUY' ? 'default' : 'destructive'} className="font-mono">
      {type}
    </Badge>
  );
}

function statusVariant(status: string) {
  switch (status) {
    case 'filled':
    case 'paper_ticket_reviewed':
      return 'success';
    case 'submitted':
      return 'default';
    case 'approved':
    case 'paper_ticket_created':
      return 'secondary';
    case 'blocked':
      return 'warning';
    case 'paper_ticket_cancelled':
      return 'destructive';
    default:
      return 'outline';
  }
}

function statusLabel(status: string) {
  return status.replace(/_/g, ' ');
}

function compactId(value?: string) {
  if (!value) return '-';
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function etfPolicyFromMetadata(metadata?: Record<string, unknown>) {
  const raw = metadata?.etfPolicy;
  if (!raw || typeof raw !== 'object') return null;
  const policy = raw as Record<string, unknown>;
  return {
    allowed: policy.allowed === true,
    reasonCode: typeof policy.reasonCode === 'string' ? policy.reasonCode : undefined,
    reason: typeof policy.reason === 'string' ? policy.reason : undefined,
    catalogVersion: typeof policy.catalogVersion === 'string' ? policy.catalogVersion : undefined,
  };
}

const SNOOZE_OPTIONS = [
  { label: '1h', hours: 1 },
  { label: '4h', hours: 4 },
  { label: '24h', hours: 24 },
];

interface CandidateRowProps {
  item: ApprovalQueueItem;
  onDecision: (
    id: string,
    action: 'approve' | 'reject' | 'snooze' | 'reanalyze',
    opts?: { snoozeHours?: number; notes?: string; overrideReason?: string },
  ) => void;
  pending: boolean;
  onEvidenceOpen: (item: ApprovalQueueItem) => void;
}

function CandidateRow({ item, onDecision, pending, onEvidenceOpen }: CandidateRowProps) {
  const [expanded, setExpanded] = useState(false);
  const [showNotes, setShowNotes] = useState(false);
  const [notes, setNotes] = useState('');
  const [overrideReason, setOverrideReason] = useState('risk_concern');
  const [snoozeHours, setSnoozeHours] = useState(4);
  const [confirmApprove, setConfirmApprove] = useState(false);
  const etfPolicy = etfPolicyFromMetadata(item.metadata);

  const submit = (action: 'approve' | 'reject' | 'snooze' | 'reanalyze') => {
    if (action === 'reject' || action === 'snooze') {
      emitAnalyticsEvent('approval_override_reason_selected', {
        source_surface: 'approvals',
        candidate_id: item.id,
        reason_code: overrideReason,
        sentiment_evidence_viewed: Boolean(item.sentiment ?? item.metadata?.sentiment),
      });
    }
    onDecision(item.id, action, { snoozeHours, notes: notes.trim() || undefined, overrideReason });
    setNotes('');
    setShowNotes(false);
    setConfirmApprove(false);
  };

  return (
    <Card className="mb-3">
      <CardHeader className="px-4 pb-2 pt-3">
        <div className="flex flex-wrap items-center gap-3">
          <SignalBadge type={item.signalType} />
          <span className="text-lg font-semibold">{item.symbol}</span>
          <ConfidenceBadge value={item.confidence} />
          <span className="ml-auto text-xs text-muted-foreground">
            Detected {fmtDate(item.detectedAt)}
          </span>
          {item.expiresAt && (
            <span className="flex items-center gap-1 text-xs text-yellow-500">
              <Clock className="h-3 w-3" /> Expires {fmtDate(item.expiresAt)}
            </span>
          )}
        </div>
        <p className="mt-1 text-xs text-muted-foreground">Strategy: {item.instanceName}</p>
      </CardHeader>
      <CardContent className="space-y-3 px-4 pb-3">
        <div className="flex flex-wrap gap-4 text-sm">
          {item.entryPrice != null && (
            <span>
              Entry: <strong>${item.entryPrice.toFixed(2)}</strong>
            </span>
          )}
          {item.stopLoss != null && (
            <span className="text-red-500">SL: ${item.stopLoss.toFixed(2)}</span>
          )}
          {item.takeProfit != null && (
            <span className="text-green-500">TP: ${item.takeProfit.toFixed(2)}</span>
          )}
        </div>

        {item.reasoning && (
          <div>
            <button
              className="text-xs text-muted-foreground underline"
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? 'Hide reasoning' : 'Show reasoning'}
            </button>
            {expanded && (
              <p className="mt-1 whitespace-pre-wrap rounded bg-muted p-2 text-sm text-foreground/80">
                {item.reasoning}
              </p>
            )}
          </div>
        )}

        {item.blockReason && (
          <div className="flex items-center gap-2 text-sm text-yellow-600">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>{item.blockReason}</span>
          </div>
        )}

        {etfPolicy ? (
          <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-sm">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={etfPolicy.allowed ? 'success' : 'destructive'}>
                ETF {etfPolicy.allowed ? 'eligible' : 'blocked'}
              </Badge>
              {etfPolicy.reasonCode && (
                <span className="font-mono text-xs text-muted-foreground">
                  {etfPolicy.reasonCode}
                </span>
              )}
              {etfPolicy.catalogVersion && (
                <span className="text-xs text-muted-foreground">
                  Policy {etfPolicy.catalogVersion}
                </span>
              )}
            </div>
            {etfPolicy.reason && (
              <p className="mt-1 text-xs text-muted-foreground">{etfPolicy.reason}</p>
            )}
          </div>
        ) : null}

        <SentimentEvidencePanel
          sentiment={item.sentiment ?? (item.metadata?.sentiment as never)}
          compact={!expanded}
        />

        {showNotes && (
          <div className="flex flex-col gap-1">
            <label
              className="text-xs font-medium text-muted-foreground"
              htmlFor={`override-${item.id}`}
            >
              Override reason
            </label>
            <select
              id={`override-${item.id}`}
              className="rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
              value={overrideReason}
              onChange={(event) => setOverrideReason(event.target.value)}
            >
              <option value="weak_sentiment_evidence">Weak sentiment evidence</option>
              <option value="stale_sources">Stale sources</option>
              <option value="policy_concern">Policy concern</option>
              <option value="risk_concern">Risk concern</option>
              <option value="price_sentiment_divergence">Price/sentiment divergence</option>
              <option value="duplicate_idea">Duplicate idea</option>
              <option value="other">Other</option>
            </select>
            <textarea
              className="w-full resize-none rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
              rows={2}
              placeholder="Optional notes (saved with the decision)..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
          </div>
        )}

        <div className="flex flex-wrap items-center gap-2 pt-1">
          {!confirmApprove ? (
            <Button
              size="sm"
              disabled={pending}
              onClick={() => setConfirmApprove(true)}
              className="bg-green-600 text-white hover:bg-green-700"
            >
              <CheckCircle className="mr-1 h-4 w-4" /> Approve for paper order
            </Button>
          ) : (
            <div className="flex items-center gap-1">
              <span className="text-xs text-muted-foreground">Create paper instruction?</span>
              <Button
                size="sm"
                disabled={pending}
                onClick={() => submit('approve')}
                className="bg-green-600 text-white hover:bg-green-700"
              >
                Yes, create paper order
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setConfirmApprove(false)}>
                Cancel
              </Button>
            </div>
          )}

          <Button
            size="sm"
            variant="destructive"
            disabled={pending}
            onClick={() => submit('reject')}
          >
            <XCircle className="mr-1 h-4 w-4" /> Reject
          </Button>

          <div className="flex items-center gap-1">
            <Button size="sm" variant="outline" disabled={pending} onClick={() => submit('snooze')}>
              <Clock className="mr-1 h-4 w-4" /> Snooze
            </Button>
            <select
              className="h-8 rounded-md border border-input bg-background px-1 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
              value={snoozeHours}
              title="Snooze duration"
              aria-label="Snooze duration"
              onChange={(e) => setSnoozeHours(Number(e.target.value))}
              disabled={pending}
            >
              {SNOOZE_OPTIONS.map((o) => (
                <option key={o.hours} value={o.hours}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>

          <Button size="sm" variant="ghost" disabled={pending} onClick={() => submit('reanalyze')}>
            <RefreshCw className="mr-1 h-4 w-4" /> Re-analyse
          </Button>

          <Button size="sm" variant="outline" asChild>
            <Link to={`/candidates/${item.id}/evidence`} onClick={() => onEvidenceOpen(item)}>
              Evidence
            </Link>
          </Button>

          <button
            className="ml-auto text-xs text-muted-foreground underline"
            onClick={() => setShowNotes(!showNotes)}
          >
            {showNotes ? 'Hide notes' : 'Add notes'}
          </button>
        </div>
      </CardContent>
    </Card>
  );
}

function SummaryCard({
  label,
  value,
  hint,
}: {
  label: string;
  value: string | number;
  hint: string;
}) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {label}
        </p>
        <p className="mt-2 text-2xl font-semibold">{value}</p>
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}

function CandidateMeta({ label, value }: { label: string; value?: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
      <code className="rounded bg-muted px-2 py-0.5 text-xs">{compactId(value)}</code>
    </div>
  );
}

function ExecutionActivityRow({ item }: { item: CandidateTrade }) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex flex-wrap items-center gap-2">
        <SignalBadge type={item.signalType} />
        <span className="font-semibold">{item.symbol}</span>
        <Badge variant={statusVariant(item.status)}>{statusLabel(item.status)}</Badge>
        <span className="ml-auto text-xs text-muted-foreground">
          Updated {fmtDate(item.filledAt ?? item.submittedAt ?? item.detectedAt)}
        </span>
      </div>
      <div className="mt-3 grid gap-2 text-sm md:grid-cols-2">
        <div className="space-y-2">
          <CandidateMeta label="Candidate" value={item.id} />
          <CandidateMeta label="Approval" value={item.latestApproval?.id} />
          <CandidateMeta label="Instruction" value={item.executionInstructionId} />
          <CandidateMeta label="Trade" value={item.tradeId} />
        </div>
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Decision</span>
            <span>{item.latestApproval?.decision ?? '-'}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Decided</span>
            <span>{fmtDate(item.latestApproval?.decidedAt)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Provenance</span>
            <span className="truncate">{item.dataProvenance || '-'}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function formatMoney(value?: number) {
  return typeof value === 'number' ? `$${value.toFixed(2)}` : '-';
}

function formatQuantity(value?: number) {
  return typeof value === 'number'
    ? value.toLocaleString(undefined, { maximumFractionDigits: 4 })
    : '-';
}

function PaperReviewField({ label, value }: { label: string; value?: string }) {
  return (
    <div className="rounded-md bg-muted/30 px-3 py-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-semibold text-foreground">{value || '-'}</div>
    </div>
  );
}

function PaperReviewSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-2">
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      {children}
    </section>
  );
}

function PaperReviewReasonList({ label, values }: { label: string; values?: string[] }) {
  if (!values?.length) {
    return <PaperReviewField label={label} value="None" />;
  }
  return (
    <div className="rounded-md bg-muted/30 px-3 py-2">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <ul className="mt-1 list-disc space-y-1 pl-4 text-sm text-foreground">
        {values.map((value) => (
          <li key={value}>{statusLabel(value)}</li>
        ))}
      </ul>
    </div>
  );
}

function PaperTicketReviewRow({
  item,
  pending,
  onAction,
}: {
  item: PaperTicketReview;
  pending: boolean;
  onAction: (
    paperTicketId: string,
    action: 'mark_reviewed' | 'cancel' | 'add_note',
    note?: string,
  ) => void;
}) {
  const [note, setNote] = useState('');
  const isReviewed = item.status === 'paper_ticket_reviewed';
  const isCancelled = item.status === 'paper_ticket_cancelled';
  const trimmedNote = note.trim();

  const submitNote = () => {
    if (!trimmedNote || isCancelled) return;
    onAction(item.paperTicketId, 'add_note', trimmedNote);
    setNote('');
  };

  return (
    <article
      className={`rounded-md border p-4 ${
        isCancelled
          ? 'border-destructive/40 bg-destructive/5'
          : isReviewed
            ? 'border-green-500/40 bg-green-500/5'
            : 'border-border'
      }`}
    >
      <div className="flex flex-wrap items-start gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline">Paper review only</Badge>
            <Badge variant={statusVariant(item.status)}>{statusLabel(item.status)}</Badge>
            <span className="text-lg font-semibold">{item.symbol}</span>
            <span className="text-sm text-muted-foreground">{statusLabel(item.direction)}</span>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">Updated {fmtDate(item.updatedAt)}</p>
        </div>
        <div className="ml-auto text-right text-xs text-muted-foreground">
          <div>Paper ticket {compactId(item.paperTicketId)}</div>
          <div>Candidate {compactId(item.candidateId)}</div>
        </div>
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[1.1fr_1fr]">
        <div className="space-y-4">
          <PaperReviewSection title="Trade idea">
            <div className="grid gap-2 sm:grid-cols-2">
              <PaperReviewField label="Setup type" value={statusLabel(item.setupType)} />
              <PaperReviewField label="Review status" value={statusLabel(item.status)} />
            </div>
            <div className="rounded-md bg-muted/30 px-3 py-2">
              <div className="text-xs font-medium text-muted-foreground">Why this exists</div>
              <p className="mt-1 text-sm text-foreground">{item.catalystSummary || '-'}</p>
            </div>
          </PaperReviewSection>

          <PaperReviewSection title="Evidence">
            <div className="grid gap-2 sm:grid-cols-3">
              <PaperReviewField label="Evidence status" value={statusLabel(item.evidenceStatus)} />
              <PaperReviewField label="Gate status" value={statusLabel(item.gateStatus)} />
              <PaperReviewField label="Approval status" value={statusLabel(item.approvalStatus)} />
            </div>
            <PaperReviewReasonList label="Warnings" values={item.warningReasons} />
          </PaperReviewSection>
        </div>

        <div className="space-y-4">
          <PaperReviewSection title="Risk summary">
            <div className="grid gap-2 sm:grid-cols-2">
              <PaperReviewField
                label="Entry / Stop / Target"
                value={`${formatMoney(item.entryPrice)} / ${formatMoney(item.stopLossPrice)} / ${formatMoney(item.targetPrice)}`}
              />
              <PaperReviewField label="Position size" value={formatQuantity(item.positionSize)} />
              <PaperReviewField label="Max normal loss" value={formatMoney(item.maxNormalLoss)} />
              <PaperReviewField
                label="Worst planned loss with slippage"
                value={formatMoney(item.maxSlippageAdjustedLoss)}
              />
              <PaperReviewField label="Reward:risk" value={item.rewardRiskRatio.toFixed(2)} />
              <PaperReviewField label="Risk status" value={statusLabel(item.riskStatus)} />
            </div>
            <PaperReviewReasonList label="Risk blockers" values={item.rejectReasons} />
          </PaperReviewSection>

          <PaperReviewSection title="Notes">
            {item.reviewNotes ? (
              <div className="whitespace-pre-wrap rounded-md bg-muted/30 px-3 py-2 text-sm text-foreground">
                {item.reviewNotes}
              </div>
            ) : (
              <p className="rounded-md bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                No internal notes yet.
              </p>
            )}
          </PaperReviewSection>
        </div>
      </div>

      <PaperReviewSection title="Review actions">
        <div className="flex flex-col gap-2 pt-1">
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={pending || isReviewed || isCancelled}
              onClick={() => onAction(item.paperTicketId, 'mark_reviewed')}
            >
              <CheckCircle className="mr-1 h-4 w-4" /> Mark reviewed
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={pending || isCancelled}
              onClick={() => onAction(item.paperTicketId, 'cancel')}
            >
              <XCircle className="mr-1 h-4 w-4" /> Cancel paper ticket
            </Button>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <label className="sr-only" htmlFor={`paper-note-${item.paperTicketId}`}>
              Internal notes
            </label>
            <textarea
              id={`paper-note-${item.paperTicketId}`}
              className="min-h-20 flex-1 resize-none rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
              placeholder="Add an internal note for this paper review"
              value={note}
              onChange={(event) => setNote(event.target.value)}
              disabled={pending || isCancelled}
            />
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={pending || isCancelled || !trimmedNote}
              onClick={submitNote}
            >
              <MessageSquare className="mr-1 h-4 w-4" /> Add internal note
            </Button>
          </div>
        </div>
      </PaperReviewSection>
    </article>
  );
}

function BlockedCandidateRow({
  item,
  onRefresh,
  pending,
  onEvidenceOpen,
}: {
  item: CandidateTrade;
  onRefresh: (id: string) => void;
  pending: boolean;
  onEvidenceOpen: (item: CandidateTrade) => void;
}) {
  const etfPolicy = etfPolicyFromMetadata(item.metadata);
  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex flex-wrap items-center gap-2">
        <SignalBadge type={item.signalType} />
        <span className="font-semibold">{item.symbol}</span>
        <Badge variant="warning">{item.blockedReasonCode ?? 'blocked'}</Badge>
        <span className="ml-auto text-xs text-muted-foreground">
          Blocked {fmtDate(item.blockedAt ?? item.detectedAt)}
        </span>
      </div>
      <p className="mt-2 text-sm text-foreground/80">
        {item.blockReason ?? 'No block reason recorded.'}
      </p>
      {etfPolicy ? (
        <p className="mt-1 text-xs text-muted-foreground">
          ETF policy: {etfPolicy.reasonCode ?? 'unknown'}
          {etfPolicy.reason ? ` - ${etfPolicy.reason}` : ''}
        </p>
      ) : null}
      <div className="mt-3 grid gap-2 text-sm md:grid-cols-2">
        <div className="space-y-2">
          <CandidateMeta label="Candidate" value={item.id} />
          <CandidateMeta label="Signal" value={item.signalId} />
          <CandidateMeta label="Artifact" value={item.artifactId} />
        </div>
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Strategy</span>
            <span>{item.strategyId ?? '-'}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Detected</span>
            <span>{fmtDate(item.detectedAt)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Provenance</span>
            <span className="truncate">{item.dataProvenance || '-'}</span>
          </div>
          <div className="flex flex-wrap items-center gap-2 pt-2">
            <Button size="sm" variant="outline" asChild>
              <Link to={`/candidates/${item.id}/evidence`} onClick={() => onEvidenceOpen(item)}>
                Evidence
              </Link>
            </Button>
            <Button size="sm" disabled={pending} onClick={() => onRefresh(item.id)}>
              <RefreshCw className="mr-1 h-4 w-4" /> Re-qualify & Queue Mobile
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export function ApprovalsPage() {
  const qc = useQueryClient();
  const [notification, setNotification] = useState<string | null>(null);
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [pendingRefreshId, setPendingRefreshId] = useState<string | null>(null);
  const [pendingPaperTicketId, setPendingPaperTicketId] = useState<string | null>(null);

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'approvals' });
  }, []);

  const {
    data: queue = [],
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['approvals-queue'],
    queryFn: () => approvalsService.getQueue(),
    refetchInterval: 30_000,
  });
  const { data: blockedCandidates = [] } = useQuery({
    queryKey: ['approvals-blocked-candidates'],
    queryFn: () => candidatesService.list({ status: 'blocked', limit: 8 }),
    refetchInterval: 30_000,
  });
  const { data: executionActivity = [] } = useQuery({
    queryKey: ['approvals-execution-activity'],
    queryFn: async () => {
      const [approved, submitted, filled] = await Promise.all([
        candidatesService.list({ status: 'approved', limit: 6 }),
        candidatesService.list({ status: 'submitted', limit: 6 }),
        candidatesService.list({ status: 'filled', limit: 6 }),
      ]);
      return [...approved, ...submitted, ...filled]
        .sort((a, b) => {
          const left = new Date(a.filledAt ?? a.submittedAt ?? a.detectedAt).getTime();
          const right = new Date(b.filledAt ?? b.submittedAt ?? b.detectedAt).getTime();
          return right - left;
        })
        .slice(0, 8);
    },
    refetchInterval: 30_000,
  });
  const {
    data: paperTicketQueue = [],
    isLoading: isPaperTicketLoading,
    isError: isPaperTicketError,
    refetch: refetchPaperTickets,
  } = useQuery({
    queryKey: ['paper-ticket-review-queue'],
    queryFn: () => approvalsService.getPaperTicketQueue(),
    refetchInterval: 30_000,
  });

  const blockedReasonEntries = useMemo(() => {
    const counts = blockedCandidates.reduce<Record<string, number>>((acc, item) => {
      const key = item.blockedReasonCode ?? 'unspecified';
      acc[key] = (acc[key] ?? 0) + 1;
      return acc;
    }, {});
    return Object.entries(counts).sort((a, b) => b[1] - a[1]);
  }, [blockedCandidates]);

  const mutation = useMutation({
    mutationFn: async ({
      id,
      action,
      opts,
    }: {
      id: string;
      action: 'approve' | 'reject' | 'snooze' | 'reanalyze';
      opts?: { snoozeHours?: number; notes?: string; overrideReason?: string };
    }) => {
      switch (action) {
        case 'approve':
          return approvalsService.approve(id, opts?.notes);
        case 'reject':
          return approvalsService.reject(
            id,
            opts?.notes,
            opts?.overrideReason
              ? { reasonCode: opts.overrideReason, sentimentEvidenceViewed: true }
              : undefined,
          );
        case 'snooze':
          return approvalsService.snooze(
            id,
            opts?.snoozeHours ?? 4,
            opts?.notes,
            opts?.overrideReason
              ? { reasonCode: opts.overrideReason, sentimentEvidenceViewed: true }
              : undefined,
          );
        case 'reanalyze':
          return approvalsService.reanalyze(id, opts?.notes);
      }
    },
    onMutate: ({ id }) => setPendingId(id),
    onSuccess: (data: CandidateApprovalDetail, { action }) => {
      qc.invalidateQueries({ queryKey: ['approvals-queue'] });
      qc.invalidateQueries({ queryKey: ['approvals-blocked-candidates'] });
      qc.invalidateQueries({ queryKey: ['approvals-execution-activity'] });
      setPendingId(null);
      const executionStatus = data.execution?.status;
      setNotification(
        executionStatus
          ? `Decision recorded: ${action}. Execution: ${executionStatus}`
          : `Decision recorded: ${action}`,
      );
      setTimeout(() => setNotification(null), 3000);
    },
    onError: (err: Error) => {
      setPendingId(null);
      setNotification(`Error: ${err.message}`);
      setTimeout(() => setNotification(null), 5000);
    },
  });

  const refreshMutation = useMutation({
    mutationFn: async (id: string) => candidatesService.refresh(id),
    onMutate: (id) => setPendingRefreshId(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['approvals-queue'] });
      qc.invalidateQueries({ queryKey: ['approvals-blocked-candidates'] });
      qc.invalidateQueries({ queryKey: ['approvals-execution-activity'] });
      setPendingRefreshId(null);
      setNotification('Candidate re-qualified and mobile approval notification queued.');
      setTimeout(() => setNotification(null), 3000);
    },
    onError: (err: Error) => {
      setPendingRefreshId(null);
      setNotification(`Error: ${err.message}`);
      setTimeout(() => setNotification(null), 5000);
    },
  });

  const paperTicketMutation = useMutation({
    mutationFn: async ({
      id,
      action,
      note,
    }: {
      id: string;
      action: 'mark_reviewed' | 'cancel' | 'add_note';
      note?: string;
    }) => {
      if (action === 'mark_reviewed') {
        return approvalsService.markPaperTicketReviewed(
          id,
          note ?? 'marked reviewed from approvals page',
        );
      }
      if (action === 'add_note') {
        return approvalsService.addPaperTicketNote(id, note ?? '');
      }
      return approvalsService.cancelPaperTicket(id, note ?? 'cancelled from approvals page');
    },
    onMutate: ({ id }) => setPendingPaperTicketId(id),
    onSuccess: (_data, { action }) => {
      qc.invalidateQueries({ queryKey: ['paper-ticket-review-queue'] });
      setPendingPaperTicketId(null);
      setNotification(
        action === 'mark_reviewed'
          ? 'Paper ticket marked reviewed.'
          : action === 'add_note'
            ? 'Internal note added.'
            : 'Paper ticket cancelled.',
      );
      setTimeout(() => setNotification(null), 3000);
    },
    onError: (err: Error) => {
      setPendingPaperTicketId(null);
      setNotification(`Error: ${err.message}`);
      setTimeout(() => setNotification(null), 5000);
    },
  });

  const handleDecision = (
    id: string,
    action: 'approve' | 'reject' | 'snooze' | 'reanalyze',
    opts?: { snoozeHours?: number; notes?: string },
  ) => {
    mutation.mutate({ id, action, opts });
  };

  const handleRefreshCandidate = (id: string) => {
    refreshMutation.mutate(id);
  };

  const handlePaperTicketAction = (
    id: string,
    action: 'mark_reviewed' | 'cancel' | 'add_note',
    note?: string,
  ) => {
    paperTicketMutation.mutate({ id, action, note });
  };

  const handleEvidenceOpen = (item: ApprovalQueueItem | CandidateTrade) => {
    const routeType =
      'status' in item && typeof item.status === 'string' ? item.status : 'approval_required';

    emitAnalyticsEvent('approval_sentiment_evidence_viewed', {
      source_surface: 'approvals',
      candidate_id: item.id,
      route_type: routeType,
      sentiment_mode:
        typeof item.metadata?.sentimentMode === 'string'
          ? item.metadata.sentimentMode
          : typeof item.metadata?.sentiment_mode === 'string'
            ? item.metadata.sentiment_mode
            : undefined,
    });
  };

  return (
    <div className="container mx-auto max-w-5xl p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Approval Queue</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Candidate trades awaiting your decision. AI is advisory only - you remain in control.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            refetch();
            refetchPaperTickets();
          }}
        >
          <RefreshCw className="mr-1 h-4 w-4" /> Refresh
        </Button>
      </div>

      {notification && (
        <div className="mb-4 rounded-md bg-accent px-4 py-2 text-sm text-accent-foreground">
          {notification}
        </div>
      )}

      <div className="mb-6 grid gap-3 md:grid-cols-4">
        <SummaryCard
          label="Pending approvals"
          value={queue.length}
          hint="Candidates waiting for a human decision."
        />
        <SummaryCard
          label="Paper review"
          value={paperTicketQueue.length}
          hint="Persisted paper tickets awaiting review."
        />
        <SummaryCard
          label="Recent execution chain"
          value={executionActivity.length}
          hint="Approved, submitted, or filled candidates."
        />
        <SummaryCard
          label="Recent blocked"
          value={blockedCandidates.length}
          hint="Latest blocked candidates with reason codes."
        />
      </div>

      {isLoading && <p className="text-muted-foreground">Loading approval queue...</p>}
      {isError && (
        <p className="text-destructive">
          Failed to load approval queue. Check backend connectivity.
        </p>
      )}

      {!isLoading && !isError && queue.length === 0 && (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            No candidates awaiting approval.
          </CardContent>
        </Card>
      )}

      {queue.map((item) => (
        <CandidateRow
          key={item.id}
          item={item}
          onDecision={handleDecision}
          pending={pendingId === item.id}
          onEvidenceOpen={handleEvidenceOpen}
        />
      ))}

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Paper Ticket Review Queue</CardTitle>
          <p className="text-sm text-muted-foreground">
            Paper review only: read the idea, evidence, and risk before recording a review note or
            closing the ticket.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {isPaperTicketLoading ? (
            <div className="rounded-md border border-dashed border-border px-4 py-6 text-sm text-muted-foreground">
              Loading paper tickets for review...
            </div>
          ) : isPaperTicketError ? (
            <div className="rounded-md border border-destructive/40 bg-destructive/5 px-4 py-6 text-sm text-destructive">
              Paper ticket review is unavailable right now. Check the backend connection and refresh
              this page.
            </div>
          ) : paperTicketQueue.length === 0 ? (
            <div className="rounded-md border border-dashed border-border px-4 py-8 text-center">
              <p className="font-medium">No paper tickets need review.</p>
              <p className="mt-1 text-sm text-muted-foreground">
                New paper review cards will appear here after a candidate passes evidence and risk
                checks.
              </p>
            </div>
          ) : (
            paperTicketQueue.map((item) => (
              <PaperTicketReviewRow
                key={item.paperTicketId}
                item={item}
                pending={pendingPaperTicketId === item.paperTicketId}
                onAction={handlePaperTicketAction}
              />
            ))
          )}
        </CardContent>
      </Card>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Recent Execution Activity</CardTitle>
          <p className="text-sm text-muted-foreground">
            Shows the approval {'->'} instruction {'->'} trade chain for the most recent paper-mode
            candidates.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {executionActivity.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No recent approved, submitted, or filled candidates.
            </p>
          ) : (
            executionActivity.map((item) => (
              <ExecutionActivityRow key={`${item.status}-${item.id}`} item={item} />
            ))
          )}
        </CardContent>
      </Card>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle>Recently Blocked</CardTitle>
          <p className="text-sm text-muted-foreground">
            Recent blocked candidates and the blocker codes preventing promotion.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {blockedReasonEntries.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {blockedReasonEntries.map(([reason, count]) => (
                <Badge key={reason} variant="outline">
                  {reason}: {count}
                </Badge>
              ))}
            </div>
          )}
          {blockedCandidates.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No blocked candidates found in the current window.
            </p>
          ) : (
            blockedCandidates.map((item) => (
              <BlockedCandidateRow
                key={item.id}
                item={item}
                onRefresh={handleRefreshCandidate}
                pending={pendingRefreshId === item.id}
                onEvidenceOpen={handleEvidenceOpen}
              />
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
