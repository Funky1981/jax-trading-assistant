import { useMemo, useState } from 'react';
import { Brain, CheckCircle2, XCircle, AlertTriangle, Clock, Sparkles } from 'lucide-react';
import { CollapsiblePanel, StatusDot } from './CollapsiblePanel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAnalyzeSignal, useApproveSignal, useRecommendations, useRejectSignal, useSignals } from '@/hooks/useSignals';
import type { Recommendation, Signal } from '@/data/types';

interface SignalsQueuePanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

function formatConfidence(value?: number | null) {
  if (value === undefined || value === null || Number.isNaN(value)) return '—';
  return `${Math.round(value * 100)}%`;
}

function riskReward(signal: Signal) {
  if (!signal.entry_price || !signal.stop_loss || !signal.take_profit) return '—';
  const entry = signal.entry_price;
  const stop = signal.stop_loss;
  const target = signal.take_profit;
  if (signal.signal_type === 'SELL') {
    const risk = stop - entry;
    const reward = entry - target;
    if (risk <= 0 || reward <= 0) return '—';
    return (reward / risk).toFixed(2);
  }
  const risk = entry - stop;
  const reward = target - entry;
  if (risk <= 0 || reward <= 0) return '—';
  return (reward / risk).toFixed(2);
}

function getSignalBadgeVariant(signalType: string) {
  switch (signalType.toUpperCase()) {
    case 'BUY':
      return 'success';
    case 'SELL':
      return 'destructive';
    default:
      return 'secondary';
  }
}

function getStatusDot(status?: string) {
  switch (status) {
    case 'completed':
      return 'healthy';
    case 'failed':
      return 'error';
    case 'running':
      return 'warning';
    default:
      return 'degraded';
  }
}

