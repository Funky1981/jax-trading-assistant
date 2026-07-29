import { useMemo, useState, type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, ExternalLink, RefreshCw } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageIntro } from '@/components/ui/beginner-help';
import { aiService, type WorldMonitorInboxItem } from '@/data/ai-service';
import { HttpError } from '@/data/http-client';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';
import { isPaperSafe } from '@/lib/operator-safety';

type BeginnerFilter =
  | 'all'
  | 'genuine'
  | 'synthetic'
  | 'rejected'
  | 'no_trade'
  | 'watch'
  | 'candidate';

const FILTERS: Array<{ value: BeginnerFilter; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'genuine', label: 'Genuine' },
  { value: 'synthetic', label: 'Synthetic tests' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'no_trade', label: 'NO_TRADE' },
  { value: 'watch', label: 'WATCH' },
  { value: 'candidate', label: 'CANDIDATE' },
];

function formatTime(value?: string) {
  if (!value) return 'Not supplied';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Unavailable';
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function sourceName(item: WorldMonitorInboxItem) {
  const firstUrl = item.sourceUrls?.[0];
  if (firstUrl) {
    try {
      return new URL(firstUrl).hostname.replace(/^www\./, '');
    } catch {
      // Use the persisted source label below.
    }
  }
  return item.source || 'Source unavailable';
}

function isDuplicate(item: WorldMonitorInboxItem) {
  return (
    item.status === 'ignored' &&
    /dedup|duplicate/i.test(item.rejectionReason || item.operatorReason || '')
  );
}

function disposition(item: WorldMonitorInboxItem) {
  if (item.decision) return item.decision.decision;
  if (item.status === 'rejected') return 'Rejected';
  if (isDuplicate(item)) return 'Duplicate ignored';
  if (item.status === 'new') return 'Awaiting processing';
  if (['researching', 'ignored'].includes(item.status)) return 'Research only';
  return 'Unknown';
}

function shortExplanation(item: WorldMonitorInboxItem) {
  if (item.decision)
    return item.decision.reasons.slice(0, 2).join(' ') || 'Jax persisted a deterministic decision.';
  if (item.status === 'rejected')
    return item.rejectionReason || 'Jax rejected this evidence during validation.';
  if (isDuplicate(item)) return 'Jax recognised evidence it had already received.';
  if (item.status === 'new')
    return 'No candidate has been created. Jax is awaiting a persisted processing decision.';
  if (item.normalizedEventId)
    return 'No candidate was created. This is a valid research-only outcome.';
  return item.operatorReason || 'No candidate was created. This is a valid outcome.';
}

function journeySummary(item: WorldMonitorInboxItem) {
  if (isDuplicate(item)) return 'Duplicate ignored';
  if (item.status === 'rejected') return 'Received → Rejected';
  if (item.decision) return `Received → Normalised → ${item.decision.decision}`;
  if (item.normalizedEventId) return 'Received → Normalised → Research only';
  return 'Awaiting processing';
}

function matchesFilter(item: WorldMonitorInboxItem, filter: BeginnerFilter) {
  if (filter === 'genuine') return item.provenanceAvailable !== false && item.isSynthetic === false;
  if (filter === 'synthetic')
    return item.provenanceAvailable !== false && item.isSynthetic === true;
  if (filter === 'rejected') return item.status === 'rejected';
  if (filter === 'no_trade') return item.decision?.decision === 'NO_TRADE';
  if (filter === 'watch') return item.decision?.decision === 'WATCH';
  if (filter === 'candidate') return item.decision?.decision === 'CANDIDATE';
  return true;
}

function emptyMessage(filter: BeginnerFilter) {
  if (filter === 'genuine') return 'No genuine evidence matches this filter.';
  if (filter === 'synthetic') return 'No synthetic test evidence matches this filter.';
  if (filter === 'rejected') return 'No rejected evidence matches this filter.';
  if (filter === 'no_trade') return 'No evidence in this view has a persisted NO_TRADE decision.';
  if (filter === 'watch') return 'No evidence in this view has a persisted WATCH decision.';
  if (filter === 'candidate') return 'No evidence in this view created a candidate.';
  return 'No evidence has arrived yet. Genuine and controlled test events will appear here after Jax receives them.';
}

function EvidenceBadge({ item }: { item: WorldMonitorInboxItem }) {
  if (item.provenanceAvailable === false)
    return <Badge variant="outline">PROVENANCE UNAVAILABLE</Badge>;
  if (item.isSynthetic === true) return <Badge variant="warning">SYNTHETIC TEST</Badge>;
  if (item.isSynthetic === false) return <Badge variant="success">GENUINE</Badge>;
  return <Badge variant="outline">PROVENANCE UNAVAILABLE</Badge>;
}

function DispositionBadge({ item }: { item: WorldMonitorInboxItem }) {
  const label = disposition(item);
  const variant =
    label === 'CANDIDATE'
      ? 'success'
      : label === 'Rejected'
        ? 'destructive'
        : label === 'Awaiting processing' || label === 'WATCH'
          ? 'warning'
          : 'secondary';
  return <Badge variant={variant}>{label}</Badge>;
}

export function MonitorInboxPage() {
  const [filter, setFilter] = useState<BeginnerFilter>('all');
  const [expandedID, setExpandedID] = useState<string>();
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(1);
  const safety = useOperatorEvidenceOverview();
  const inbox = useQuery({
    queryKey: ['world-monitor-inbox', 'beginner'],
    queryFn: () => aiService.getWorldMonitorInbox({ limit: 100 }),
  });
  const items = useMemo(() => inbox.data?.items ?? [], [inbox.data?.items]);
  const filtered = useMemo(
    () => items.filter((item) => matchesFilter(item, filter)),
    [filter, items],
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const safePage = Math.min(page, pageCount);
  const start = (safePage - 1) * pageSize;
  const visible = filtered.slice(start, start + pageSize);
  const authError = inbox.error instanceof HttpError && [401, 403].includes(inbox.error.status);

  const resetView = () => {
    setPage(1);
    setExpandedID(undefined);
  };
  const selectFilter = (value: BeginnerFilter) => {
    setFilter(value);
    resetView();
  };

  return (
    <div className="min-w-0 space-y-4">
      <PageIntro
        eyebrow="Review evidence"
        title="Evidence Inbox"
        description="Review genuine and controlled test evidence received by Jax. Opening an item does not approve or place a trade."
      >
        <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
          <span>Filter the inbox, then expand one item when you need its persisted detail.</span>
          <Link className="font-medium text-primary underline" to="/guide#evidence-inbox">
            Read the Evidence Inbox guide
          </Link>
        </div>
      </PageIntro>

      {safety.isError || (safety.data && !isPaperSafe(safety.data)) ? (
        <section
          role="status"
          className="rounded-lg border border-warning/60 bg-warning/5 p-3 text-sm"
        >
          Jax cannot confirm runtime safety. Evidence remains read-only; check{' '}
          <Link className="font-medium text-primary underline" to="/system">
            System Safety
          </Link>{' '}
          before relying on this view.
        </section>
      ) : null}

      {inbox.isLoading ? (
        <p role="status" aria-live="polite">
          Loading evidence…
        </p>
      ) : inbox.isError ? (
        <Card role="alert">
          <CardHeader>
            <CardTitle>
              {authError ? 'Sign in to view evidence' : 'Evidence Inbox unavailable'}
            </CardTitle>
            <CardDescription>
              {authError
                ? 'Your session does not currently allow access to this protected evidence.'
                : 'Jax could not load the Evidence Inbox. Your data has not been changed.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button variant="outline" onClick={() => void inbox.refetch()}>
              <RefreshCw aria-hidden="true" /> Try again
            </Button>
          </CardContent>
        </Card>
      ) : (
        <>
          <section aria-labelledby="evidence-summary-heading">
            <h2 id="evidence-summary-heading" className="sr-only">
              Evidence summary
            </h2>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <SummaryMetric label="NO_TRADE" value={inbox.data?.counts.noTrade} />
              <SummaryMetric label="WATCH" value={inbox.data?.counts.watch} />
              <SummaryMetric label="CANDIDATE" value={inbox.data?.counts.candidate} />
              <SummaryMetric
                label="Awaiting processing"
                value={inbox.data?.counts.awaitingProcessing}
              />
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              Most evidence should be rejected or watched. Candidates are created only when the full
              deterministic ruleset passes.
            </p>
          </section>

          <section aria-labelledby="evidence-list-heading" className="min-w-0 space-y-3">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <div>
                <h2 id="evidence-list-heading" className="text-xl font-semibold">
                  Evidence received
                </h2>
                <p className="text-sm text-muted-foreground">
                  Ten compact records per page. Technical fields stay collapsed.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => void inbox.refetch()}
                disabled={inbox.isFetching}
              >
                <RefreshCw aria-hidden="true" /> {inbox.isFetching ? 'Refreshing…' : 'Refresh'}
              </Button>
            </div>

            <div className="flex flex-col gap-3 rounded-lg border bg-card p-3 md:flex-row md:items-center md:justify-between">
              <div aria-label="Evidence filters" className="flex max-w-full flex-wrap gap-2">
                {FILTERS.map((option) => (
                  <Button
                    key={option.value}
                    size="sm"
                    variant={filter === option.value ? 'default' : 'outline'}
                    aria-pressed={filter === option.value}
                    onClick={() => selectFilter(option.value)}
                  >
                    {option.label}
                  </Button>
                ))}
              </div>
              <label className="flex items-center gap-2 text-sm">
                Per page
                <select
                  aria-label="Evidence items per page"
                  className="h-9 rounded-md border bg-background px-2"
                  value={pageSize}
                  onChange={(event) => {
                    setPageSize(Number(event.target.value));
                    resetView();
                  }}
                >
                  <option value={10}>10</option>
                  <option value={20}>20</option>
                </select>
              </label>
            </div>

            {filtered.length === 0 ? (
              <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
                {emptyMessage(filter)}
              </div>
            ) : (
              <>
                <ul className="min-w-0 space-y-2" aria-label="Evidence items">
                  {visible.map((item) => (
                    <EvidenceItem
                      key={item.id}
                      item={item}
                      expanded={expandedID === item.id}
                      onToggle={() =>
                        setExpandedID((current) => (current === item.id ? undefined : item.id))
                      }
                    />
                  ))}
                </ul>
                <nav
                  aria-label="Evidence pages"
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
                      aria-label="Previous evidence page"
                      disabled={safePage === 1}
                      onClick={() => {
                        setPage((value) => Math.max(1, value - 1));
                        setExpandedID(undefined);
                      }}
                    >
                      <ChevronLeft className="mr-1 h-4 w-4" />
                      Previous
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      aria-label="Next evidence page"
                      disabled={safePage === pageCount}
                      onClick={() => {
                        setPage((value) => Math.min(pageCount, value + 1));
                        setExpandedID(undefined);
                      }}
                    >
                      Next
                      <ChevronRight className="ml-1 h-4 w-4" />
                    </Button>
                  </div>
                </nav>
              </>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function SummaryMetric({ label, value }: { label: string; value: number | undefined }) {
  return (
    <div className="min-w-0 rounded-md border bg-card px-3 py-2">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="text-xl font-bold">{value === undefined ? 'Unavailable' : value}</p>
    </div>
  );
}

function EvidenceItem({
  item,
  expanded,
  onToggle,
}: {
  item: WorldMonitorInboxItem;
  expanded: boolean;
  onToggle: () => void;
}) {
  const panelID = `evidence-panel-${item.id}`;
  return (
    <li className="min-w-0 rounded-lg border bg-card">
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={panelID}
        onClick={onToggle}
        className="w-full min-w-0 p-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring sm:p-4"
      >
        <div className="grid min-w-0 gap-3 md:grid-cols-[minmax(0,2fr)_minmax(8rem,0.7fr)_minmax(9rem,0.8fr)_auto] md:items-center">
          <div className="min-w-0">
            <div className="flex flex-wrap gap-2">
              <EvidenceBadge item={item} />
              <DispositionBadge item={item} />
            </div>
            <h3
              className={
                expanded
                  ? 'mt-2 break-words font-semibold leading-snug'
                  : 'mt-2 line-clamp-2 break-words font-semibold leading-snug'
              }
            >
              {item.headline}
            </h3>
            <p className="mt-1 line-clamp-1 text-sm text-muted-foreground">
              {shortExplanation(item)}
            </p>
          </div>
          <div className="min-w-0 text-sm">
            <p className="text-xs text-muted-foreground">Source</p>
            <p className="truncate font-medium" title={sourceName(item)}>
              {sourceName(item)}
            </p>
          </div>
          <div className="text-sm">
            <p className="text-xs text-muted-foreground">Published</p>
            <p className="font-medium">{formatTime(item.publishedAt)}</p>
          </div>
          <span className="min-h-9 whitespace-nowrap rounded-md border px-3 py-2 text-center text-sm font-medium text-primary">
            {expanded ? 'Collapse details' : 'Expand details'}
          </span>
        </div>
      </button>
      {expanded ? (
        <div id={panelID} className="border-t p-3 sm:p-4">
          <EvidenceDetail item={item} />
        </div>
      ) : null}
    </li>
  );
}

function EvidenceDetail({ item }: { item: WorldMonitorInboxItem }) {
  return (
    <article className="min-w-0 space-y-3" aria-label={`${item.headline} details`}>
      <div className="grid gap-3 lg:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
        <div>
          <h4 className="font-semibold">{item.headline}</h4>
          <p className="mt-1 text-sm text-muted-foreground">
            {item.summary || 'No concise summary was supplied.'}
          </p>
          {item.sourceUrls.length ? (
            <a
              className="mt-2 inline-flex max-w-full items-center gap-1 break-all text-sm font-medium text-primary underline"
              href={item.sourceUrls[0]}
              target="_blank"
              rel="noreferrer"
            >
              Open original source <ExternalLink className="h-4 w-4 shrink-0" aria-hidden="true" />
            </a>
          ) : (
            <p className="mt-2 text-sm text-muted-foreground">Original source link unavailable.</p>
          )}
        </div>
        <div className="rounded-md border bg-muted/20 p-3 text-sm">
          <p className="font-semibold">{disposition(item)}</p>
          <p className="mt-1 break-all text-muted-foreground">{shortExplanation(item)}</p>
          {item.decision?.decision === 'CANDIDATE' && item.decision.candidateId ? (
            <Button asChild variant="outline" size="sm" className="mt-2">
              <Link to={`/candidates/${item.decision.candidateId}/evidence`}>
                Open Candidate Review
              </Link>
            </Button>
          ) : (
            <p className="mt-2 font-medium">No candidate was created. This is a valid outcome.</p>
          )}
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        <Definition label="Published" value={formatTime(item.publishedAt)} />
        <Definition label="Collected" value={formatTime(item.collectedAt)} />
        <Definition label="Received" value={formatTime(item.receivedAt)} />
        <Definition label="Decided" value={formatTime(item.decision?.decisionAt)} />
      </div>
      <p className="text-sm text-muted-foreground">{provenanceExplanation(item)}</p>
      {item.decision ? (
        <Disclosure title={`Decision — ${item.decision.decision}`}>
          <DecisionDetail item={item} />
        </Disclosure>
      ) : null}
      <Disclosure title="Source and provenance">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <Definition label="Source" value={sourceName(item)} />
          <Definition label="Original URL" value={item.sourceUrls[0] || 'Not supplied'} />
          <Definition label="Discovery method" value={item.discoveryMethod || 'Not supplied'} />
          <Definition label="Published" value={formatTime(item.publishedAt)} />
          <Definition label="Collected" value={formatTime(item.collectedAt)} />
          <Definition label="Received" value={formatTime(item.receivedAt)} />
          <Definition label="Decided" value={formatTime(item.decision?.decisionAt)} />
          <Definition
            label="Provenance availability"
            value={item.provenanceAvailable === false ? 'Unavailable' : 'Persisted'}
          />
        </div>
      </Disclosure>
      <Disclosure title="Analysis">
        <Analysis item={item} />
      </Disclosure>
      <Disclosure title={`Journey — ${journeySummary(item)}`}>
        <Journey item={item} />
      </Disclosure>
      <Disclosure title="Audit">
        <Audit item={item} />
      </Disclosure>
    </article>
  );
}

function provenanceExplanation(item: WorldMonitorInboxItem) {
  if (item.provenanceAvailable === false)
    return 'Jax does not have enough linked raw provenance to classify this evidence.';
  if (item.isSynthetic === true)
    return `Controlled test evidence, not live news.${item.syntheticReason ? ` ${item.syntheticReason}` : ''}`;
  if (item.isSynthetic === false)
    return 'Genuine evidence collected from a persisted external source.';
  return 'Persisted provenance is unavailable, so Jax does not infer genuine or synthetic status.';
}

function Disclosure({ title, children }: { title: string; children: ReactNode }) {
  return (
    <details className="min-w-0 rounded-md border">
      <summary className="cursor-pointer px-3 py-2 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
        {title}
      </summary>
      <div className="min-w-0 space-y-3 border-t p-3 text-sm">{children}</div>
    </details>
  );
}

function Analysis({ item }: { item: WorldMonitorInboxItem }) {
  const aiUsed = Boolean(item.aiProvider || item.aiModel);
  const missing = [
    !item.analysisIdentity && !aiUsed && 'analysis identity',
    !item.eventType && 'event type',
    item.confidenceReasons.length === 0 && 'confidence reasons',
  ].filter(Boolean) as string[];
  return (
    <>
      <div className="flex flex-wrap gap-2">
        {aiUsed ? (
          <Badge>AI ANALYSED</Badge>
        ) : item.analysisIdentity ? (
          <Badge variant="secondary">DETERMINISTIC ANALYSIS</Badge>
        ) : (
          <Badge variant="outline">ANALYSIS METADATA UNAVAILABLE</Badge>
        )}
      </div>
      {aiUsed ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <Definition label="AI provider" value={item.aiProvider || 'Not supplied'} />
          <Definition label="AI model" value={item.aiModel || 'Not supplied'} />
        </div>
      ) : (
        <>
          <p>Rules or configured logic were used. This was not an AI model call.</p>
          <Definition
            label="Deterministic identity"
            value={item.analysisIdentity || 'Not supplied'}
          />
          <strong>No AI used</strong>
        </>
      )}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <Definition label="Event type" value={item.eventType || 'Not supplied'} />
        <Definition
          label="Confidence"
          value={
            Number.isFinite(item.confidence)
              ? `${Math.round(item.confidence * 100)}%`
              : 'Not supplied'
          }
        />
        <Definition
          label="Affected assets"
          value={
            item.possibleAffectedEtfs.length
              ? item.possibleAffectedEtfs.join(', ')
              : 'Unknown assets'
          }
        />
        <Definition
          label="Confidence reasons"
          value={item.confidenceReasons.join(' ') || 'Not supplied'}
        />
        <Definition label="Mapping reason" value={item.mappingReason || 'Not supplied'} />
      </div>
      {item.possibleAffectedEtfs.length === 0 ? (
        <p className="text-muted-foreground">
          Unknown assets means no truthful persisted asset mapping exists; none was fabricated.
        </p>
      ) : null}
      {missing.length ? (
        <p className="text-muted-foreground">
          Additional analysis information not recorded: {missing.join(', ')}.
        </p>
      ) : null}
    </>
  );
}

function DecisionDetail({ item }: { item: WorldMonitorInboxItem }) {
  const decision = item.decision;
  if (!decision) return null;
  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <Definition label="Current decision" value={decision.decision} />
        <Definition label="Decision time" value={formatTime(decision.decisionAt)} />
        <Definition label="Ruleset version" value={decision.rulesetVersion} />
        <Definition label="Processing mode" value={decision.processingMode} />
        <Definition label="Deterministic processor" value={decision.processorIdentity} />
        <Definition
          label="Affected assets"
          value={decision.unknownAssets ? 'Unknown assets' : decision.affectedAssets.join(', ')}
        />
        <Definition label="Trust state" value={decision.trustGateState} />
        <Definition label="Risk state" value={decision.riskReviewState} />
        <Definition
          label="Candidate linkage"
          value={decision.candidateId || 'No candidate linked'}
        />
      </div>
      <Definition label="Reasons" value={decision.reasons.join(' ') || 'None persisted'} />
      <Definition
        label="Blockers"
        value={decision.blockingReasons.join(', ') || 'None persisted'}
      />
      <Definition
        label="Missing evidence"
        value={decision.missingEvidence.join(', ') || 'None persisted'}
      />
      {(item.decisionHistory?.length ?? 0) > 1 ? (
        <Disclosure title={`Decision history — ${item.decisionHistory?.length ?? 0} versions`}>
          <ol className="space-y-2">
            {item.decisionHistory?.map((history) => (
              <li key={history.decisionId} className="rounded border p-2">
                <strong>{history.decision}</strong> — {history.rulesetVersion}, version{' '}
                {history.decisionVersion}, {formatTime(history.decisionAt)}
              </li>
            ))}
          </ol>
        </Disclosure>
      ) : null}
    </div>
  );
}

function Journey({ item }: { item: WorldMonitorInboxItem }) {
  const candidate = item.decision?.decision === 'CANDIDATE' && Boolean(item.decision.candidateId);
  const stages = [
    ['Published', item.publishedAt ? 'Complete' : 'Not supplied'],
    ['Collected', item.collectedAt ? 'Complete' : 'Missing persisted evidence'],
    ['Delivered', 'Missing persisted evidence'],
    ['Received by Jax', item.receivedAt ? 'Complete' : 'Missing persisted evidence'],
    [
      'Validated',
      item.status === 'rejected' ? 'Rejected' : item.status ? 'Complete' : 'Awaiting processing',
    ],
    [
      'Normalised',
      item.normalizedEventId
        ? 'Complete'
        : item.status === 'rejected'
          ? 'Not applicable'
          : 'Not run',
    ],
    [
      'Decision processed',
      item.decision
        ? item.decision.decision
        : item.status === 'new'
          ? 'Awaiting processing'
          : item.status === 'rejected'
            ? 'Rejected'
            : isDuplicate(item)
              ? 'Duplicate ignored'
              : item.operatorDecision || item.status
                ? 'Complete'
                : 'Missing persisted evidence',
    ],
    ['Candidate created', candidate ? 'Complete' : 'No candidate created'],
    [
      'Human approval',
      !candidate
        ? 'Not applicable'
        : item.approvalId
          ? item.approvalDecision || 'Complete'
          : 'Not run',
    ],
    ['Paper ticket', !candidate ? 'Not applicable' : item.paperTicketId ? 'Complete' : 'Not run'],
    [
      'Outcomes',
      !candidate || !item.paperTicketId
        ? 'Not applicable'
        : item.outcomeCount > 0
          ? 'Complete'
          : 'Not run',
    ],
  ];
  return (
    <ol className="grid gap-2 md:grid-cols-2">
      {stages.map(([label, state], index) => (
        <li key={label} className="flex min-w-0 items-start gap-2 rounded border p-2">
          <span
            aria-hidden="true"
            className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border text-xs font-semibold"
          >
            {index + 1}
          </span>
          <div className="min-w-0">
            <strong className="text-sm">{label}</strong>
            <p className="break-words text-xs text-muted-foreground">{state}</p>
          </div>
        </li>
      ))}
    </ol>
  );
}

function Audit({ item }: { item: WorldMonitorInboxItem }) {
  return (
    <>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <Definition label="Inbox ID" value={item.id} />
        <Definition label="Source-event ID" value={item.sourceEventId || 'Not supplied'} />
        <Definition
          label="Normalised-event ID"
          value={item.normalizedEventId || 'Not applicable'}
        />
        <Definition label="Candidate ID" value={item.candidateId || 'Not applicable'} />
        <Definition label="Original status" value={item.status || 'Not supplied'} />
        <Definition label="Complete provenance" value={item.source || 'Not supplied'} />
        {item.rejectionReason ? (
          <Definition label="Technical rejection data" value={item.rejectionReason} />
        ) : null}
      </div>
      <Disclosure title="Show raw payload">
        <pre className="max-w-full overflow-auto whitespace-pre-wrap break-all text-xs">
          {JSON.stringify(item.rawPayload ?? {}, null, 2)}
        </pre>
      </Disclosure>
    </>
  );
}

function Definition({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="break-words">{value}</p>
    </div>
  );
}
