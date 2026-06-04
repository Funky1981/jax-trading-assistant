import type { EventSummary, NotificationCategory, NotificationInboxEntry } from '@/data/types';

const STALE_WINDOW_MS = 24 * 60 * 60 * 1000;

function normalizeTime(event: EventSummary): string {
  return event.createdAt ?? event.eventTime ?? new Date(0).toISOString();
}

function normalizeKind(kind?: string): string {
  return (kind ?? '').trim().toLowerCase();
}

function parseChannels(attributes?: Record<string, unknown>): string[] {
  const explicitChannels = attributes?.channels;
  if (Array.isArray(explicitChannels)) {
    const labels = explicitChannels
      .map((channel) => (typeof channel === 'string' ? channel.trim() : ''))
      .filter((channel) => channel.length > 0)
      .map(toChannelLabel);

    if (labels.length > 0) {
      return labels;
    }
  }

  const explicitChannel = attributes?.channel;
  if (typeof explicitChannel === 'string' && explicitChannel.trim().length > 0) {
    return [toChannelLabel(explicitChannel.trim())];
  }

  return ['In-app inbox'];
}

function toChannelLabel(value: string): string {
  const lower = value.toLowerCase();
  if (lower === 'in_app' || lower === 'in-app' || lower === 'inapp') {
    return 'In-app inbox';
  }

  return value
    .split(/[_\-\s]+/)
    .filter((part) => part.length > 0)
    .map((part) => part[0].toUpperCase() + part.slice(1).toLowerCase())
    .join(' ');
}

function categoryForEvent(event: EventSummary): NotificationCategory {
  const kind = normalizeKind(event.kind);
  const tags = Array.isArray(event.attributes?.tags)
    ? event.attributes.tags.filter((tag): tag is string => typeof tag === 'string').map((tag) => tag.toLowerCase())
    : [];

  if (kind.includes('sentiment_invalidated') || tags.includes('sentiment_invalidated')) {
    return 'sentiment_invalidated';
  }

  if (kind.includes('sentiment') || tags.includes('sentiment_triggered')) {
    return 'sentiment_triggered';
  }

  if (kind.includes('approval')) {
    return 'approval';
  }

  if (kind.includes('opportunity') || kind.includes('candidate') || kind.includes('signal') || kind.includes('scanner')) {
    return 'opportunity';
  }

  if (kind.includes('analysis') || kind.includes('backtest') || kind.includes('research')) {
    return 'analysis';
  }

  if (kind.includes('settings') || kind.includes('preference') || kind.includes('notification_rule')) {
    return 'settings';
  }

  return 'system';
}

function destinationForEvent(event: EventSummary): string {
  const routeFromAttributes = event.attributes?.route;
  if (typeof routeFromAttributes === 'string' && routeFromAttributes.startsWith('/')) {
    return routeFromAttributes;
  }

  const entityRoute = event.attributes?.entityRoute;
  if (typeof entityRoute === 'string' && entityRoute.startsWith('/')) {
    return entityRoute;
  }

  const category = categoryForEvent(event);
  switch (category) {
    case 'approval':
      return '/etf/approvals';
    case 'opportunity':
    case 'sentiment_triggered':
    case 'sentiment_invalidated':
      return '/ai-trading';
    case 'analysis':
      return '/analysis';
    case 'settings':
      return '/settings';
    default:
      return '/system';
  }
}

export function eventToNotificationEntry(event: EventSummary, nowMs = Date.now()): NotificationInboxEntry {
  const createdAt = normalizeTime(event);
  const createdAtMs = new Date(createdAt).getTime();
  const stale = Number.isFinite(createdAtMs) ? nowMs - createdAtMs > STALE_WINDOW_MS : false;

  return {
    id: event.id,
    category: categoryForEvent(event),
    eventType: event.kind,
    title: event.title,
    body: event.summary?.trim() || 'Open this notification for details and next steps.',
    destinationPath: destinationForEvent(event),
    createdAt,
    stale,
    channels: parseChannels(event.attributes),
    severity: event.severity,
    primarySymbol: event.primarySymbol,
    sentimentTriggerType:
      typeof event.attributes?.sentimentTriggerType === 'string' ? event.attributes.sentimentTriggerType : undefined,
    entityType: typeof event.attributes?.entityType === 'string' ? event.attributes.entityType : undefined,
    entityId: typeof event.attributes?.entityId === 'string' ? event.attributes.entityId : undefined,
  };
}

export function toNotificationEntries(events: EventSummary[], nowMs = Date.now()): NotificationInboxEntry[] {
  return events
    .map((event) => eventToNotificationEntry(event, nowMs))
    .sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime());
}