function RecommendationCard({
  signal,
  recommendation,
  approver,
  onApprove,
  onReject,
  onAnalyze,
  isApproving,
  isRejecting,
  isAnalyzing,
}: {
  signal: Signal;
  recommendation?: Recommendation;
  approver: string;
  onApprove: () => void;
  onReject: () => void;
  onAnalyze: () => void;
  isApproving: boolean;
  isRejecting: boolean;
  isAnalyzing: boolean;
}) {
  const analysis = recommendation?.ai_analysis;
  const hasAI = Boolean(analysis);

  return (
    <div className="rounded-lg border border-border bg-muted/30 p-4 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h4 className="text-lg font-semibold">{signal.symbol}</h4>
            <Badge variant={getSignalBadgeVariant(signal.signal_type)} className="text-xs">
              {signal.signal_type.toUpperCase()}
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground">
            {signal.strategy_id} • {formatConfidence(signal.confidence)} confidence
          </p>
        </div>
        <Badge variant="outline" className="text-xs">
          {signal.status}
        </Badge>
      </div>

      <div className="grid grid-cols-3 gap-3 text-sm">
        <div>
          <p className="text-xs text-muted-foreground">Entry</p>
          <p className="font-mono">{signal.entry_price?.toFixed(2) ?? '—'}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Stop</p>
          <p className="font-mono text-red-500">{signal.stop_loss?.toFixed(2) ?? '—'}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Target</p>
          <p className="font-mono text-emerald-500">{signal.take_profit?.toFixed(2) ?? '—'}</p>
        </div>
      </div>

      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>R:R {riskReward(signal)}</span>
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {new Date(signal.generated_at).toLocaleString()}
        </span>
      </div>

      {signal.reasoning && (
        <div className="rounded-md bg-muted/50 p-3 text-sm text-muted-foreground">
          <span className="font-medium text-foreground">Strategy:</span> {signal.reasoning}
        </div>
      )}

      <div className="rounded-md border border-border bg-background p-3 space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <StatusDot status={getStatusDot(analysis?.status)} />
            <span className="text-sm font-medium">AI Analysis</span>
          </div>
          {analysis?.status && (
            <Badge variant="outline" className="text-xs">
              {analysis.status}
            </Badge>
          )}
        </div>
        {analysis ? (
          <div className="space-y-2 text-sm">
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-primary" />
              <span className="font-medium">{analysis.agent_suggestion ?? 'Recommendation'}</span>
              <span className="text-muted-foreground">
                {formatConfidence(analysis.confidence)}
              </span>
            </div>
            {analysis.reasoning && (
              <p className="text-muted-foreground">{analysis.reasoning}</p>
            )}
            {analysis.error && (
              <p className="text-destructive flex items-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                {analysis.error}
              </p>
            )}
          </div>
        ) : (
          <div className="text-xs text-muted-foreground">
            No AI analysis yet. Run analysis to get Jax’s recommendation.
          </div>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          onClick={onApprove}
          disabled={!approver.trim() || isApproving}
          className="flex-1 min-w-[140px]"
        >
          <CheckCircle2 className="h-4 w-4 mr-2" />
          {isApproving ? 'Approving...' : 'Approve Trade'}
        </Button>
        <Button
          variant="outline"
          onClick={onReject}
          disabled={!approver.trim() || isRejecting}
          className="flex-1 min-w-[140px]"
        >
          <XCircle className="h-4 w-4 mr-2" />
          {isRejecting ? 'Rejecting...' : 'Reject'}
        </Button>
        {!hasAI && (
          <Button
            variant="secondary"
            onClick={onAnalyze}
            disabled={isAnalyzing}
            className="min-w-[140px]"
          >
            <Brain className="h-4 w-4 mr-2" />
            {isAnalyzing ? 'Analyzing...' : 'Run AI'}
          </Button>
        )}
      </div>
    </div>
  );
}

export function SignalsQueuePanel({ isOpen, onToggle }: SignalsQueuePanelProps) {
  const [approver, setApprover] = useState('dashboard@local');
  const { data: signalsResponse, isLoading } = useSignals({ status: 'pending', limit: 20 });
  const { data: recResponse } = useRecommendations(20, 0);

  const approveMutation = useApproveSignal();
  const rejectMutation = useRejectSignal();
  const analyzeMutation = useAnalyzeSignal();

  const recommendationMap = useMemo(() => {
    const map = new Map<string, Recommendation>();
    recResponse?.recommendations?.forEach((rec) => {
      map.set(rec.signal.id, rec);
    });
    return map;
  }, [recResponse]);

  const signals = signalsResponse?.signals ?? [];
  const summary = (
    <span>
      {signals.length} pending â€¢ {recResponse?.recommendations?.length ?? 0} with AI
    </span>
  );

  return (
    <CollapsiblePanel
      title="Trading Approvals"
      icon={<Brain className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <StatusDot status={signals.length > 0 ? 'healthy' : 'degraded'} />
            <span>{signals.length > 0 ? 'Ready for review' : 'No pending signals'}</span>
          </div>
          <div className="flex items-center gap-2">
            <Input
              value={approver}
              onChange={(e) => setApprover(e.target.value)}
              placeholder="Approved by"
              className="h-8 w-48"
            />
          </div>
        </div>

        {signals.length === 0 && (
          <div className="rounded-md border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            No pending signals right now. When new signals arrive, they’ll appear here for approval.
          </div>
        )}

        {signals.map((signal) => {
          const rec = recommendationMap.get(signal.id);
          return (
            <RecommendationCard
              key={signal.id}
              signal={signal}
              recommendation={rec}
              approver={approver}
              onApprove={() => approveMutation.mutate({ signalId: signal.id, approvedBy: approver })}
              onReject={() => rejectMutation.mutate({ signalId: signal.id, approvedBy: approver })}
              onAnalyze={() => analyzeMutation.mutate({ signalId: signal.id })}
              isApproving={approveMutation.isPending && approveMutation.variables?.signalId === signal.id}
              isRejecting={rejectMutation.isPending && rejectMutation.variables?.signalId === signal.id}
              isAnalyzing={analyzeMutation.isPending && analyzeMutation.variables?.signalId === signal.id}
            />
          );
        })}
      </div>
    </CollapsiblePanel>
  );
}
