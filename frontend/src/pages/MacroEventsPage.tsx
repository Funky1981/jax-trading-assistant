import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Eye, RefreshCw, XCircle } from 'lucide-react';
import { macroService } from '@/data/macro-service';
import type { MacroCandidate, MacroEvent } from '@/data/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { emitAnalyticsEvent } from '@/lib/analytics';

function formatTime(value?: string): string {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatNumber(value?: number | null, digits = 2): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  return value.toFixed(digits);
}

function symbolsFor(event?: MacroEvent): string {
  const symbols = event?.etfMappings?.map((mapping) => mapping.symbol) ?? [];
  return symbols.length > 0 ? symbols.join(', ') : 'Unmapped';
}

function statusVariant(status: string): 'default' | 'secondary' | 'outline' | 'warning' | 'destructive' {
  if (status === 'awaiting_human_approval') {
    return 'warning';
  }
  if (status === 'rejected' || status === 'blocked') {
    return 'destructive';
  }
  if (status === 'watch_only' || status === 'accepted') {
    return 'secondary';
  }
  return 'outline';
}

export function MacroEventsPage() {
  const queryClient = useQueryClient();
  const [selectedEventId, setSelectedEventId] = useState('');

  const eventsQuery = useQuery({
    queryKey: ['macro-events'],
    queryFn: () => macroService.listEvents({ limit: 100 }),
    staleTime: 30_000,
  });

  const events = useMemo(() => eventsQuery.data?.events ?? [], [eventsQuery.data?.events]);
  const selectedEvent = useMemo(
    () => events.find((event) => event.id === selectedEventId) ?? events[0],
    [events, selectedEventId]
  );

  useEffect(() => {
    if (!selectedEventId && events[0]?.id) {
      setSelectedEventId(events[0].id);
    }
  }, [events, selectedEventId]);

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'macro_events' });
  }, []);

  const detailQuery = useQuery({
    queryKey: ['macro-event-detail', selectedEvent?.id],
    queryFn: () => macroService.getEvent(selectedEvent?.id ?? ''),
    enabled: Boolean(selectedEvent?.id),
    staleTime: 30_000,
  });

  const updateCandidate = useMutation({
    mutationFn: ({ candidate, action }: { candidate: MacroCandidate; action: 'approve' | 'reject' }) =>
      action === 'approve'
        ? macroService.approveCandidate(candidate.id, 'Paper candidate reviewed in Macro Events')
        : macroService.rejectCandidate(candidate.id, 'Rejected from Macro Events review'),
    onSuccess: async (candidate) => {
      await queryClient.invalidateQueries({ queryKey: ['macro-events'] });
      await queryClient.invalidateQueries({ queryKey: ['macro-event-detail', candidate.macroEventId] });
    },
  });

  const detail = detailQuery.data;
  const candidateCount = events.reduce((sum, event) => sum + event.candidateCount, 0);
  const evidenceReadyCount = events.filter((event) => event.evidenceCount > 0).length;

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">Macro reaction engine</p>
        <div className="max-w-3xl space-y-2">
          <h1 className="text-2xl font-bold md:text-3xl">Macro Events</h1>
          <p className="text-base text-muted-foreground">
            Review macro news, ETF reaction evidence, and paper-only candidates before any separate trading workflow.
          </p>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-3" aria-label="Macro event summary">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Eye className="h-4 w-4" />
              Events
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{eventsQuery.data?.total ?? events.length}</p>
            <p className="mt-1 text-sm text-muted-foreground">Macro events retained for review.</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <CheckCircle2 className="h-4 w-4" />
              Evidence
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{evidenceReadyCount}</p>
            <p className="mt-1 text-sm text-muted-foreground">Events with evidence bundles attached.</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <AlertTriangle className="h-4 w-4" />
              Paper Candidates
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{candidateCount}</p>
            <p className="mt-1 text-sm text-muted-foreground">Stored as macro paper candidates only.</p>
          </CardContent>
        </Card>
      </section>

      {eventsQuery.isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 px-4 py-3 text-sm">
          Macro events are unavailable. Check the API service and database migrations.
        </div>
      )}

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(420px,0.95fr)]">
        <Card>
          <CardHeader className="gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle>Event Inbox</CardTitle>
              <CardDescription>High-signal macro events mapped to ETF themes.</CardDescription>
            </div>
            <Button type="button" variant="outline" onClick={() => eventsQuery.refetch()}>
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
          </CardHeader>
          <CardContent>
            {eventsQuery.isPending && <p className="text-sm text-muted-foreground">Loading macro events...</p>}
            {!eventsQuery.isPending && events.length === 0 && (
              <p className="rounded-md border border-border p-6 text-sm text-muted-foreground">
                No macro events are available yet.
              </p>
            )}
            {events.length > 0 && (
              <div className="w-full overflow-x-auto">
                <Table className="min-w-[900px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead>Time</TableHead>
                      <TableHead>Event</TableHead>
                      <TableHead>Actual / Expected</TableHead>
                      <TableHead>Direction</TableHead>
                      <TableHead>Mapped ETFs</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">Confidence</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {events.map((event) => (
                      <TableRow
                        key={event.id}
                        className={selectedEvent?.id === event.id ? 'cursor-pointer bg-muted/40' : 'cursor-pointer'}
                        onClick={() => setSelectedEventId(event.id)}
                      >
                        <TableCell>{formatTime(event.eventTimeUtc)}</TableCell>
                        <TableCell>
                          <div className="max-w-[320px]">
                            <p className="font-medium">{event.headline}</p>
                            <p className="truncate text-xs text-muted-foreground">{event.eventType} / {event.region}</p>
                          </div>
                        </TableCell>
                        <TableCell>
                          {formatNumber(event.actualValue)} / {formatNumber(event.expectedValue)} {event.unit}
                        </TableCell>
                        <TableCell>{event.direction}</TableCell>
                        <TableCell>{symbolsFor(event)}</TableCell>
                        <TableCell>
                          <Badge variant={statusVariant(event.status)}>{event.status}</Badge>
                        </TableCell>
                        <TableCell className="text-right">{formatNumber(event.confidence, 2)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Event Detail</CardTitle>
              <CardDescription>
                {selectedEvent ? selectedEvent.headline : 'Select an event to inspect macro reaction analysis.'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {detailQuery.isPending && selectedEvent && <p className="text-sm text-muted-foreground">Loading detail...</p>}
              {detailQuery.isError && (
                <p className="text-sm text-destructive">Event detail could not be loaded.</p>
              )}
              {detail && (
                <>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Metric label="Actual" value={`${formatNumber(detail.event.actualValue)} ${detail.event.unit ?? ''}`} />
                    <Metric label="Expected" value={`${formatNumber(detail.event.expectedValue)} ${detail.event.unit ?? ''}`} />
                    <Metric label="Surprise" value={formatNumber(detail.event.surpriseValue)} />
                    <Metric label="ETFs" value={symbolsFor(detail.event)} />
                  </div>
                  <div className="rounded-md border border-border p-3">
                    <p className="text-sm font-medium">Summary</p>
                    <p className="mt-1 text-sm text-muted-foreground">{detail.event.summary || 'No summary provided.'}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {detail && (
            <>
              <Card>
                <CardHeader>
                  <CardTitle>Reaction Snapshots</CardTitle>
                  <CardDescription>Chart reaction summary by symbol and timeframe.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  {detail.reactions.length === 0 && (
                    <p className="text-sm text-muted-foreground">No reaction snapshots are stored for this event.</p>
                  )}
                  {detail.reactions.map((reaction) => (
                    <div key={reaction.id} className="rounded-md border border-border p-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <p className="font-medium">{reaction.symbol} / {reaction.timeframe}</p>
                        <Badge variant={reaction.confirmsEvent ? 'success' : 'outline'}>
                          {reaction.confirmsEvent ? 'Confirmed' : 'Not confirmed'}
                        </Badge>
                      </div>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {reaction.direction}: {formatNumber(reaction.prePrice)} to {formatNumber(reaction.postPrice)}
                        {' '}({formatNumber(reaction.changePercent)}%)
                      </p>
                      <p className="mt-2 text-sm">{reaction.reason}</p>
                    </div>
                  ))}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Evidence Bundle</CardTitle>
                  <CardDescription>Missing evidence and walk-away reasons stay visible.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  {detail.evidenceBundles.length === 0 && (
                    <p className="text-sm text-muted-foreground">No evidence bundle is stored for this event.</p>
                  )}
                  {detail.evidenceBundles.map((bundle) => (
                    <div key={bundle.id} className="rounded-md border border-border p-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="font-medium">{bundle.symbol}</p>
                        <Badge variant="outline">{bundle.verdict}</Badge>
                        <Badge variant="secondary">{bundle.status}</Badge>
                      </div>
                      <p className="mt-2 text-sm">{bundle.summary}</p>
                      <ListBlock title="Missing evidence" items={bundle.missingEvidence} empty="No missing evidence recorded." />
                      <ListBlock title="Walk-away reasons" items={bundle.walkawayReasons} empty="No walk-away reasons recorded." />
                    </div>
                  ))}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Paper Candidate Review</CardTitle>
                  <CardDescription>Approval here changes macro candidate status only; it does not send an order.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  {detail.candidates.length === 0 && (
                    <p className="text-sm text-muted-foreground">No paper candidate is stored for this event.</p>
                  )}
                  {detail.candidates.map((candidate) => (
                    <div key={candidate.id} className="rounded-md border border-border p-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div>
                          <p className="font-medium">{candidate.symbol} {candidate.side}</p>
                          <p className="text-sm text-muted-foreground">{candidate.bias} / {candidate.entryType}</p>
                        </div>
                        <Badge variant={statusVariant(candidate.status)}>{candidate.status}</Badge>
                      </div>
                      <div className="mt-3 grid gap-2 text-sm sm:grid-cols-3">
                        <Metric label="Entry" value={formatNumber(candidate.entryReferencePrice)} />
                        <Metric label="Stop" value={formatNumber(candidate.stopReferencePrice)} />
                        <Metric label="Target" value={formatNumber(candidate.targetReferencePrice)} />
                      </div>
                      <p className="mt-3 text-sm">{candidate.createdReason}</p>
                      <ListBlock title="Why not to take it" items={candidate.walkawayReasons} empty="No walk-away reasons recorded." />
                      {candidate.humanApprovalRequired ? (
                        <div className="mt-3 flex flex-wrap gap-2">
                          <Button
                            type="button"
                            variant="outline"
                            onClick={() => updateCandidate.mutate({ candidate, action: 'approve' })}
                            disabled={updateCandidate.isPending}
                          >
                            <CheckCircle2 className="h-4 w-4" />
                            Mark watch only
                          </Button>
                          <Button
                            type="button"
                            variant="outline"
                            onClick={() => updateCandidate.mutate({ candidate, action: 'reject' })}
                            disabled={updateCandidate.isPending}
                          >
                            <XCircle className="h-4 w-4" />
                            Reject paper candidate
                          </Button>
                        </div>
                      ) : (
                        <p className="mt-3 text-sm text-muted-foreground">Human approval is no longer pending for this paper candidate.</p>
                      )}
                    </div>
                  ))}
                </CardContent>
              </Card>
            </>
          )}
        </div>
      </section>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-muted/30 px-3 py-2">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-semibold">{value}</p>
    </div>
  );
}

function ListBlock({ title, items, empty }: { title: string; items: string[]; empty: string }) {
  return (
    <div className="mt-3">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</p>
      {items.length === 0 ? (
        <p className="mt-1 text-sm text-muted-foreground">{empty}</p>
      ) : (
        <ul className="mt-1 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
          {items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      )}
    </div>
  );
}
