import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ExternalLink, RefreshCw } from 'lucide-react';
import { aiService, type WorldMonitorInboxItem } from '@/data/ai-service';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { emitAnalyticsEvent } from '@/lib/analytics';

const STATUS_OPTIONS = [
  { label: 'All', value: 'all' },
  { label: 'Accepted', value: 'new' },
  { label: 'Promoted', value: 'candidate_created' },
  { label: 'Rejected', value: 'rejected' },
  { label: 'Ignored', value: 'ignored' },
];

function formatTime(value?: string): string {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function pct(value?: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '-';
  return `${Math.round(value * 100)}%`;
}

function statusVariant(status: string): 'default' | 'secondary' | 'outline' | 'warning' | 'destructive' | 'success' {
  switch (status) {
    case 'candidate_created':
      return 'success';
    case 'new':
      return 'warning';
    case 'rejected':
      return 'destructive';
    case 'ignored':
      return 'secondary';
    default:
      return 'outline';
  }
}

function joinOrDash(values?: string[]): string {
  return values && values.length > 0 ? values.join(', ') : '-';
}

function rawPayloadText(item?: WorldMonitorInboxItem): string {
  if (!item) return '{}';
  return JSON.stringify(item.rawPayload ?? {}, null, 2);
}

function SummaryTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-border bg-card/60 p-3">
      <p className="text-xs font-semibold uppercase text-muted-foreground">{label}</p>
      <p className="mt-1 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function DetailList({ title, items, empty }: { title: string; items: string[]; empty: string }) {
  return (
    <div className="rounded-md border border-border p-3">
      <p className="text-sm font-semibold">{title}</p>
      {items.length === 0 ? (
        <p className="mt-1 text-sm text-muted-foreground">{empty}</p>
      ) : (
        <ul className="mt-2 space-y-1 text-sm text-muted-foreground">
          {items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function MonitorInboxPage() {
  const [status, setStatus] = useState('all');
  const [selectedId, setSelectedId] = useState('');

  const inboxQuery = useQuery({
    queryKey: ['world-monitor-inbox', status],
    queryFn: () => aiService.getWorldMonitorInbox({ status, limit: 100 }),
    refetchInterval: 30_000,
    staleTime: 30_000,
  });

  const items = useMemo(() => inboxQuery.data?.items ?? [], [inboxQuery.data?.items]);
  const selected = useMemo(
    () => items.find((item) => item.id === selectedId) ?? items[0],
    [items, selectedId]
  );

  useEffect(() => {
    if (!selectedId && items[0]?.id) {
      setSelectedId(items[0].id);
    }
  }, [items, selectedId]);

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'monitor_inbox' });
  }, []);

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">World News Monitor</p>
        <div className="max-w-3xl space-y-2">
          <h1 className="text-2xl font-bold md:text-3xl">Monitor Inbox</h1>
          <p className="text-base text-muted-foreground">
            Review every news payload Jax received from the Monitor, including rejected and ignored items.
          </p>
        </div>
      </section>

      <section className="grid gap-3 md:grid-cols-5" aria-label="Monitor inbox summary">
        <SummaryTile label="Total" value={inboxQuery.data?.counts.total ?? 0} />
        <SummaryTile label="Accepted" value={inboxQuery.data?.counts.pending ?? 0} />
        <SummaryTile label="Promoted" value={inboxQuery.data?.counts.candidatesCreated ?? 0} />
        <SummaryTile label="Rejected" value={inboxQuery.data?.counts.rejected ?? 0} />
        <SummaryTile label="Ignored" value={inboxQuery.data?.counts.ignored ?? 0} />
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(420px,0.9fr)]">
        <Card>
          <CardHeader className="gap-3 md:flex-row md:items-center md:justify-between">
            <div>
              <CardTitle>Received Payloads</CardTitle>
              <CardDescription>Filter by Jax decision and select a row to inspect the full payload.</CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <select
                aria-label="Monitor inbox status"
                className="min-h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"
                value={status}
                onChange={(event) => {
                  setStatus(event.target.value);
                  setSelectedId('');
                }}
              >
                {STATUS_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              <Button type="button" variant="outline" onClick={() => inboxQuery.refetch()}>
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {inboxQuery.isPending && <p className="text-sm text-muted-foreground">Loading Monitor inbox...</p>}
            {inboxQuery.isError && (
              <p className="rounded-md border border-destructive bg-destructive/10 p-4 text-sm text-destructive">
                Monitor inbox could not be loaded. Check the API service and database migrations.
              </p>
            )}
            {!inboxQuery.isPending && !inboxQuery.isError && items.length === 0 && (
              <p className="rounded-md border border-border p-6 text-sm text-muted-foreground">
                No Monitor payloads match this filter yet.
              </p>
            )}
            {items.length > 0 && (
              <div className="w-full overflow-x-auto">
                <Table className="min-w-[980px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead>Received</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Headline</TableHead>
                      <TableHead>ETFs</TableHead>
                      <TableHead>Confidence</TableHead>
                      <TableHead>Decision reason</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow
                        key={item.id}
                        className={selected?.id === item.id ? 'cursor-pointer bg-muted/40' : 'cursor-pointer'}
                        onClick={() => setSelectedId(item.id)}
                      >
                        <TableCell>{formatTime(item.receivedAt)}</TableCell>
                        <TableCell>
                          <Badge variant={statusVariant(item.status)}>{item.status}</Badge>
                        </TableCell>
                        <TableCell>
                          <div className="max-w-[360px]">
                            <p className="font-medium">{item.headline}</p>
                            <p className="truncate text-xs text-muted-foreground">{item.sourceEventId}</p>
                          </div>
                        </TableCell>
                        <TableCell>{joinOrDash(item.possibleAffectedEtfs)}</TableCell>
                        <TableCell>{pct(item.confidence)}</TableCell>
                        <TableCell>
                          <p className="max-w-[280px] truncate text-sm text-muted-foreground">
                            {item.rejectionReason || item.mappingReason || '-'}
                          </p>
                        </TableCell>
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
              <CardTitle>Payload Detail</CardTitle>
              <CardDescription>
                {selected ? selected.headline : 'Select a Monitor payload to inspect what Jax received.'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {!selected && <p className="text-sm text-muted-foreground">No payload selected.</p>}
              {selected && (
                <>
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={statusVariant(selected.status)}>{selected.status}</Badge>
                    <Badge variant="outline">{selected.eventType}</Badge>
                    <Badge variant="secondary">{selected.severity}</Badge>
                    <Badge variant="secondary">{selected.sourceTier}</Badge>
                  </div>

                  {selected.rejectionReason ? (
                    <div className="rounded-md border border-destructive bg-destructive/10 p-3 text-sm">
                      <p className="font-semibold text-destructive">Rejected by Jax</p>
                      <p className="mt-1 text-destructive">{selected.rejectionReason}</p>
                    </div>
                  ) : null}

                  <div className="grid gap-3 sm:grid-cols-2">
                    <Metric label="Source event" value={selected.sourceEventId} />
                    <Metric label="Received" value={formatTime(selected.receivedAt)} />
                    <Metric label="Event time" value={formatTime(selected.eventTime)} />
                    <Metric label="Mapped ETFs" value={joinOrDash(selected.possibleAffectedEtfs)} />
                    <Metric label="Candidate" value={selected.candidateId || '-'} />
                    <Metric label="Normalized event" value={selected.normalizedEventId || '-'} />
                  </div>

                  {selected.summary ? (
                    <div className="rounded-md border border-border p-3">
                      <p className="text-sm font-semibold">Summary</p>
                      <p className="mt-1 text-sm text-muted-foreground">{selected.summary}</p>
                    </div>
                  ) : null}

                  <DetailList title="Confidence reasons" items={selected.confidenceReasons} empty="No confidence reasons were supplied." />
                  <DetailList title="Asset themes" items={selected.assetThemes} empty="No themes were supplied." />

                  <div className="rounded-md border border-border p-3">
                    <p className="text-sm font-semibold">Source URLs</p>
                    {selected.sourceUrls.length === 0 ? (
                      <p className="mt-1 text-sm text-muted-foreground">No source URLs were supplied.</p>
                    ) : (
                      <div className="mt-2 space-y-1">
                        {selected.sourceUrls.map((url) => (
                          <a
                            key={url}
                            className="flex items-center gap-1 break-all text-sm text-primary underline"
                            href={url}
                            rel="noreferrer"
                            target="_blank"
                          >
                            {url}
                            <ExternalLink className="h-3 w-3 shrink-0" />
                          </a>
                        ))}
                      </div>
                    )}
                  </div>

                  {selected.candidateId ? (
                    <div className="flex flex-wrap gap-2">
                      <Button asChild variant="outline">
                        <Link to={`/candidates/${selected.candidateId}/evidence`}>Open candidate evidence</Link>
                      </Button>
                      <Button asChild variant="outline">
                        <Link to="/ai-trading">Open AI Trading</Link>
                      </Button>
                    </div>
                  ) : null}
                </>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Raw Payload</CardTitle>
              <CardDescription>Original Monitor fields retained by Jax for audit.</CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="max-h-[460px] overflow-auto rounded-md border border-border bg-muted/30 p-3 text-xs text-foreground">
                {rawPayloadText(selected)}
              </pre>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-muted/30 px-3 py-2">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-1 break-all text-sm font-semibold">{value}</p>
    </div>
  );
}
