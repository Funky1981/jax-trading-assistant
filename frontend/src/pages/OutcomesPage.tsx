import { useMemo, useState } from 'react';
import { useQueries, useQuery } from '@tanstack/react-query';
import { ArrowLeft, ArrowRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { GlossaryTerm, PageIntro } from '@/components/ui/beginner-help';
import {
  operatorEvidenceService,
  type OperatorCandidateEvidence,
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
  if (checkpoint.status === 'pending_not_due') return 'Pending';
  if (['pending_market_data', 'insufficient_data'].includes(checkpoint.status))
    return 'Missing data';
  if (checkpoint.status === 'ambiguous_same_candle') return 'Ambiguous';
  if (checkpoint.targetTouched || checkpoint.status === 'target_touched') return 'Target touched';
  if (checkpoint.stopTouched || checkpoint.status === 'stop_touched') return 'Stop touched';
  if (checkpoint.status === 'completed') return 'Completed';
  return 'Unavailable';
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 break-words text-sm font-semibold">{value}</p>
    </div>
  );
}

function defaultCheckpoint(checkpoints: OutcomeCheckpoint[]) {
  const completed = checkpoints.filter((item) =>
    ['completed', 'target_touched', 'stop_touched', 'ambiguous_same_candle'].includes(item.status),
  );
  if (completed.length) return completed[completed.length - 1].name;
  return (
    checkpoints.find((item) => item.status === 'pending_not_due')?.name ?? checkpoints[0]?.name
  );
}

function SelectedCheckpoint({ checkpoint }: { checkpoint: OutcomeCheckpoint }) {
  const missingPrimary = [
    checkpoint.checkpointPrice,
    checkpoint.percentageReturn,
    checkpoint.hypotheticalPnl,
  ].some((value) => value === undefined);
  const hasAdvanced = [
    checkpoint.maximumFavourableExcursion,
    checkpoint.maximumAdverseExcursion,
    checkpoint.firstStopTouchAt,
    checkpoint.firstTargetTouchAt,
  ].some((value) => value !== undefined);
  return (
    <section
      aria-labelledby={`checkpoint-${checkpoint.name}`}
      className="rounded-lg border bg-card p-4"
    >
      <div className="flex flex-wrap items-center gap-2">
        <h3 id={`checkpoint-${checkpoint.name}`} className="mr-auto text-lg font-semibold">
          {checkpointName(checkpoint.name)} checkpoint
        </h3>
        <Badge variant={checkpoint.status === 'completed' ? 'success' : 'outline'}>
          {checkpointLabel(checkpoint)}
        </Badge>
      </div>
      {checkpoint.status === 'pending_not_due' && (
        <p className="mt-3 text-sm text-muted-foreground">This checkpoint is not due yet.</p>
      )}
      {['pending_market_data', 'insufficient_data'].includes(checkpoint.status) && (
        <p className="mt-3 text-sm text-muted-foreground">
          No suitable genuine market data was available for this checkpoint.
        </p>
      )}
      <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Field label="Due time" value={formatDate(checkpoint.dueAt)} />
        <Field label="Observation time" value={formatDate(checkpoint.observationAt)} />
        <Field label="Checkpoint price" value={money(checkpoint.checkpointPrice)} />
        <Field label="Market-data source" value={checkpoint.marketDataSource ?? 'Unavailable'} />
        <Field label="Hypothetical return" value={percent(checkpoint.percentageReturn)} />
        <Field label="Hypothetical P&L" value={money(checkpoint.hypotheticalPnl)} />
        <Field
          label="Stop result"
          value={checkpoint.stopTouched ? 'Stop touched' : 'Not touched'}
        />
        <Field
          label="Target result"
          value={checkpoint.targetTouched ? 'Target touched' : 'Not touched'}
        />
      </div>
      {missingPrimary && (
        <p className="mt-4 text-sm text-muted-foreground">
          Primary values marked unavailable were not persisted for this checkpoint.
        </p>
      )}
      <details className="mt-4 rounded-md border">
        <summary className="cursor-pointer px-3 py-2 text-sm font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          Show checkpoint details
        </summary>
        <div className="grid gap-4 border-t p-3 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Tracking start" value={formatDate(checkpoint.trackingStartedAt)} />
          {checkpoint.maximumFavourableExcursion !== undefined && (
            <Field label="MFE" value={money(checkpoint.maximumFavourableExcursion)} />
          )}
          {checkpoint.maximumAdverseExcursion !== undefined && (
            <Field label="MAE" value={money(checkpoint.maximumAdverseExcursion)} />
          )}
          {checkpoint.firstStopTouchAt && (
            <Field label="First stop-touch time" value={formatDate(checkpoint.firstStopTouchAt)} />
          )}
          {checkpoint.firstTargetTouchAt && (
            <Field
              label="First target-touch time"
              value={formatDate(checkpoint.firstTargetTouchAt)}
            />
          )}
          <Field label="Created" value={formatDate(checkpoint.createdAt)} />
          <Field label="Updated" value={formatDate(checkpoint.updatedAt)} />
          <Field label="Internal status" value={checkpoint.status} />
          {checkpoint.status === 'ambiguous_same_candle' && (
            <p className="col-span-full text-sm text-muted-foreground">
              The same market-data candle touched both levels, so the exact order cannot be known.
            </p>
          )}
          {!hasAdvanced && (
            <p className="col-span-full text-sm text-muted-foreground">
              Additional checkpoint information was not available.
            </p>
          )}
        </div>
      </details>
    </section>
  );
}

