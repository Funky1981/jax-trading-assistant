import { useQuery } from '@tanstack/react-query';
import { ArrowRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { PageIntro } from '@/components/ui/beginner-help';
import {
  operatorEvidenceService,
  type OperatorCandidateSummary,
} from '@/data/operator-evidence-service';

function formatDate(value?: string) {
  if (!value) return 'Not supplied';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function candidateState(candidate: OperatorCandidateSummary) {
  if (candidate.blockReason || candidate.candidateStatus === 'blocked') return 'Blocked by risk';
  if (candidate.candidateStatus === 'expired') return 'Expired';
  if (candidate.completedCheckpoints > 0 || candidate.latestOutcomeStatus)
    return 'Outcomes available';
  if (candidate.paperTicketId) return 'Paper plan created';
  if (candidate.decisionProvenance === 'human' && candidate.humanDecision === 'approved')
    return 'Human approved for paper review';
  if (candidate.decisionProvenance === 'human' && candidate.humanDecision === 'rejected')
    return 'Human rejected';
  return 'Awaiting human review';
}

function decisionState(candidate: OperatorCandidateSummary) {
  if (candidate.decisionProvenance === 'non_human') return 'Historical non-human record';
  if (candidate.decisionProvenance === 'human' && candidate.humanDecision === 'approved')
    return 'Human approved';
  if (candidate.decisionProvenance === 'human' && candidate.humanDecision === 'rejected')
    return 'Human rejected';
  return 'No human decision';
}

function CandidateCard({ candidate }: { candidate: OperatorCandidateSummary }) {
  const expired = candidate.expiresAt
    ? new Date(candidate.expiresAt).getTime() < Date.now()
    : false;
  const checkpointCount =
    candidate.completedCheckpoints + candidate.pendingCheckpoints + candidate.missingCheckpoints;
  return (
    <Card>
      <CardHeader className="gap-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold">{candidate.symbol}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{candidate.setupType}</p>
          </div>
          <Badge variant={candidate.blockReason ? 'warning' : 'outline'}>
            {candidateState(candidate)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm leading-6">{candidate.reason}</p>
        {candidate.blockReason && (
          <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
            Blocked because: {candidate.blockReason}
          </p>
        )}
        <dl className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <dt className="text-muted-foreground">Candidate state</dt>
            <dd className="font-semibold">{candidateState(candidate)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Human decision</dt>
            <dd className="font-semibold">{decisionState(candidate)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Paper plan</dt>
            <dd className="font-semibold">
              {candidate.paperTicketId ? 'Persisted paper plan available' : 'No paper plan'}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Outcome checkpoints</dt>
            <dd className="font-semibold">
              {checkpointCount ? `${checkpointCount} persisted` : 'No outcome data yet'}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Created</dt>
            <dd>{formatDate(candidate.createdAt)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Expiry</dt>
            <dd>
              {expired
                ? `Expired — ${formatDate(candidate.expiresAt)}`
                : formatDate(candidate.expiresAt)}
            </dd>
          </div>
        </dl>
        <Button asChild variant="outline">
          <Link to={`/candidates/${candidate.candidateId}/evidence`}>
            Open Candidate Review
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </CardContent>
    </Card>
  );
}

export function CandidatesPage() {
  const query = useQuery({
    queryKey: ['operator-candidates'],
    queryFn: operatorEvidenceService.candidates,
  });
  return (
    <div className="mx-auto w-full max-w-5xl space-y-6">
      <PageIntro
        eyebrow="Evidence review"
        title="Candidates"
        description="Review why candidates exist, what a human decided and whether a hypothetical paper plan has outcome evidence."
      >
        <p className="rounded-md border border-primary/40 bg-primary/5 p-3 text-sm font-semibold">
          Read only — candidates are not recommendations and this page has no approval or execution
          controls.
        </p>
      </PageIntro>
      {query.isPending ? (
        <p className="text-muted-foreground">Loading persisted candidates…</p>
      ) : query.isError ? (
        <p role="alert" className="text-destructive">
          Jax could not load this evidence. Your data has not been changed.
        </p>
      ) : query.data.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Jax has not created any candidates yet. This is a valid outcome while it gathers and
            rejects weak evidence.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {query.data.map((candidate) => (
            <CandidateCard key={candidate.candidateId} candidate={candidate} />
          ))}
        </div>
      )}
    </div>
  );
}
