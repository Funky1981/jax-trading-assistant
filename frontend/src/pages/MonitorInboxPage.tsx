import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ExternalLink, RefreshCw } from 'lucide-react';
import { aiService, type WorldMonitorInboxItem } from '@/data/ai-service';
import { HttpError } from '@/data/http-client';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';
import { isPaperSafe } from '@/lib/operator-safety';
import { PageIntro } from '@/components/ui/beginner-help';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { cn } from '@/lib/utils';

type BeginnerFilter = 'all' | 'genuine' | 'synthetic' | 'rejected' | 'candidate';

const FILTERS: { value: BeginnerFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'genuine', label: 'Genuine' },
  { value: 'synthetic', label: 'Synthetic tests' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'candidate', label: 'Candidate created' },
];

function formatTime(value?: string): string {
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

function sourceName(item: WorldMonitorInboxItem): string {
  const firstUrl = item.sourceUrls?.[0];
  if (firstUrl) {
    try {
      return new URL(firstUrl).hostname.replace(/^www\./, '');
    } catch {
      // Fall back to the persisted source label.
    }
  }
  return item.source || 'Source unavailable';
}

function isDuplicate(item: WorldMonitorInboxItem): boolean {
  return (
    item.status === 'ignored' &&
    /dedup|duplicate/i.test(item.rejectionReason || item.operatorReason || '')
  );
}

function disposition(item: WorldMonitorInboxItem): string {
  if (item.candidateId) return 'Candidate created';
  if (item.status === 'rejected') return 'Rejected';
  if (isDuplicate(item)) return 'Duplicate ignored';
  if (item.status === 'new') return 'Awaiting processing';
  if (item.status === 'researching') return 'Research only';
  if (item.status === 'ignored') return 'Research only';
  return 'Unknown';
}

function shortExplanation(item: WorldMonitorInboxItem): string {
  if (item.candidateId) return 'Jax created a candidate for separate human review.';
  if (item.status === 'rejected')
    return item.rejectionReason || 'Jax rejected this evidence during validation.';
  if (isDuplicate(item)) return 'Jax recognised evidence it had already received.';
  if (item.status === 'new')
    return 'No candidate has been created. Jax is awaiting a persisted processing decision.';
  if (item.normalizedEventId)
    return 'No candidate was created. This is a valid research-only outcome.';
  return item.operatorReason || 'No further action is recorded.';
}

function matchesFilter(item: WorldMonitorInboxItem, filter: BeginnerFilter): boolean {
  if (filter === 'genuine')
    return item.provenanceAvailable !== false && item.isSynthetic === false;
  if (filter === 'synthetic')
    return item.provenanceAvailable !== false && item.isSynthetic === true;
  if (filter === 'rejected') return item.status === 'rejected';
  if (filter === 'candidate') return Boolean(item.candidateId);
  return true;
}

function emptyMessage(filter: BeginnerFilter): string {
  if (filter === 'genuine') return 'No genuine evidence matches this filter.';
  if (filter === 'rejected') return 'No rejected evidence matches this filter.';
  if (filter === 'candidate') return 'No evidence in this view created a candidate.';
  if (filter === 'synthetic') return 'No synthetic test evidence matches this filter.';
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
    label === 'Candidate created'
      ? 'success'
      : label === 'Rejected'
        ? 'destructive'
        : label === 'Awaiting processing'
          ? 'warning'
          : 'secondary';
  return <Badge variant={variant}>{label}</Badge>;
}

export function MonitorInboxPage() {
  const [filter, setFilter] = useState<BeginnerFilter>('all');
  const [selectedID, setSelectedID] = useState<string>();
  const detailRef = useRef<HTMLDivElement>(null);
  const safety = useOperatorEvidenceOverview();
  const inbox = useQuery({
    queryKey: ['world-monitor-inbox', 'beginner'],
    queryFn: () => aiService.getWorldMonitorInbox({ limit: 100 }),
  });

  const visibleItems = useMemo(
    () => (inbox.data?.items ?? []).filter((item) => matchesFilter(item, filter)),
    [filter, inbox.data?.items],
  );

  useEffect(() => {
    if (!visibleItems.some((item) => item.id === selectedID)) {
      setSelectedID(undefined);
    }
  }, [selectedID, visibleItems]);

  useEffect(() => {
    if (selectedID && window.matchMedia?.('(max-width: 1023px)').matches) {
      detailRef.current?.scrollIntoView?.({ behavior: 'smooth', block: 'start' });
    }
  }, [selectedID]);

  const selected = visibleItems.find((item) => item.id === selectedID);
  const authError = inbox.error instanceof HttpError && [401, 403].includes(inbox.error.status);

  return (
    <div className="min-w-0 space-y-6">
      <PageIntro
        eyebrow="Review evidence"
        title="Evidence Inbox"
        description="Review genuine and controlled test evidence received by Jax. Opening an item does not approve or place a trade."
      >
        <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
          <span>Choose a filter, open an item, then read what Jax did next.</span>
          <Link className="font-medium text-primary underline" to="/guide#evidence-inbox">
            Read the Evidence Inbox guide
          </Link>
        </div>
      </PageIntro>

      {safety.isError || (safety.data && !isPaperSafe(safety.data)) ? (
        <section
          role="status"
          className="rounded-lg border border-warning/60 bg-warning/5 p-4 text-sm"
        >
          Jax cannot confirm runtime safety. Evidence remains read-only, but check{' '}
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
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-5">
              <SummaryMetric label="Genuine" value={inbox.data?.counts.genuine} />
              <SummaryMetric label="Synthetic tests" value={inbox.data?.counts.syntheticTests} />
              <SummaryMetric label="Rejected" value={inbox.data?.counts.rejected} />
              <SummaryMetric label="Duplicates" value={inbox.data?.counts.duplicates} />
              <SummaryMetric
                label="Candidates created"
                value={inbox.data?.counts.candidatesCreated}
              />
            </div>
          </section>

          <section aria-labelledby="evidence-list-heading" className="min-w-0 space-y-4">
            <div className="flex flex-wrap items-end justify-between gap-3">
              <div>
                <h2 id="evidence-list-heading" className="text-xl font-semibold">
                  Evidence received
                </h2>
                <p className="text-sm text-muted-foreground">
                  Technical IDs and raw payloads remain inside Audit details.
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

            <div aria-label="Evidence filters" className="flex max-w-full flex-wrap gap-2">
              {FILTERS.map((option) => (
                <Button
                  key={option.value}
                  size="sm"
                  variant={filter === option.value ? 'default' : 'outline'}
                  aria-pressed={filter === option.value}
                  onClick={() => setFilter(option.value)}
                >
                  {option.label}
                </Button>
              ))}
            </div>

            {visibleItems.length === 0 ? (
              <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
                {emptyMessage(filter)}
              </div>
            ) : (
              <div className="grid min-w-0 gap-5 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
                <ul className="min-w-0 space-y-3" aria-label="Evidence items">
                  {visibleItems.map((item) => (
                    <li key={item.id}>
                      <button
                        type="button"
                        aria-expanded={selectedID === item.id}
                        onClick={() => setSelectedID(item.id)}
                        className={cn(
                          'w-full min-w-0 rounded-lg border bg-card p-4 text-left transition-colors',
                          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                          selectedID === item.id
                            ? 'border-primary ring-1 ring-primary'
                            : 'hover:bg-accent/40',
                        )}
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <EvidenceBadge item={item} />
                          <DispositionBadge item={item} />
                        </div>
                        <h3 className="mt-3 break-words font-semibold leading-snug">
                          {item.headline}
                        </h3>
                        <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
                          <div>
                            <dt className="text-xs text-muted-foreground">Source</dt>
                            <dd className="break-words">{sourceName(item)}</dd>
                          </div>
                          <div>
                            <dt className="text-xs text-muted-foreground">Published</dt>
                            <dd>{formatTime(item.eventTime)}</dd>
                          </div>
                        </dl>
                        <p className="mt-3 text-sm text-muted-foreground">
                          {shortExplanation(item)}
                        </p>
                        <span className="mt-3 inline-block text-sm font-medium text-primary">
                          {selectedID === item.id ? 'Details open' : 'Open details'}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>

                <div
                  ref={detailRef}
                  className="order-first min-w-0 scroll-mt-4 lg:order-none lg:sticky lg:top-4 lg:self-start"
                >
                  {selected ? (
                    <EvidenceDetail item={selected} />
                  ) : (
                    <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
                      Open an evidence item to see its source, timestamps, analysis, journey and
                      audit details.
                    </div>
                  )}
                </div>
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function SummaryMetric({ label, value }: { label: string; value: number | undefined }) {
  return (
    <div className="min-w-0 rounded-lg border bg-card p-4">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-1 text-2xl font-bold">{value === undefined ? 'Unavailable' : value}</p>
    </div>
  );
}

function EvidenceDetail({ item }: { item: WorldMonitorInboxItem }) {
  const aiUsed = Boolean(item.aiProvider || item.aiModel);
  return (
    <article className="min-w-0 space-y-4" aria-labelledby={`evidence-detail-${item.id}`}>
      <Card className="min-w-0">
        <CardHeader>
          <div className="flex flex-wrap gap-2">
            <EvidenceBadge item={item} />
            <DispositionBadge item={item} />
          </div>
          <CardTitle id={`evidence-detail-${item.id}`} className="break-words text-xl">
            {item.headline}
          </CardTitle>
          <CardDescription>{shortExplanation(item)}</CardDescription>
        </CardHeader>
        <CardContent className="min-w-0 space-y-6">
          <DetailSection title="What happened">
            <p>{item.summary || 'No summary was supplied.'}</p>
            <Definition label="Source" value={sourceName(item)} />
            {item.sourceUrls.length > 0 ? (
              <ul className="space-y-2">
                {item.sourceUrls.map((url, index) => (
                  <li key={url}>
                    <a
                      className="inline-flex max-w-full items-start gap-1 break-all text-primary underline"
                      href={url}
                      target="_blank"
                      rel="noreferrer"
                    >
                      Open original article {index + 1}
                      <ExternalLink className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
                    </a>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">Original article link unavailable.</p>
            )}
          </DetailSection>

          <DetailSection title="Is this genuine?">
            <EvidenceBadge item={item} />
            <p className="text-sm text-muted-foreground">
              {item.provenanceAvailable === false
                ? 'Jax does not have enough linked raw provenance to classify this evidence.'
                : item.isSynthetic === true
                ? `This is controlled test evidence, not live news.${item.syntheticReason ? ` ${item.syntheticReason}` : ''}`
                : item.isSynthetic === false
                  ? 'This evidence was collected from a real external source.'
                  : 'Jax does not have enough persisted provenance to classify this evidence.'}
            </p>
            <Definition label="Discovery method" value={item.discoveryMethod || 'Unavailable'} />
          </DetailSection>

          <DetailSection title="When did it happen?">
            <div className="grid gap-3 sm:grid-cols-3">
              <Definition label="Published time" value={formatTime(item.eventTime)} />
              <Definition label="Collected time" value={formatTime(item.collectedAt)} />
              <Definition label="Received by Jax" value={formatTime(item.receivedAt)} />
            </div>
            <p className="text-xs text-muted-foreground">
              Times are displayed in your local timezone. Jax retains the persisted values in UTC.
            </p>
          </DetailSection>

          <DetailSection title="How was it analysed?">
            {aiUsed ? (
              <>
                <Badge>AI ANALYSED</Badge>
                <Definition label="AI provider" value={item.aiProvider || 'Unavailable'} />
                <Definition label="AI model" value={item.aiModel || 'Unavailable'} />
              </>
            ) : item.analysisIdentity ? (
              <>
                <Badge variant="secondary">DETERMINISTIC ANALYSIS</Badge>
                <p className="text-sm text-muted-foreground">
                  Rules or configured logic were used. This was not an AI model call.
                </p>
                <Definition label="Analysis identity" value={item.analysisIdentity} />
                <strong className="text-sm">No AI used</strong>
              </>
            ) : (
              <>
                <Badge variant="outline">NO ANALYSIS METADATA</Badge>
                <strong className="text-sm">No AI used</strong>
              </>
            )}
          </DetailSection>

          <DetailSection title="What did Jax recognise?">
            <div className="grid gap-3 sm:grid-cols-2">
              <Definition label="Event type" value={item.eventType || 'Unavailable'} />
              <Definition label="Severity" value={item.severity || 'Unavailable'} />
              <Definition
                label="Confidence"
                value={
                  Number.isFinite(item.confidence)
                    ? `${Math.round(item.confidence * 100)}%`
                    : 'Unavailable'
                }
              />
              <Definition
                label="Affected assets"
                value={
                  item.possibleAffectedEtfs.length > 0
                    ? item.possibleAffectedEtfs.join(', ')
                    : 'Unknown assets'
                }
              />
            </div>
            <Definition
              label="Confidence reasons"
              value={
                item.confidenceReasons.length > 0
                  ? item.confidenceReasons.join(' ')
                  : 'Not supplied'
              }
            />
            <Definition label="Mapping reason" value={item.mappingReason || 'Not supplied'} />
          </DetailSection>

          <DetailSection title="What did Jax do next?">
            <DispositionBadge item={item} />
            <p>{shortExplanation(item)}</p>
            {item.status === 'rejected' ? (
              <Definition
                label="Rejection reason"
                value={item.rejectionReason || 'Reason unavailable'}
              />
            ) : null}
            {item.candidateId ? (
              <div className="rounded-lg border bg-muted/20 p-3">
                <p className="font-medium">
                  {item.candidateSymbol || 'Symbol unavailable'} ·{' '}
                  {item.candidateStatus || 'Status unavailable'}
                </p>
                <Button asChild className="mt-3" variant="outline">
                  <Link to={`/candidates/${item.candidateId}/evidence`}>Open Candidate Review</Link>
                </Button>
              </div>
            ) : null}
          </DetailSection>
        </CardContent>
      </Card>

      <Journey item={item} />

      <details className="min-w-0 rounded-lg border bg-card">
        <summary className="cursor-pointer px-4 py-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          Audit details
        </summary>
        <div className="min-w-0 space-y-4 border-t p-4 text-sm">
          <div className="grid min-w-0 gap-3 sm:grid-cols-2">
            <Definition label="Source-event ID" value={item.sourceEventId || 'Unavailable'} />
            <Definition
              label="World Monitor event ID"
              value={item.worldMonitorEventId || 'Unavailable'}
            />
            <Definition label="Inbox ID" value={item.id || 'Unavailable'} />
            <Definition label="Raw event ID" value={item.rawEventId || 'Unavailable'} />
            <Definition
              label="Normalised event ID"
              value={item.normalizedEventId || 'Unavailable'}
            />
            <Definition label="Candidate ID" value={item.candidateId || 'Not applicable'} />
            <Definition label="Original status" value={item.status || 'Unavailable'} />
            <Definition label="Provenance" value={item.source || 'Unavailable'} />
          </div>
          <details className="min-w-0 rounded border">
            <summary className="cursor-pointer p-3 font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
              Raw payload
            </summary>
            <pre className="max-h-96 max-w-full overflow-auto border-t p-3 text-xs">
              {JSON.stringify(item.rawPayload ?? {}, null, 2)}
            </pre>
          </details>
        </div>
      </details>
    </article>
  );
}

function DetailSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="min-w-0 space-y-3 border-t pt-5 first:border-0 first:pt-0">
      <h3 className="font-semibold">{title}</h3>
      {children}
    </section>
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

function Journey({ item }: { item: WorldMonitorInboxItem }) {
  const candidate = Boolean(item.candidateId);
  const stages = [
    {
      label: 'Discovered',
      state: item.eventTime ? 'Complete' : 'Missing persisted evidence',
      time: item.eventTime,
      explanation: 'The source published or identified the event.',
    },
    {
      label: 'Collected',
      state: item.collectedAt ? 'Complete' : 'Missing persisted evidence',
      time: item.collectedAt,
      explanation: 'The source collector retained the evidence.',
    },
    {
      label: 'Delivered',
      state: 'Missing persisted evidence',
      explanation: 'A separate delivery timestamp is not retained.',
    },
    {
      label: 'Received by Jax',
      state: item.receivedAt ? 'Complete' : 'Missing persisted evidence',
      time: item.receivedAt,
      explanation: 'Jax persisted the Inbox record.',
    },
    {
      label: 'Validated',
      state:
        item.status === 'rejected' ? 'Rejected' : item.status ? 'Complete' : 'Awaiting processing',
      time: item.receivedAt,
      explanation:
        item.status === 'rejected'
          ? item.rejectionReason || 'Validation rejected the evidence.'
          : 'The persisted Inbox status records a validation result.',
    },
    {
      label: 'Normalised',
      state: item.normalizedEventId
        ? 'Complete'
        : item.status === 'rejected'
          ? 'Not applicable'
          : 'Not run',
      time: item.normalizedAt,
      explanation: item.normalizedEventId
        ? 'Jax created a normalised research event.'
        : 'No normalised record is linked.',
    },
    {
      label: 'Decision processed',
      state:
        item.status === 'new'
          ? 'Awaiting processing'
          : item.status === 'rejected'
            ? 'Rejected'
            : isDuplicate(item)
              ? 'Duplicate ignored'
              : item.operatorDecision || item.status
                ? 'Complete'
                : 'Missing persisted evidence',
      time: item.candidateCreatedAt,
      explanation: shortExplanation(item),
    },
    {
      label: 'Candidate created',
      state: candidate ? 'Complete' : 'No candidate created',
      time: item.candidateCreatedAt,
      explanation: candidate
        ? 'A separate candidate is available for human review.'
        : 'Research-only evidence does not need to create a candidate.',
    },
    {
      label: 'Human approval',
      state: !candidate
        ? 'Not applicable'
        : item.approvalId
          ? item.approvalDecision || 'Complete'
          : 'Not run',
      time: item.approvalAt,
      explanation: item.approvalId
        ? 'A persisted human decision is linked.'
        : 'No linked human approval record exists.',
    },
    {
      label: 'Paper ticket',
      state: !candidate ? 'Not applicable' : item.paperTicketId ? 'Complete' : 'Not run',
      time: item.paperTicketCreatedAt,
      explanation: item.paperTicketId
        ? 'A hypothetical paper ticket is linked. It is not an order.'
        : 'No paper ticket is linked.',
    },
    {
      label: 'Outcomes',
      state:
        !candidate || !item.paperTicketId
          ? 'Not applicable'
          : item.outcomeCount > 0
            ? 'Complete'
            : 'Not run',
      time: item.latestOutcomeAt,
      explanation:
        item.outcomeCount > 0
          ? `${item.outcomeCount} hypothetical outcome checkpoint${item.outcomeCount === 1 ? '' : 's'} linked.`
          : 'No hypothetical outcome checkpoints are linked.',
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Event journey</CardTitle>
        <CardDescription>
          Persisted stages are shown in order. Missing evidence is not guessed.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ol className="space-y-3">
          {stages.map((stage, index) => (
            <li key={stage.label} className="grid grid-cols-[2rem_minmax(0,1fr)] gap-3">
              <span
                aria-hidden="true"
                className="flex h-8 w-8 items-center justify-center rounded-full border font-semibold"
              >
                {index + 1}
              </span>
              <div className="min-w-0 rounded-lg border p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <strong>{stage.label}</strong>
                  <Badge variant="outline">{stage.state}</Badge>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{stage.explanation}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {stage.time ? formatTime(stage.time) : 'No timestamp retained'}
                </p>
              </div>
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  );
}
