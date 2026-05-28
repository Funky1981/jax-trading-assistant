import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle, Clock, RefreshCw, XCircle } from 'lucide-react';
import {
  approvalsService,
  candidatesService,
  type ApprovalQueueItem,
  type CandidateApprovalDetail,
  type CandidateTrade,
} from '@/data/approvals-service';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
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
      return 'success';
    case 'submitted':
      return 'default';
    case 'approved':
      return 'secondary';
    case 'blocked':
      return 'warning';
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
    opts?: { snoozeHours?: number; notes?: string }
  ) => void;
  pending: boolean;
  onEvidenceOpen: (item: ApprovalQueueItem) => void;
}

function CandidateRow({ item, onDecision, pending, onEvidenceOpen }: CandidateRowProps) {
  const [expanded, setExpanded] = useState(false);
  const [showNotes, setShowNotes] = useState(false);
  const [notes, setNotes] = useState('');
  const [snoozeHours, setSnoozeHours] = useState(4);
  const [confirmApprove, setConfirmApprove] = useState(false);
  const etfPolicy = etfPolicyFromMetadata(item.metadata);

  const submit = (action: 'approve' | 'reject' | 'snooze' | 'reanalyze') => {
    onDecision(item.id, action, { snoozeHours, notes: notes.trim() || undefined });
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
          <span className="ml-auto text-xs text-muted-foreground">Detected {fmtDate(item.detectedAt)}</span>
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
          {item.stopLoss != null && <span className="text-red-500">SL: ${item.stopLoss.toFixed(2)}</span>}
          {item.takeProfit != null && <span className="text-green-500">TP: ${item.takeProfit.toFixed(2)}</span>}
        </div>

        {item.reasoning && (
          <div>
            <button className="text-xs text-muted-foreground underline" onClick={() => setExpanded(!expanded)}>
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
              {etfPolicy.reasonCode && <span className="font-mono text-xs text-muted-foreground">{etfPolicy.reasonCode}</span>}
              {etfPolicy.catalogVersion && <span className="text-xs text-muted-foreground">Policy {etfPolicy.catalogVersion}</span>}
            </div>
            {etfPolicy.reason && <p className="mt-1 text-xs text-muted-foreground">{etfPolicy.reason}</p>}
          </div>
        ) : null}

        {showNotes && (
          <div className="flex flex-col gap-1">
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
              <CheckCircle className="mr-1 h-4 w-4" /> Approve
            </Button>
          ) : (
            <div className="flex items-center gap-1">
              <span className="text-xs text-muted-foreground">Confirm?</span>
              <Button
                size="sm"
                disabled={pending}
                onClick={() => submit('approve')}
                className="bg-green-600 text-white hover:bg-green-700"
              >
                Yes, approve
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setConfirmApprove(false)}>
                Cancel
              </Button>
            </div>
          )}

          <Button size="sm" variant="destructive" disabled={pending} onClick={() => submit('reject')}>
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

          <button className="ml-auto text-xs text-muted-foreground underline" onClick={() => setShowNotes(!showNotes)}>
            {showNotes ? 'Hide notes' : 'Add notes'}
          </button>
        </div>
      </CardContent>
    </Card>
  );
}

function SummaryCard({ label, value, hint }: { label: string; value: string | number; hint: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</p>
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
        <span className="ml-auto text-xs text-muted-foreground">Blocked {fmtDate(item.blockedAt ?? item.detectedAt)}</span>
      </div>
      <p className="mt-2 text-sm text-foreground/80">{item.blockReason ?? 'No block reason recorded.'}</p>
      {etfPolicy ? (
        <p className="mt-1 text-xs text-muted-foreground">
          ETF policy: {etfPolicy.reasonCode ?? 'unknown'}{etfPolicy.reason ? ` - ${etfPolicy.reason}` : ''}
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

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'approvals' });
  }, []);

  const { data: queue = [], isLoading, isError, refetch } = useQuery({
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
      opts?: { snoozeHours?: number; notes?: string };
    }) => {
      switch (action) {
        case 'approve':
          return approvalsService.approve(id, opts?.notes);
        case 'reject':
          return approvalsService.reject(id, opts?.notes);
        case 'snooze':
          return approvalsService.snooze(id, opts?.snoozeHours ?? 4, opts?.notes);
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
        executionStatus ? `Decision recorded: ${action}. Execution: ${executionStatus}` : `Decision recorded: ${action}`
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

  const handleDecision = (
    id: string,
    action: 'approve' | 'reject' | 'snooze' | 'reanalyze',
    opts?: { snoozeHours?: number; notes?: string }
  ) => {
    mutation.mutate({ id, action, opts });
  };

  const handleRefreshCandidate = (id: string) => {
    refreshMutation.mutate(id);
  };

  const handleEvidenceOpen = (item: ApprovalQueueItem | CandidateTrade) => {
    const routeType = 'status' in item && typeof item.status === 'string' ? item.status : 'approval_required';

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
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="mr-1 h-4 w-4" /> Refresh
        </Button>
      </div>

      {notification && (
        <div className="mb-4 rounded-md bg-accent px-4 py-2 text-sm text-accent-foreground">{notification}</div>
      )}

      <div className="mb-6 grid gap-3 md:grid-cols-3">
        <SummaryCard label="Pending approvals" value={queue.length} hint="Candidates waiting for a human decision." />
        <SummaryCard
          label="Recent execution chain"
          value={executionActivity.length}
          hint="Approved, submitted, or filled candidates."
        />
        <SummaryCard label="Recent blocked" value={blockedCandidates.length} hint="Latest blocked candidates with reason codes." />
      </div>

      {isLoading && <p className="text-muted-foreground">Loading approval queue...</p>}
      {isError && <p className="text-destructive">Failed to load approval queue. Check backend connectivity.</p>}

      {!isLoading && !isError && queue.length === 0 && (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">No candidates awaiting approval.</CardContent>
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
          <CardTitle>Recent Execution Activity</CardTitle>
          <p className="text-sm text-muted-foreground">
            Shows the approval {'->'} instruction {'->'} trade chain for the most recent paper-mode candidates.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {executionActivity.length === 0 ? (
            <p className="text-sm text-muted-foreground">No recent approved, submitted, or filled candidates.</p>
          ) : (
            executionActivity.map((item) => <ExecutionActivityRow key={`${item.status}-${item.id}`} item={item} />)
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
            <p className="text-sm text-muted-foreground">No blocked candidates found in the current window.</p>
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
