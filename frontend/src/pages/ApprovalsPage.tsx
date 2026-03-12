import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle, XCircle, Clock, RefreshCw, AlertTriangle } from 'lucide-react';
import { approvalsService, type ApprovalQueueItem } from '@/data/approvals-service';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';

function fmtDate(raw?: string | null) {
  if (!raw) return '-';
  return new Date(raw).toLocaleString();
}

function ConfidenceBadge({ value }: { value?: number }) {
  if (value == null) return <span className="text-muted-foreground">—</span>;
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

interface CandidateRowProps {
  item: ApprovalQueueItem;
  onDecision: (id: string, action: 'approve' | 'reject' | 'snooze' | 'reanalyze') => void;
  pending: boolean;
}

function CandidateRow({ item, onDecision, pending }: CandidateRowProps) {
  const [expanded, setExpanded] = useState(false);
  return (
    <Card className="mb-3">
      <CardHeader className="pb-2 pt-3 px-4">
        <div className="flex flex-wrap items-center gap-3">
          <SignalBadge type={item.signalType} />
          <span className="font-semibold text-lg">{item.symbol}</span>
          <ConfidenceBadge value={item.confidence} />
          <span className="text-xs text-muted-foreground ml-auto">
            Detected {fmtDate(item.detectedAt)}
          </span>
          {item.expiresAt && (
            <span className="text-xs text-yellow-500 flex items-center gap-1">
              <Clock className="h-3 w-3" /> Expires {fmtDate(item.expiresAt)}
            </span>
          )}
        </div>
        <p className="text-xs text-muted-foreground mt-1">Strategy: {item.instanceName}</p>
      </CardHeader>
      <CardContent className="px-4 pb-3 space-y-3">
        {/* Price levels */}
        <div className="flex gap-4 text-sm flex-wrap">
          {item.entryPrice != null && <span>Entry: <strong>${item.entryPrice.toFixed(2)}</strong></span>}
          {item.stopLoss != null && <span className="text-red-500">SL: ${item.stopLoss.toFixed(2)}</span>}
          {item.takeProfit != null && <span className="text-green-500">TP: ${item.takeProfit.toFixed(2)}</span>}
        </div>

        {/* Reasoning toggle */}
        {item.reasoning && (
          <div>
            <button
              className="text-xs text-muted-foreground underline"
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? 'Hide reasoning' : 'Show reasoning'}
            </button>
            {expanded && (
              <p className="mt-1 text-sm text-foreground/80 bg-muted rounded p-2 whitespace-pre-wrap">
                {item.reasoning}
              </p>
            )}
          </div>
        )}

        {/* Block reason */}
        {item.blockReason && (
          <div className="flex items-center gap-2 text-sm text-yellow-600">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>{item.blockReason}</span>
          </div>
        )}

        {/* Actions */}
        <div className="flex flex-wrap gap-2 pt-1">
          <Button
            size="sm"
            disabled={pending}
            onClick={() => onDecision(item.id, 'approve')}
            className="bg-green-600 hover:bg-green-700 text-white"
          >
            <CheckCircle className="h-4 w-4 mr-1" /> Approve
          </Button>
          <Button
            size="sm"
            variant="destructive"
            disabled={pending}
            onClick={() => onDecision(item.id, 'reject')}
          >
            <XCircle className="h-4 w-4 mr-1" /> Reject
          </Button>
          <Button
            size="sm"
            variant="outline"
            disabled={pending}
            onClick={() => onDecision(item.id, 'snooze')}
          >
            <Clock className="h-4 w-4 mr-1" /> Snooze 4h
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={pending}
            onClick={() => onDecision(item.id, 'reanalyze')}
          >
            <RefreshCw className="h-4 w-4 mr-1" /> Re-analyse
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

export function ApprovalsPage() {
  const qc = useQueryClient();
  const [notification, setNotification] = useState<string | null>(null);

  const { data: queue = [], isLoading, isError, refetch } = useQuery({
    queryKey: ['approvals-queue'],
    queryFn: () => approvalsService.getQueue(),
    refetchInterval: 30_000,
  });

  const mutation = useMutation({
    mutationFn: async ({ id, action }: { id: string; action: 'approve' | 'reject' | 'snooze' | 'reanalyze' }) => {
      switch (action) {
        case 'approve': return approvalsService.approve(id);
        case 'reject': return approvalsService.reject(id);
        case 'snooze': return approvalsService.snooze(id, 4);
        case 'reanalyze': return approvalsService.reanalyze(id);
      }
    },
    onSuccess: (_data, { action }) => {
      qc.invalidateQueries({ queryKey: ['approvals-queue'] });
      setNotification(`Decision recorded: ${action}`);
      setTimeout(() => setNotification(null), 3000);
    },
    onError: (err: Error) => {
      setNotification(`Error: ${err.message}`);
      setTimeout(() => setNotification(null), 5000);
    },
  });

  const handleDecision = (id: string, action: 'approve' | 'reject' | 'snooze' | 'reanalyze') => {
    mutation.mutate({ id, action });
  };

  return (
    <div className="container mx-auto p-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Approval Queue</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Candidate trades awaiting your decision. AI is advisory only — you remain in control.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4 mr-1" /> Refresh
        </Button>
      </div>

      {/* Notification */}
      {notification && (
        <div className="mb-4 rounded-md bg-accent text-accent-foreground px-4 py-2 text-sm">
          {notification}
        </div>
      )}

      {/* Content */}
      {isLoading && <p className="text-muted-foreground">Loading approval queue…</p>}
      {isError && <p className="text-destructive">Failed to load approval queue. Check backend connectivity.</p>}

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
          pending={mutation.isPending}
        />
      ))}
    </div>
  );
}
