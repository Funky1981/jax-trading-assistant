import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NotificationCentrePage } from './NotificationCentrePage';
import { eventsService } from '@/data/events-service';
import { emitAnalyticsEvent } from '@/lib/analytics';

vi.mock('@/data/events-service', () => ({
  eventsService: {
    list: vi.fn(),
  },
}));

vi.mock('@/lib/analytics', () => ({
  emitAnalyticsEvent: vi.fn(),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <NotificationCentrePage />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('NotificationCentrePage', () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.mocked(emitAnalyticsEvent).mockClear();
  });

  it('renders durable inbox entries with route-aware destinations', async () => {
    vi.mocked(eventsService.list).mockResolvedValue({
      events: [
        {
          id: 'event-approval',
          kind: 'approval_required',
          title: 'Approval required for QQQ',
          summary: 'Open the approval queue to continue.',
          eventTime: '2026-05-22T10:00:00Z',
          attributes: { channels: ['in_app'] },
        },
        {
          id: 'event-opportunity',
          kind: 'sentiment_invalidated',
          title: 'Sentiment invalidated previous setup',
          summary: 'Review AI Trading for updated evidence and next steps.',
          eventTime: '2026-05-22T09:30:00Z',
          attributes: { route: '/ai-trading', channels: ['in_app', 'email'] },
        },
      ],
      total: 2,
      limit: 100,
      offset: 0,
    });

    renderPage();

    expect(await screen.findByRole('heading', { name: 'Notification Centre' })).toBeInTheDocument();
    expect(await screen.findByText('Approval required for QQQ')).toBeInTheDocument();
    const destinationLinks = screen.getAllByRole('link', { name: 'Open destination' }).map((link) => link.getAttribute('href'));
    expect(destinationLinks).toContain('/etf/approvals');
    expect(destinationLinks).toContain('/ai-trading');

    await userEvent.setup().click(screen.getAllByRole('link', { name: 'Open destination' })[1]);
    const sentimentCall = vi.mocked(emitAnalyticsEvent).mock.calls.find(
      (call) => call[0] === 'sentiment_alert_opened' && call[1]?.destination_path === '/ai-trading'
    );
    expect(sentimentCall).toBeTruthy();

    expect(screen.getAllByText('In-app inbox').length).toBeGreaterThan(0);
    expect(screen.getByText('Email')).toBeInTheDocument();
  });

  it('supports read and unread state toggles with persisted ids', async () => {
    const user = userEvent.setup();
    vi.mocked(eventsService.list).mockResolvedValue({
      events: [
        {
          id: 'event-opportunity',
          kind: 'opportunity_detected',
          title: 'Scanner update',
          eventTime: '2026-05-22T09:30:00Z',
        },
      ],
      total: 1,
      limit: 100,
      offset: 0,
    });

    renderPage();

    expect(await screen.findByText('Scanner update')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Mark read' }));

    expect(screen.getByText('Read')).toBeInTheDocument();
    expect(window.localStorage.getItem('notification-centre-read-v1')).toContain('event-opportunity');
  });

  it('renders empty state when no notifications are returned', async () => {
    vi.mocked(eventsService.list).mockResolvedValue({
      events: [],
      total: 0,
      limit: 100,
      offset: 0,
    });

    renderPage();

    expect(await screen.findByText(/No notifications yet/i)).toBeInTheDocument();
  });
});
