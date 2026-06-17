import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MonitorInboxPage } from './MonitorInboxPage';
import { aiService } from '@/data/ai-service';
import { HttpError } from '@/data/http-client';
import { emitAnalyticsEvent } from '@/lib/analytics';

vi.mock('@/data/ai-service', () => ({
  aiService: {
    getWorldMonitorInbox: vi.fn(),
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
        <MonitorInboxPage />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('MonitorInboxPage', () => {
  beforeEach(() => {
    vi.mocked(aiService.getWorldMonitorInbox).mockResolvedValue({
      total: 2,
      counts: {
        total: 2,
        pending: 0,
        candidatesCreated: 1,
        rejected: 1,
        ignored: 0,
      },
      checkedAt: '2026-06-12T10:45:00Z',
      items: [
        {
          id: 'inbox-accepted',
          source: 'world-monitor',
          sourceEventId: 'accepted-1',
          worldMonitorEventId: 'accepted-1',
          status: 'candidate_created',
          eventType: 'macro_rates',
          headline: 'Accepted monitor item',
          summary: 'Rates news mapped to QQQ.',
          sourceUrls: ['https://example.com/accepted'],
          sourceCount: 1,
          eventTime: '2026-06-12T10:30:00Z',
          receivedAt: '2026-06-12T10:31:00Z',
          region: 'US',
          possibleAffectedEtfs: ['QQQ'],
          assetThemes: ['rates'],
          severity: 'high',
          sourceTier: 'tier2',
          confidence: 0.82,
          confidenceReasons: ['trusted source'],
          mappingReason: 'Mapped to QQQ from rates theme.',
          normalizedEventId: 'event-1',
          candidateId: 'candidate-1',
          rawPayload: { fixture: true, monitor_score: 0.82 },
        },
        {
          id: 'inbox-rejected',
          source: 'world-monitor',
          sourceEventId: 'rejected-1',
          worldMonitorEventId: 'rejected-1',
          status: 'rejected',
          rejectionReason: 'source_urls are required',
          eventType: 'macro_rates',
          headline: 'Rejected monitor item',
          sourceUrls: [],
          sourceCount: 0,
          eventTime: '2026-06-12T10:20:00Z',
          receivedAt: '2026-06-12T10:21:00Z',
          possibleAffectedEtfs: [],
          assetThemes: [],
          severity: 'high',
          sourceTier: 'tier2',
          confidence: 0.7,
          confidenceReasons: [],
          mappingReason: 'Rejected before mapping.',
          rawPayload: { bad: true },
        },
      ],
    });
    vi.mocked(emitAnalyticsEvent).mockClear();
  });

  it('renders accepted and rejected monitor payloads with raw audit detail', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { name: 'Monitor Inbox' })).toBeInTheDocument();
    expect((await screen.findAllByText('Accepted monitor item')).length).toBeGreaterThan(0);
    expect(screen.getByText('Rejected monitor item')).toBeInTheDocument();
    expect(screen.getAllByText('candidate_created').length).toBeGreaterThan(0);
    expect(screen.getAllByText('rejected').length).toBeGreaterThan(0);
    expect(screen.getByText('Mapped to QQQ from rates theme.')).toBeInTheDocument();
    expect(screen.getByText(/monitor_score/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Open candidate evidence/i })).toHaveAttribute(
      'href',
      '/candidates/candidate-1/evidence'
    );
  });

  it('filters by rejected status and shows the rejection reason detail', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findAllByText('Accepted monitor item');
    await user.selectOptions(screen.getByLabelText('Monitor inbox status'), 'rejected');

    expect(aiService.getWorldMonitorInbox).toHaveBeenLastCalledWith({ status: 'rejected', limit: 100 });
    expect(await screen.findByText('source_urls are required')).toBeInTheDocument();
  });

  it('distinguishes a stale or missing Monitor API route from an empty inbox', async () => {
    vi.mocked(aiService.getWorldMonitorInbox).mockRejectedValue(
      new HttpError('Request failed: 404', 404, '404 page not found')
    );

    renderPage();

    expect(await screen.findByText(/Monitor inbox API route was not found/i)).toBeInTheDocument();
    expect(screen.getByText(/rebuild and restart jax-trader/i)).toBeInTheDocument();
    expect(screen.queryByText(/No Monitor payloads match this filter yet/i)).not.toBeInTheDocument();
  });

  it('shows an entitlement message when the Monitor inbox is protected by auth', async () => {
    vi.mocked(aiService.getWorldMonitorInbox).mockRejectedValue(
      new HttpError('Request failed: 401', 401, 'missing authorization token')
    );

    renderPage();

    expect(await screen.findByText(/Sign in to view Monitor payloads/i)).toBeInTheDocument();
    expect(screen.getByText(/missing authorization token/i)).toBeInTheDocument();
  });
});
