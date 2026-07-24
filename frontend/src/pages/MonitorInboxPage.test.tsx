import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import { MonitorInboxPage } from './MonitorInboxPage';
import { aiService } from '@/data/ai-service';
import { HttpError } from '@/data/http-client';

vi.mock('@/data/ai-service', () => ({
  aiService: {
    getWorldMonitorInbox: vi.fn(),
  },
}));

vi.mock('@/hooks/useOperatorEvidenceOverview', () => ({
  useOperatorEvidenceOverview: () => ({
    data: {
      runtimeMode: 'paper',
      allowLiveTrading: false,
      executionEnabled: false,
      executionWorkerEnabled: false,
      brokerExecutionAllowed: false,
      maximumLeverage: 1,
    },
    isError: false,
  }),
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
    </MemoryRouter>,
  );
}

describe('MonitorInboxPage', () => {
  beforeEach(() => {
    vi.mocked(aiService.getWorldMonitorInbox).mockClear();
    vi.mocked(aiService.getWorldMonitorInbox).mockResolvedValue({
      total: 2,
      counts: {
        genuine: 1,
        syntheticTests: 1,
        candidatesCreated: 1,
        rejected: 1,
        duplicates: 0,
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
          collectedAt: '2026-06-12T10:30:30Z',
          rawEventId: 'raw-1',
          isSynthetic: false,
          discoveryMethod: 'rss_poll',
          analysisIdentity: 'deterministic-v1',
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
          candidateSymbol: 'QQQ',
          candidateStatus: 'awaiting_approval',
          candidateCreatedAt: '2026-06-12T10:32:00Z',
          outcomeCount: 0,
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
          provenanceAvailable: true,
          isSynthetic: true,
          syntheticReason: 'labelled fixture',
          possibleAffectedEtfs: [],
          assetThemes: [],
          severity: 'high',
          sourceTier: 'tier2',
          confidence: 0.7,
          confidenceReasons: [],
          mappingReason: 'Rejected before mapping.',
          outcomeCount: 0,
          rawPayload: { bad: true },
        },
      ],
    });
  });

  it('renders the beginner Evidence Inbox and opens human-readable detail', async () => {
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByRole('heading', { name: 'Evidence Inbox' })).toBeInTheDocument();
    expect((await screen.findAllByText('Accepted monitor item')).length).toBeGreaterThan(0);
    expect(screen.getByText('Rejected monitor item')).toBeInTheDocument();
    expect(screen.getAllByText('GENUINE').length).toBeGreaterThan(0);
    expect(screen.getAllByText('SYNTHETIC TEST').length).toBeGreaterThan(0);

    await user.click(screen.getByRole('button', { name: /Accepted monitor item/i }));

    expect(screen.getByText('Mapped to QQQ from rates theme.')).toBeInTheDocument();
    expect(screen.getByText('No AI used')).toBeInTheDocument();
    expect(screen.getByText('DETERMINISTIC ANALYSIS')).toBeInTheDocument();
    expect(screen.getByText('rss_poll')).toBeInTheDocument();
    expect(screen.queryByText('Unknown assets')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Open Candidate Review/i })).toHaveAttribute(
      'href',
      '/candidates/candidate-1/evidence',
    );
    expect(screen.getByText('Audit details')).toBeInTheDocument();
    expect(screen.getByText(/monitor_score/)).not.toBeVisible();
    expect(
      screen.queryByRole('button', { name: /approve|execute|trade/i }),
    ).not.toBeInTheDocument();
  });

  it('filters without mutating records and shows rejection and unknown-assets states', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findAllByText('Accepted monitor item');
    await user.click(screen.getByRole('button', { name: 'Rejected' }));
    await user.click(screen.getByRole('button', { name: /Rejected monitor item/i }));

    expect((await screen.findAllByText('source_urls are required')).length).toBeGreaterThan(0);
    expect(screen.getByText('Unknown assets')).toBeInTheDocument();
    expect(aiService.getWorldMonitorInbox).toHaveBeenLastCalledWith({ limit: 100 });
  });

  it('distinguishes a stale or missing Monitor API route from an empty inbox', async () => {
    vi.mocked(aiService.getWorldMonitorInbox).mockRejectedValue(
      new HttpError('Request failed: 404', 404, '404 page not found'),
    );

    renderPage();

    expect(await screen.findByText(/Evidence Inbox unavailable/i)).toBeInTheDocument();
    expect(screen.getByText(/Your data has not been changed/i)).toBeInTheDocument();
  });

  it('shows an entitlement message when the Monitor inbox is protected by auth', async () => {
    vi.mocked(aiService.getWorldMonitorInbox).mockRejectedValue(
      new HttpError('Request failed: 401', 401, 'missing authorization token'),
    );

    renderPage();

    expect(await screen.findByText(/Sign in to view evidence/i)).toBeInTheDocument();
    expect(screen.getByText(/protected evidence/i)).toBeInTheDocument();
  });

  it('has no detectable accessibility violations in the open evidence detail', async () => {
    const user = userEvent.setup();
    const { container } = renderPage();
    await user.click(
      await screen.findByRole('button', {
        name: /Accepted monitor item/i,
      }),
    );

    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });
});