function OutcomeViewer({
  candidate,
  detail,
}: {
  candidate: OperatorCandidateSummary;
  detail: OperatorCandidateEvidence;
}) {
  const initial = defaultCheckpoint(detail.checkpoints);
  const [selectedName, setSelectedName] = useState(initial);
  const selected =
    detail.checkpoints.find((item) => item.name === selectedName) ?? detail.checkpoints[0];

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="space-y-4 pt-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-xl font-semibold">{candidate.symbol} paper plan</h2>
              <p className="text-sm text-muted-foreground">{candidate.setupType}</p>
            </div>
            <Badge variant="outline">{candidate.paperTicketStatus}</Badge>
          </div>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
            <Field label="Entry" value={money(detail.entry)} />
            <Field label="Stop" value={money(detail.stop)} />
            <Field label="Target" value={money(detail.target)} />
            <Field
              label="Quantity"
              value={detail.quantity === undefined ? 'Unavailable' : String(detail.quantity)}
            />
            <Field label="Completed" value={String(candidate.completedCheckpoints)} />
            <Field label="Pending" value={String(candidate.pendingCheckpoints)} />
            <Field label="Missing" value={String(candidate.missingCheckpoints)} />
            <Field label="Ambiguous" value={String(candidate.ambiguousCheckpoints)} />
          </div>
          <details className="rounded-md border">
            <summary className="cursor-pointer px-3 py-2 text-sm font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
              Show paper-plan details
            </summary>
            <div className="grid gap-4 border-t p-3 sm:grid-cols-2 lg:grid-cols-4">
              <Field label="Paper-ticket ID" value={detail.paperTicketId ?? 'Unavailable'} />
              <Field label="Notional" value={money(detail.notional)} />
              <Field label="Planned maximum loss" value={money(detail.plannedRisk)} />
              <Field label="Planned reward" value={money(detail.plannedReward)} />
              <Field
                label="Reward/risk"
                value={
                  detail.rewardRisk === undefined
                    ? 'Unavailable'
                    : `${detail.rewardRisk.toFixed(2)}R`
                }
              />
              <Field
                label="Leverage"
                value={
                  detail.leverage === undefined ? 'Unavailable' : `${detail.leverage.toFixed(2)}x`
                }
              />
              <Field
                label="Account-equity assumption"
                value={money(detail.accountEquityAssumption)}
              />
            </div>
          </details>
        </CardContent>
      </Card>
      {detail.checkpoints.length === 0 ? (
        <p className="rounded-lg border bg-card p-6 text-muted-foreground">
          No hypothetical outcome checkpoints are available yet.
        </p>
      ) : (
        <>
          <div
            className="flex max-w-full gap-2 overflow-x-auto rounded-lg border bg-card p-2"
            role="tablist"
            aria-label="Outcome checkpoint"
          >
            {detail.checkpoints.map((checkpoint) => (
              <button
                key={checkpoint.name}
                type="button"
                role="tab"
                aria-selected={selected?.name === checkpoint.name}
                className={`min-w-28 rounded-md border px-3 py-2 text-left text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                  selected?.name === checkpoint.name
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-background'
                }`}
                onClick={() => setSelectedName(checkpoint.name)}
              >
                <span className="block font-semibold">{checkpointName(checkpoint.name)}</span>
                <span className="block text-xs opacity-80">{checkpointLabel(checkpoint)}</span>
              </button>
            ))}
          </div>
          {selected && <SelectedCheckpoint checkpoint={selected} />}
        </>
      )}
      <div className="flex flex-wrap gap-2">
        <Button asChild variant="outline">
          <Link to={`/candidates/${candidate.candidateId}/evidence`}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Open Candidate Review
          </Link>
        </Button>
        <Button asChild variant="outline">
          <Link to={`/system?candidateId=${candidate.candidateId}`}>
            Open System Safety
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </div>
    </div>
  );
}

