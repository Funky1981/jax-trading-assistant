import { describe, expect, it } from 'vitest';
import { eventToNotificationEntry, toNotificationEntries } from './notification-adapter';
import type { EventSummary } from './types';

describe('notification adapter', () => {
  it('maps approval events to approval destination and channel labels', () => {
    const event: EventSummary = {
      id: 'event-approval-1',
      kind: 'approval_required',
      title: 'Approval needed for QQQ candidate',
      summary: 'ETF candidate requires operator approval.',
      eventTime: '2026-05-22T09:40:00Z',
      attributes: {
        channels: ['in_app', 'email'],
      },
    };

    const mapped = eventToNotificationEntry(event, new Date('2026-05-22T10:00:00Z').getTime());

    expect(mapped).toMatchObject({
      id: 'event-approval-1',
      category: 'approval',
      destinationPath: '/etf/approvals',
      channels: ['In-app inbox', 'Email'],
      stale: false,
    });
  });

  it('uses explicit attribute route and sentiment-invalidated category', () => {
    const event: EventSummary = {
      id: 'event-sentiment-1',
      kind: 'sentiment_invalidated',
      title: 'Sentiment changed for SPY setup',
      summary: 'Existing trade idea no longer meets sentiment criteria.',
      createdAt: '2026-05-20T08:00:00Z',
      attributes: {
        route: '/ai-trading',
      },
    };

    const mapped = eventToNotificationEntry(event, new Date('2026-05-22T10:00:00Z').getTime());

    expect(mapped).toMatchObject({
      category: 'sentiment_invalidated',
      destinationPath: '/ai-trading',
      stale: true,
    });
  });

  it('preserves sentiment alert metadata for inbox labels and routing', () => {
    const event: EventSummary = {
      id: 'event-sentiment-boost',
      kind: 'sentiment_conviction_boost',
      title: 'Sentiment boosted SPY setup',
      summary: 'Trusted news moved above the configured threshold.',
      createdAt: '2026-05-22T09:00:00Z',
      attributes: {
        sentimentTriggerType: 'conviction_boost',
        entityType: 'opportunity',
        entityId: 'candidate-1',
        route: '/ai-trading?symbol=SPY',
        channels: ['in_app', 'desktop_web'],
      },
      primarySymbol: 'SPY',
    };

    expect(eventToNotificationEntry(event)).toMatchObject({
      category: 'sentiment_triggered',
      destinationPath: '/ai-trading?symbol=SPY',
      sentimentTriggerType: 'conviction_boost',
      entityType: 'opportunity',
      entityId: 'candidate-1',
      channels: ['In-app inbox', 'Desktop Web'],
      primarySymbol: 'SPY',
    });
  });

  it('returns deterministic newest-first ordering', () => {
    const events: EventSummary[] = [
      {
        id: 'event-old',
        kind: 'opportunity_detected',
        title: 'Old event',
        createdAt: '2026-05-22T08:00:00Z',
      },
      {
        id: 'event-new',
        kind: 'analysis_completed',
        title: 'New event',
        createdAt: '2026-05-22T09:00:00Z',
      },
    ];

    expect(toNotificationEntries(events).map((entry) => entry.id)).toEqual(['event-new', 'event-old']);
  });
});
