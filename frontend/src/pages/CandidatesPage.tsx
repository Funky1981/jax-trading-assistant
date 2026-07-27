import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ArrowRight, ChevronLeft, ChevronRight, Search } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { PageIntro } from '@/components/ui/beginner-help';
import {
  operatorEvidenceService,
  type OperatorCandidateSummary,
} from '@/data/operator-evidence-service';

type CandidateFilter =
  | 'all'
  | 'needs_review'
  | 'approved'
  | 'rejected'
  | 'paper_plan'
  | 'outcomes'
  | 'expired';

const filters: Array<{ value: CandidateFilter; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'needs_review', label: 'Needs review' },
  { value: 'approved', label: 'Human approved' },
  { value: 'rejected', label: 'Human rejected' },
  { value: 'paper_plan', label: 'Paper plan created' },
  { value: 'outcomes', label: 'Outcomes available' },
  { value: 'expired', label: 'Expired' },
];

function formatDate(value?: string) {
  if (!value) return 'Not supplied';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function isExpired(candidate: OperatorCandidateSummary) {
  return (
    candidate.candidateStatus === 'expired' ||
    Boolean(candidate.expiresAt && new Date(candidate.expiresAt).getTime() < Date.now())
  );
}

function candidateState(candidate: OperatorCandidateSummary) {
  if (candidate.blockReason || candidate.candidateStatus === 'blocked') return 'Blocked by risk';
  if (isExpired(candidate)) return 'Expired';
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

function planState(candidate: OperatorCandidateSummary) {
  if (candidate.completedCheckpoints > 0 || candidate.latestOutcomeStatus)
    return 'Outcomes available';
  return candidate.paperTicketId ? 'Paper plan created' : 'No paper plan';
}

function matchesFilter(candidate: OperatorCandidateSummary, filter: CandidateFilter) {
  switch (filter) {
    case 'needs_review':
      return candidate.decisionProvenance === 'none';
    case 'approved':
      return candidate.decisionProvenance === 'human' && candidate.humanDecision === 'approved';
    case 'rejected':
      return candidate.decisionProvenance === 'human' && candidate.humanDecision === 'rejected';
    case 'paper_plan':
      return Boolean(candidate.paperTicketId);
    case 'outcomes':
      return candidate.completedCheckpoints > 0 || Boolean(candidate.latestOutcomeStatus);
    case 'expired':
      return isExpired(candidate);
    default:
      return true;
  }
}

function Summary({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border bg-card px-3 py-2">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-xl font-bold">{value}</p>
    </div>
  );
}

function CandidateRow({ candidate }: { candidate: OperatorCandidateSummary }) {
  return (
    <article className="rounded-lg border bg-card p-4">
      <div className="grid items-start gap-3 md:grid-cols-[minmax(8rem,0.7fr)_1fr_1fr_auto]">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold">{candidate.symbol}</h2>
          <p className="truncate text-sm text-muted-foreground" title={candidate.setupType}>
            {candidate.setupType}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">{formatDate(candidate.createdAt)}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge variant={candidate.blockReason ? 'warning' : 'outline'}>
            {candidateState(candidate)}
          </Badge>
          <Badge variant="outline">{decisionState(candidate)}</Badge>
        </div>
        <div>
          <p className="text-sm font-medium">{planState(candidate)}</p>
          <p className="mt-1 line-clamp-2 text-sm text-muted-foreground" title={candidate.reason}>
            {candidate.reason}
          </p>
        </div>
        <Button asChild variant="outline" size="sm">
          <Link to={`/candidates/${candidate.candidateId}/evidence`}>
            Open Candidate Review
            <ArrowRight className="ml-2 h-4 w-4" />
          </Link>
        </Button>
      </div>
    </article>
  );
}

export function CandidatesPage() {
  const query = useQuery({
    queryKey: ['operator-candidates'],
    queryFn: operatorEvidenceService.candidates,
  });
  const [filter, setFilter] = useState<CandidateFilter>('all');
  const [search, setSearch] = useState('');
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(1);

  const candidates = useMemo(() => query.data ?? [], [query.data]);
  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return candidates.filter(
      (candidate) =>
        matchesFilter(candidate, filter) &&
        (!term || candidate.symbol.toLowerCase().includes(term)),
    );
  }, [candidates, filter, search]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const safePage = Math.min(page, pageCount);
  const start = (safePage - 1) * pageSize;
  const visible = filtered.slice(start, start + pageSize);
  const setCurrentFilter = (value: CandidateFilter) => {
    setFilter(value);
    setPage(1);
  };

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4">
      <PageIntro
        eyebrow="Evidence review"
        title="Candidates"
        description="Scan current candidate states, human decisions and hypothetical follow-up."
      />
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Summary label="All candidates" value={candidates.length} />
        <Summary
          label="Needs review"
          value={candidates.filter((item) => matchesFilter(item, 'needs_review')).length}
        />
        <Summary
          label="Paper plans"
          value={candidates.filter((item) => matchesFilter(item, 'paper_plan')).length}
        />
        <Summary
          label="With outcomes"
          value={candidates.filter((item) => matchesFilter(item, 'outcomes')).length}
        />
      </div>
      <section aria-label="Candidate filters" className="space-y-3 rounded-lg border bg-card p-3">
        <div className="flex flex-col gap-3 md:flex-row md:items-center">
          <label className="relative min-w-0 flex-1">
            <span className="sr-only">Search by symbol</span>
            <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              value={search}
              onChange={(event) => {
                setSearch(event.target.value);
                setPage(1);
              }}
              placeholder="Search by symbol"
              className="pl-9"
            />
          </label>
          <label className="flex items-center gap-2 text-sm">
            Per page
            <select
              aria-label="Candidates per page"
              className="h-9 rounded-md border bg-background px-2"
              value={pageSize}
              onChange={(event) => {
                setPageSize(Number(event.target.value));
                setPage(1);
              }}
            >
              <option value={10}>10</option>
              <option value={20}>20</option>
            </select>
          </label>
        </div>
        <div
          className="flex max-w-full gap-2 overflow-x-auto pb-1"
          role="group"
          aria-label="Candidate state"
        >
          {filters.map((item) => (
            <Button
              key={item.value}
              type="button"
              size="sm"
              variant={filter === item.value ? 'default' : 'outline'}
              aria-pressed={filter === item.value}
              onClick={() => setCurrentFilter(item.value)}
            >
              {item.label}
            </Button>
          ))}
        </div>
      </section>
      {query.isPending ? (
        <p className="text-muted-foreground">Loading persisted candidates…</p>
      ) : query.isError ? (
        <p role="alert" className="text-destructive">
          Jax could not load this evidence. Your data has not been changed.
        </p>
      ) : candidates.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-muted-foreground">
            Jax has not created any candidates yet. This is a valid outcome while it gathers and
            rejects weak evidence.
          </CardContent>
        </Card>
      ) : filtered.length === 0 ? (
        <p className="rounded-lg border bg-card p-6 text-center text-muted-foreground">
          No candidates match this search and filter.
        </p>
      ) : (
        <>
          <div className="space-y-2" aria-live="polite">
            {visible.map((candidate) => (
              <CandidateRow key={candidate.candidateId} candidate={candidate} />
            ))}
          </div>
          <nav
            aria-label="Candidate pages"
            className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-3"
          >
            <p className="text-sm text-muted-foreground">
              {start + 1}–{Math.min(start + pageSize, filtered.length)} of {filtered.length}
            </p>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={safePage === 1}
                aria-label="Previous candidate page"
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                <ChevronLeft className="mr-1 h-4 w-4" />
                Previous
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={safePage === pageCount}
                aria-label="Next candidate page"
                onClick={() => setPage((value) => Math.min(pageCount, value + 1))}
              >
                Next
                <ChevronRight className="ml-1 h-4 w-4" />
              </Button>
            </div>
          </nav>
        </>
      )}
    </div>
  );
}
