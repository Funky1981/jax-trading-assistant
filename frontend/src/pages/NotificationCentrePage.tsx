import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Bell, Clock3, Inbox, RefreshCw } from 'lucide-react';
import { eventsService } from '@/data/events-service';
import { toNotificationEntries } from '@/data/notification-adapter';
import type { NotificationInboxEntry } from '@/data/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

const READ_STORAGE_KEY = 'notification-centre-read-v1';

const categoryLabels: Record<NotificationInboxEntry['category'], string> = {
  opportunity: 'Opportunity',
  approval: 'Approval',
  sentiment_triggered: 'Sentiment alert',
  sentiment_invalidated: 'Sentiment invalidated',
  analysis: 'Analysis',
  settings: 'Settings',
  system: 'System',
};

function readStorageIds(): Set<string> {
  if (typeof window === 'undefined') {
    return new Set();
  }

  const value = window.localStorage.getItem(READ_STORAGE_KEY);
  if (!value) {
    return new Set();
  }

  try {
    const parsed = JSON.parse(value);
    if (!Array.isArray(parsed)) {
      return new Set();
    }

    return new Set(parsed.filter((entry): entry is string => typeof entry === 'string'));
  } catch {
    return new Set();
  }
}

function formatTime(value: string): string {
  const asDate = new Date(value);
  if (Number.isNaN(asDate.getTime())) {
    return value;
  }

  return asDate.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function NotificationCentrePage() {
  const [readIds, setReadIds] = useState<Set<string>>(() => readStorageIds());

  const notificationsQuery = useQuery({
    queryKey: ['notifications', 'inbox'],
    queryFn: () => eventsService.list({ limit: 100 }),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const entries = useMemo(
    () => toNotificationEntries(notificationsQuery.data?.events ?? []),
    [notificationsQuery.data?.events]
  );

  const unreadCount = useMemo(
    () => entries.filter((entry) => !readIds.has(entry.id)).length,
    [entries, readIds]
  );

  useEffect(() => {
    window.localStorage.setItem(READ_STORAGE_KEY, JSON.stringify(Array.from(readIds)));
  }, [readIds]);

  const toggleRead = (entryId: string) => {
    setReadIds((current) => {
      const next = new Set(current);
      if (next.has(entryId)) {
        next.delete(entryId);
      } else {
        next.add(entryId);
      }
      return next;
    });
  };

  const markAllRead = () => {
    setReadIds(new Set(entries.map((entry) => entry.id)));
  };

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">Notifications</p>
        <div className="max-w-3xl space-y-2">
          <h1 className="text-2xl font-bold md:text-3xl">Notification Centre</h1>
          <p className="text-base text-muted-foreground">
            Durable in-app inbox for opportunities, approvals, and sentiment-related alerts after transient toasts disappear.
          </p>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-3" aria-label="Notification inbox summary">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Bell className="h-4 w-4" />
              Unread
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{unreadCount}</p>
            <p className="mt-1 text-sm text-muted-foreground">Items still requiring review.</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Inbox className="h-4 w-4" />
              Total
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{entries.length}</p>
            <p className="mt-1 text-sm text-muted-foreground">Durable events retained in the in-app inbox.</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Clock3 className="h-4 w-4" />
              Refresh
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Button type="button" variant="outline" onClick={() => notificationsQuery.refetch()}>
              <RefreshCw className="h-4 w-4" />
              Refresh inbox
            </Button>
          </CardContent>
        </Card>
      </section>

      <section className="space-y-3" aria-label="Notification inbox">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-xl font-semibold">Inbox</h2>
            <p className="text-sm text-muted-foreground">Select a notification to continue at the relevant workflow.</p>
          </div>
          <Button type="button" variant="outline" onClick={markAllRead} disabled={entries.length === 0 || unreadCount === 0}>
            Mark all read
          </Button>
        </div>

        {notificationsQuery.isPending && (
          <p className="rounded-md border border-border p-6 text-muted-foreground">Loading notifications...</p>
        )}

        {notificationsQuery.isError && (
          <div className="rounded-md border border-destructive bg-destructive/10 px-4 py-3 text-sm text-foreground">
            Notification inbox is temporarily unavailable. Refresh to retry.
          </div>
        )}

        {!notificationsQuery.isPending && entries.length === 0 && (
          <Card>
            <CardContent className="py-10 text-center text-muted-foreground">
              No notifications yet. New opportunity and approval events will appear here.
            </CardContent>
          </Card>
        )}

        {!notificationsQuery.isPending && entries.map((entry) => {
          const unread = !readIds.has(entry.id);

          return (
            <Card key={entry.id} className={unread ? 'border-primary/40' : undefined}>
              <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <CardTitle className="text-lg">{entry.title}</CardTitle>
                    <Badge variant={unread ? 'default' : 'secondary'}>{unread ? 'Unread' : 'Read'}</Badge>
                    <Badge variant="outline">{categoryLabels[entry.category]}</Badge>
                    {entry.stale && <Badge variant="outline">Stale</Badge>}
                  </div>
                  <CardDescription>{entry.body}</CardDescription>
                </div>
                <div className="text-sm text-muted-foreground">{formatTime(entry.createdAt)}</div>
              </CardHeader>

              <CardContent className="space-y-4">
                <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                  <span className="font-semibold text-foreground">Event type:</span>
                  <span>{entry.eventType}</span>
                  {entry.severity && (
                    <>
                      <span className="font-semibold text-foreground">Severity:</span>
                      <span>{entry.severity}</span>
                    </>
                  )}
                  {entry.primarySymbol && (
                    <>
                      <span className="font-semibold text-foreground">Symbol:</span>
                      <span>{entry.primarySymbol}</span>
                    </>
                  )}
                </div>

                <div className="flex flex-wrap gap-2">
                  {entry.channels.map((channel) => (
                    <Badge key={`${entry.id}-${channel}`} variant="secondary">
                      {channel}
                    </Badge>
                  ))}
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button asChild>
                    <Link to={entry.destinationPath}>Open destination</Link>
                  </Button>
                  <Button type="button" variant="outline" onClick={() => toggleRead(entry.id)}>
                    {unread ? 'Mark read' : 'Mark unread'}
                  </Button>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </section>
    </div>
  );
}