export function OutcomesPage() {
  const candidates = useQuery({
    queryKey: ['operator-candidates'],
    queryFn: operatorEvidenceService.candidates,
  });
  const plans = useMemo(
    () => (candidates.data ?? []).filter((candidate) => candidate.paperTicketId),
    [candidates.data],
  );
  const details = useQueries({
    queries: plans.map((candidate) => ({
      queryKey: ['operator-candidate-evidence', candidate.candidateId],
      queryFn: () => operatorEvidenceService.candidate(candidate.candidateId),
    })),
  });
  const [selectedPlan, setSelectedPlan] = useState(0);
  const safePlan = Math.min(selectedPlan, Math.max(0, plans.length - 1));

  return (
    <div className="mx-auto w-full max-w-5xl space-y-4">
      <PageIntro
        eyebrow="Retrospective evidence"
        title="Hypothetical Outcomes"
        description="Review one persisted checkpoint at a time. These calculations are not trades or realised profit and loss."
      >
        <p className="rounded-md border border-primary/40 bg-primary/5 p-3 text-sm font-semibold">
          Hypothetical — no order, fill or position exists.
        </p>
      </PageIntro>
      <details className="rounded-lg border bg-card">
        <summary className="cursor-pointer px-4 py-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          How to understand these results
        </summary>
        <div className="grid gap-3 border-t p-4 text-sm md:grid-cols-2">
          <GlossaryTerm term="Hypothetical return" />
          <GlossaryTerm term="Hypothetical P&L" />
          <GlossaryTerm term="MFE" />
          <GlossaryTerm term="MAE" />
          <GlossaryTerm term="Same-candle ambiguity" />
          <GlossaryTerm term="Checkpoint" />
        </div>
      </details>
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
        <>
          {plans.length > 1 && (
            <div className="flex max-w-full gap-2 overflow-x-auto" aria-label="Paper plans">
              {plans.map((plan, index) => (
                <Button
                  key={plan.candidateId}
                  type="button"
                  size="sm"
                  variant={safePlan === index ? 'default' : 'outline'}
                  onClick={() => setSelectedPlan(index)}
                >
                  {plan.symbol} paper plan
                </Button>
              ))}
            </div>
          )}
          {details[safePlan]?.isPending ? (
            <p className="text-muted-foreground">Loading persisted checkpoints…</p>
          ) : details[safePlan]?.isError || !details[safePlan]?.data ? (
            <p role="alert" className="text-destructive">
              Jax could not load this evidence. Your data has not been changed.
            </p>
          ) : (
            <OutcomeViewer candidate={plans[safePlan]} detail={details[safePlan].data} />
          )}
        </>
      )}
    </div>
  );
}
