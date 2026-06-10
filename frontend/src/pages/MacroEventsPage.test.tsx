import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MacroEventsPage } from './MacroEventsPage';
import { macroService } from '@/data/macro-service';
import { emitAnalyticsEvent } from '@/lib/analytics';
import type { MacroEventDetail, MacroEventListResponse } from '@/data/types';

vi.mock('@/data/macro-service', () => ({
  macroService: {
    listEvents: vi.fn(),
    getEvent: vi.fn(),
    approveCandidate: vi.fn(),
    rejectCandidate: vi.fn(),
  },
}));

vi.mock('@/lib/analytics', () => ({
  emitAnalyticsEvent: vi.fn(),
}));

const eventList: MacroEventListResponse = {
  events: [
    {
      id: 'macro-event-1',
      source: 'test',
      sourceEventId: 'cpi-1',
      eventType: 'cpi',
      region: 'US',
      eventTimeUtc: '2026-06-10T12:30:00Z',
      headline: 'CPI cooler than expected',
      summary: 'Inflation cooled versus forecast.',
      actualValue: 3.1,
      expectedValue: 3.3,
      previousValue: 3.4,
      unit: 'pct',
      surpriseValue: -0.2,
      surprisePercent: -6.06,
      direction: 'inflation_cool',
      confidence: 0.88,
      status: 'accepted',
      createdAt: '2026-06-10T12:31:00Z',
      updatedAt: '2026-06-10T12:31:00Z',
      etfMappings: [
        {
          id: 'mapping-1',
          symbol: 'TLT',
          theme: 'rates',
          mappingReason: 'Cool CPI supports duration',
          confidence: 0.8,
          createdAt: '2026-06-10T12:31:00Z',
        },
      ],
      candidateCount: 1,
      evidenceCount: 1,
    },
  ],
  total: 1,
  limit: 100,
  offset: 0,
};

const eventDetail: MacroEventDetail = {
  event: eventList.events[0],
  reactions: [
    {
      id: 'reaction-1',
      symbol: 'TLT',
      timeframe: '15m',
      prePrice: 91.1,
      postPrice: 92.2,
      changeAbs: 1.1,
      changePercent: 1.21,
      direction: 'up',
      confirmsEvent: true,
      tooExtended: false,
      noisy: false,
      reason: 'TLT confirmed the rates reaction.',
      createdAt: '2026-06-10T12:45:00Z',
    },
  ],
  scenarios: [],
  pricedInScores: [],
  confounders: [],
  evidenceBundles: [
    {
      id: 'evidence-1',
      symbol: 'TLT',
      status: 'complete',
      verdict: 'candidate_allowed',
      summary: 'Reaction and evidence support a paper candidate.',
      evidence: { reaction: 'confirmed' },
      missingEvidence: [],
      walkawayReasons: ['Fed speaker invalidates setup'],
      createdAt: '2026-06-10T12:46:00Z',
    },
  ],
  candidates: [
    {
      id: 'candidate-1',
      macroEventId: 'macro-event-1',
      evidenceBundleId: 'evidence-1',
      symbol: 'TLT',
      side: 'long',
      bias: 'duration_bid',
      entryType: 'pullback_retest',
      entryReferencePrice: 92.2,
      stopReferencePrice: 91.5,
      targetReferencePrice: 94,
      riskPercent: 0.01,
      timeLimit: 'same_session',
      status: 'awaiting_human_approval',
      createdReason: 'Paper candidate only after macro evidence bundle.',
      walkawayReasons: ['Fed speaker invalidates setup'],
      createdAt: '2026-06-10T12:47:00Z',
      humanApprovalRequired: true,
    },
  ],
};

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MacroEventsPage />
    </QueryClientProvider>
  );
}

describe('MacroEventsPage', () => {
  beforeEach(() => {
    vi.mocked(macroService.listEvents).mockResolvedValue(eventList);
    vi.mocked(macroService.getEvent).mockResolvedValue(eventDetail);
    vi.mocked(macroService.approveCandidate).mockResolvedValue({
      ...eventDetail.candidates[0],
      status: 'watch_only',
      humanApprovalRequired: false,
    });
    vi.mocked(macroService.rejectCandidate).mockResolvedValue({
      ...eventDetail.candidates[0],
      status: 'rejected',
      humanApprovalRequired: false,
    });
    vi.mocked(emitAnalyticsEvent).mockClear();
  });

  it('renders macro event list, reaction snapshot, evidence bundle, and paper candidate review', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { name: 'Macro Events' })).toBeInTheDocument();
    expect((await screen.findAllByText('CPI cooler than expected')).length).toBeGreaterThan(0);
    expect(await screen.findByText('TLT / 15m')).toBeInTheDocument();
    expect(screen.getByText('Reaction and evidence support a paper candidate.')).toBeInTheDocument();
    expect(screen.getAllByText('Fed speaker invalidates setup').length).toBeGreaterThan(0);
    expect(screen.getByText('Approval here changes macro candidate status only; it does not send an order.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Mark watch only/i })).toBeInTheDocument();
  });

  it('marks a macro candidate watch-only without routing to execution approval', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('Paper candidate only after macro evidence bundle.');
    await user.click(screen.getByRole('button', { name: /Mark watch only/i }));

    expect(macroService.approveCandidate).toHaveBeenCalledWith(
      'candidate-1',
      'Paper candidate reviewed in Macro Events'
    );
  });
});
