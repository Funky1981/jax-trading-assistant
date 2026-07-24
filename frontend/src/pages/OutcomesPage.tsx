import { useQueries, useQuery } from '@tanstack/react-query';
import { ArrowLeft, ArrowRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { GlossaryTerm, PageIntro } from '@/components/ui/beginner-help';
import {
  operatorEvidenceService,
  type OperatorCandidateSummary,
  type OutcomeCheckpoint,
} from '@/data/operator-evidence-service';

function formatDate(value?: string) {
  if (!value) return 'Unavailable';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function money(value?: number) {
  return value === undefined
    ? 'Unavailable'
    : new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value);
}

function percent(value?: number) {
  return value === undefined ? 'Unavailable' : `${value.toFixed(2)}%`;
}

function checkpointName(name: string) {
  return (
    ({ '1h': '1 hour', '1d': '1 day', '1w': '1 week' } as Record<string, string>)[name] ?? name
  );
}

function checkpointLabel(checkpoint: OutcomeCheckpoint) {
  switch (checkpoint.status) {
    case 'pending_not_due':
      return 'PENDING — NOT DUE';
    case 'pending_market_data':
    case 'insufficient_data':
      return 'MISSING MARKET DATA';
    case 'ambiguous_same_candle':
      return 'AMBIGUOUS SAME CANDLE';
    case 'target_touched':
      return 'TARGET TOUCHED';
    case 'stop_touched':
      return 'STOP TOUCHED';
    case 'completed':
      return 'COMPLETED';
    default:
      return 'UNAVAILABLE';
  }
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md border bg-muted/20 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 break-words text-sm font-semibold">{value}</p>
    </div>
  );
}

function CheckpointCard({ checkpoint }: { checkpoint: OutcomeCheckpoint }) {
  const missing =
    checkpoint.status === 'pending_market_data' || checkpoint.status === 'insufficient_data';
  return (
    <article
      className="rounded-lg border p-4"
      aria-label={`${checkpointName(checkpoint.name)} checkpoint`}
    >
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <h3 className="mr-auto text-lg font-semibold">{checkpointName(checkpoint.name)}</h3>
        <Badge variant="secondary">HYPOTHETICAL — NOT A FILL</Badge>
        <Badge variant={checkpoint.status === 'completed' ? 'success' : 'warning'}>
          {checkpointLabel(checkpoint)}
        </Badge>
        {checkpoint.targetTouched && <Badge variant="success">TARGET TOUCHED</Badge>}
        {checkpoint.stopTouched && <Badge variant="destructive">STOP TOUCHED</Badge>}
      </div>
      {checkpoint.status === 'pending_not_due' && (
        <p className="mb-4 text-sm text-muted-foreground">This checkpoint is not due yet.</p>
      )}
      {missing && (
        <p className="mb-4 text-sm text-muted-foreground">
          No suitable genuine market data was available for this checkpoint.
        </p>
      )}
      {checkpoint.status === 'ambiguous_same_candle' && (
        <p className="mb-4 text-sm text-muted-foreground">
          The same market-data candle touched both levels, so the exact order cannot be known.
        </p>
      )}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Fact label="Tracking start" value={formatDate(checkpoint.trackingStartedAt)} />
        <Fact label="Due time" value={formatDate(checkpoint.dueAt)} />
        <Fact label="Status" value={checkpointLabel(checkpoint)} />
        <Fact label="Observation time" value={formatDate(checkpoint.observationAt)} />
        <Fact
          label="Checkpoint price — not an execution price"
          value={money(checkpoint.checkpointPrice)}
        />
        <Fact label="Market-data source" value={checkpoint.marketDataSource ?? 'Unavailable'} />
        <Fact
          label="Hypothetical return — not realised"
          value={percent(checkpoint.percentageReturn)}
        />
        <Fact
          label="Hypothetical P&L — no money changed hands"
          value={money(checkpoint.hypotheticalPnl)}
        />
        <Fact
          label="MFE — best hypothetical movement"
          value={money(checkpoint.maximumFavourableExcursion)}
        />
        <Fact
          label="MAE — worst hypothetical movement"
          value={money(checkpoint.maximumAdverseExcursion)}
        />
        <Fact
          label="Stop touched"
          value={checkpoint.stopTouched ? 'Yes — no position existed' : 'No'}
        />
        <Fact
          label="Target touched"
          value={checkpoint.targetTouched ? 'Yes — no position existed' : 'No'}
        />
        <Fact label="First stop-touch time" value={formatDate(checkpoint.firstStopTouchAt)} />
        <Fact label="First target-touch time" value={formatDate(checkpoint.firstTargetTouchAt)} />
        <Fact label="Created" value={formatDate(checkpoint.createdAt)} />
        <Fact label="Updated" value={formatDate(checkpoint.updatedAt)} />
      </div>
    </article>
  );
}

function OutcomePlan({
  candidate,
  detail,
  loading,
  error,
}: {
  candidate: OperatorCandidateSummary;
  detail?: Awaited<ReturnType<typeof operatorEvidenceService.candidate>>;
  loading: boolean;
  error: boolean;
}) {
  return (
    <Card id={`paper-plan-${candidate.candidateId}`}>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold">{candidate.symbol} paper plan</h2>
            <p className="mt-1 text-sm text-muted-foreground">{candidate.setupType}</p>
          </div>
          <Badge variant="outline">
            {candidate.paperTicketStatus || 'Paper-plan state unavailable'}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="font-semibold">HYPOTHETICAL PAPER PLAN — NOT AN ORDER OR FILL</p>
        {candidate.candidateStatus === 'expired' && (
          <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
            Candidate expired for new approval. Existing hypothetical outcome records remain
            available for review.
          </p>
        )}
        <div className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
          <Fact label="Completed checkpoints" value={String(candidate.completedCheckpoints)} />
          <Fact label="Pending checkpoints" value={String(candidate.pendingCheckpoints)} />
          <Fact label="Missing-data checkpoints" value={String(candidate.missingCheckpoints)} />
          <Fact label="Ambiguous checkpoints" value={String(candidate.ambiguousCheckpoints)} />
        </div>
        {loading ? (
          <p className="text-muted-foreground">Loading persisted checkpoints…</p>
        ) : error || !detail ? (
          <p role="alert" className="text-destructive">
            Jax could not load this evidence. Your data has not been changed.
          </p>
        ) : detail.checkpoints.length === 0 ? (
          <p className="text-muted-foreground">
            No hypothetical outcome checkpoints are available yet.
          </p>
        ) : (
          <div className="space-y-4">
            {detail.checkpoints.map((checkpoint) => (
              <CheckpointCard key={checkpoint.name} checkpoint={checkpoint} />
            ))}
          </div>
        )}
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link to={`/candidates/${candidate.candidateId}/evidence`}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Open Candidate Review
            </Link>
          </Button>
          <Button asChild variant="outline">
            <Link to="/system">
              Open System Safety
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

export function OutcomesPage() {
  const candidates = useQuery({
    queryKey: ['operator-candidates'],
    queryFn: operatorEvidenceService.candidates,
  });
  const plans = (candidates.data ?? []).filter((candidate) => candidate.paperTicketId);
  const details = useQueries({
    queries: plans.map((candidate) => ({
      queryKey: ['operator-candidate-evidence', candidate.candidateId],
      queryFn: () => operatorEvidenceService.candidate(candidate.candidateId),
    })),
  });

  return (
    <div className="mx-auto w-full max-w-6xl space-y-6">
      <PageIntro
        eyebrow="Retrospective evidence"
        title="Hypothetical Outcomes"
        description="Review what happened after a hypothetical paper plan. These are retrospective calculations, not trades or realised profit and loss."
      >
        <p className="rounded-md border border-primary/40 bg-primary/5 p-3 text-sm font-semibold">
          No order, fill, position or realised profit exists on this page.
        </p>
      </PageIntro>
      <Card>
        <CardHeader>
          <h2 className="text-lg font-semibold">How to read checkpoint evidence</h2>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm md:grid-cols-2">
          <GlossaryTerm term="Hypothetical return" />
          <GlossaryTerm term="Hypothetical P&L" />
          <GlossaryTerm term="MFE" />
          <GlossaryTerm term="MAE" />
          <GlossaryTerm term="Same-candle ambiguity" />
          <GlossaryTerm term="Checkpoint" />
        </CardContent>
      </Card>
      {candidates.isPending ? (
        <p className="text-muted-foreground">Loading persisted paper plans…</p>
      ) : candidates.isError ? (
        <p role="alert" className="text-destructive">
          Jax could not load this evidence. Your data has not been changed.
        </p>
      ) : plans.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            No hypothetical paper plans are available.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-5">
          {plans.map((candidate, index) => (
            <OutcomePlan
              key={candidate.candidateId}
              candidate={candidate}
              detail={details[index]?.data}
              loading={details[index]?.isPending ?? true}
              error={details[index]?.isError ?? false}
            />
          ))}
        </div>
      )}
    </div>
  );
}
