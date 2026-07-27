import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import { aiService, type WorldMonitorInboxItem } from '@/data/ai-service';
import { HttpError } from '@/data/http-client';
import { MonitorInboxPage } from './MonitorInboxPage';

vi.mock('@/data/ai-service', () => ({ aiService: { getWorldMonitorInbox: vi.fn() } }));
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

const accepted: WorldMonitorInboxItem = {
  id: 'inbox-accepted',
  source: 'world-monitor',
  sourceEventId: 'accepted-1',
  worldMonitorEventId: 'accepted-1',
  status: 'candidate_created',
  eventType: 'macro_rates',
  headline: 'Accepted monitor item with a complete persisted headline',
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
  decision: {
    decisionId: 'decision-1',
    decision: 'CANDIDATE',
    decisionVersion: 1,
    rulesetVersion: 'genuine-event-decision-v1',
    processorIdentity: 'jax-genuine-event-decision-processor',
    processingMode: 'deterministic',
    decisionAt: '2026-07-27T12:00:00Z',
    evidenceScore: 0.82,
    evidenceScoreSource: 'candidate_evidence_scores',
    affectedAssets: ['QQQ'],
    unknownAssets: false,
    assetMappingProvenance: { mappingMethod: 'provider_symbol' },
    reasons: ['Existing candidate passed the complete persisted contract.'],
    blockingReasons: [],
    missingEvidence: [],
    trustGateState: 'ready_for_risk_review',
    riskReviewState: 'ready_for_approval_review',
    candidateId: 'candidate-1',
    replayIdentity: 'gedr_1',
    createdAt: '2026-07-27T12:00:00Z',
    updatedAt: '2026-07-27T12:00:00Z',
  },
  decisionHistory: [],
  outcomeCount: 0,
  rawPayload: { fixture: true, monitor_score: 0.82 },
};

const rejected: WorldMonitorInboxItem = {
  ...accepted,
  id: 'inbox-rejected',
  sourceEventId: 'rejected-1',
  worldMonitorEventId: 'rejected-1',
  status: 'rejected',
  rejectionReason: 'source_urls are required',
  headline: 'Rejected synthetic monitor item',
  summary: undefined,
  sourceUrls: [],
  sourceCount: 0,
  eventTime: '2026-06-12T10:20:00Z',
  receivedAt: '2026-06-12T10:21:00Z',
  collectedAt: undefined,
  provenanceAvailable: true,
  isSynthetic: true,
  syntheticReason: 'labelled fixture',
  discoveryMethod: undefined,
  analysisIdentity: undefined,
  possibleAffectedEtfs: [],
  confidenceReasons: [],
  mappingReason: 'Rejected before mapping.',
  normalizedEventId: undefined,
  candidateId: undefined,
  candidateSymbol: undefined,
  candidateStatus: undefined,
  candidateCreatedAt: undefined,
  decision: undefined,
  decisionHistory: [],
  rawPayload: { bad: true },
};

function item(index: number): WorldMonitorInboxItem {
  return {
    ...accepted,
    id: `inbox-${index}`,
    sourceEventId: `source-${index}`,
    worldMonitorEventId: `world-${index}`,
    headline: `Evidence record ${index}`,
    candidateId: index % 2 ? `candidate-${index}` : undefined,
    candidateSymbol: index % 2 ? 'QQQ' : undefined,
    candidateStatus: index % 2 ? 'awaiting_approval' : undefined,
  };
}

function response(
  items = [accepted, rejected, ...Array.from({ length: 23 }, (_, index) => item(index + 3))],
) {
  return {
    total: items.length,
    counts: {
      genuine: items.filter((entry) => entry.isSynthetic === false).length,
      syntheticTests: items.filter((entry) => entry.isSynthetic === true).length,
      candidatesCreated: items.filter((entry) => entry.candidateId).length,
      rejected: items.filter((entry) => entry.status === 'rejected').length,
      duplicates: 0,
      noTrade: items.filter((entry) => entry.decision?.decision === 'NO_TRADE').length,
      watch: items.filter((entry) => entry.decision?.decision === 'WATCH').length,
      candidate: items.filter((entry) => entry.decision?.decision === 'CANDIDATE').length,
      awaitingProcessing: items.filter((entry) => !entry.decision && entry.status !== 'rejected')
        .length,
    },
    checkedAt: '2026-06-12T10:45:00Z',
    items,
  };
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <MonitorInboxPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function itemButtons() {
  return within(screen.getByRole('list', { name: 'Evidence items' })).getAllByRole('button');
}

describe('MonitorInboxPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(aiService.getWorldMonitorInbox).mockResolvedValue(response());
  });

  it('starts with ten compact collapsed items and no permanent detail placeholder', async () => {
    renderPage();
    await screen.findByText('1–10 of 25');
    expect(itemButtons()).toHaveLength(10);
    expect(itemButtons().every((button) => button.getAttribute('aria-expanded') === 'false')).toBe(
      true,
    );
    expect(screen.getByText('1–10 of 25')).toBeInTheDocument();
    expect(screen.queryByText(/Open an evidence item to see its source/i)).not.toBeInTheDocument();
    expect(
      document.querySelector('.sticky, [class*="overflow-y"], [class*="h-screen"]'),
    ).toBeNull();
  });

  it('supports twenty-item pages plus next and previous pagination', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Evidence items per page' }),
      '20',
    );
    expect(itemButtons()).toHaveLength(20);
    expect(screen.getByText('1–20 of 25')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Next evidence page' }));
    expect(itemButtons()).toHaveLength(5);
    expect(screen.getByText('21–25 of 25')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Previous evidence page' }));
    expect(screen.getByText('1–20 of 25')).toBeInTheDocument();
  });

  it('resets pagination and expansion when filtering without mutating the read model', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    await user.click(screen.getByRole('button', { name: 'Next evidence page' }));
    await user.click(screen.getByRole('button', { name: 'Rejected' }));
    expect(screen.getByText('1–1 of 1')).toBeInTheDocument();
    expect(itemButtons()).toHaveLength(1);
    expect(aiService.getWorldMonitorInbox).toHaveBeenCalledTimes(1);
    expect(aiService.getWorldMonitorInbox).toHaveBeenLastCalledWith({ limit: 100 });
  });

  it('allows only one inline item to expand and keeps subsections collapsed', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    const first = itemButtons()[0];
    const second = itemButtons()[1];
    await user.click(first);
    expect(first).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('Rates news mapped to QQQ.')).toBeVisible();
    expect(screen.getByRole('link', { name: 'Open original source' })).toHaveAttribute(
      'href',
      'https://example.com/accepted',
    );
    expect(screen.getByText('Published time')).toBeVisible();
    expect(screen.getAllByText('Collection time').some((node) => !node.closest('details'))).toBe(
      true,
    );
    expect(screen.getAllByText('Jax receipt time').some((node) => !node.closest('details'))).toBe(
      true,
    );
    for (const label of ['Source and provenance', 'Analysis', /Journey —/, 'Audit']) {
      expect(screen.getByText(label).closest('details')).not.toHaveAttribute('open');
    }
    expect(screen.getByText('Show raw payload')).not.toBeVisible();
    await user.click(second);
    expect(second).toHaveAttribute('aria-expanded', 'true');
    expect(first).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('Rates news mapped to QQQ.')).not.toBeInTheDocument();
  });

  it('exposes truthful analysis, asset mapping, candidate linkage and nested audit on request', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    await user.click(itemButtons()[0]);
    expect(screen.getByRole('link', { name: 'Open Candidate Review' })).toHaveAttribute(
      'href',
      '/candidates/candidate-1/evidence',
    );
    await user.click(screen.getByText('Analysis'));
    expect(screen.getByText('DETERMINISTIC ANALYSIS')).toBeVisible();
    expect(screen.getByText('No AI used')).toBeVisible();
    expect(screen.getAllByText('QQQ').length).toBeGreaterThan(0);
    await user.click(screen.getByText('Audit'));
    expect(screen.getByText('Source-event ID')).toBeVisible();
    expect(screen.getByText('Show raw payload').closest('details')).not.toHaveAttribute('open');
    expect(screen.getByText(/monitor_score/)).not.toBeVisible();
    expect(
      screen.queryByRole('button', { name: /^(approve|execute|trade)$/i }),
    ).not.toBeInTheDocument();
  });

  it('does not fabricate unknown assets and provides a valid no-candidate outcome', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    await user.click(itemButtons()[1]);
    expect(screen.queryByRole('link', { name: 'Open Candidate Review' })).not.toBeInTheDocument();
    expect(screen.getByText('No candidate was created. This is a valid outcome.')).toBeVisible();
    await user.click(screen.getByText('Analysis'));
    expect(screen.getByText('Unknown assets')).toBeVisible();
    expect(screen.getByText(/none was fabricated/i)).toBeVisible();
    expect(screen.getAllByText('SYNTHETIC TEST').length).toBeGreaterThan(0);
  });

  it('shows persisted decision outcomes and only links candidates for CANDIDATE', async () => {
    const user = userEvent.setup();
    const noTrade: WorldMonitorInboxItem = {
      ...accepted,
      id: 'no-trade',
      headline: 'Persisted no-trade event',
      decision: {
        ...accepted.decision!,
        decisionId: 'decision-no-trade',
        decision: 'NO_TRADE',
        candidateId: undefined,
        reasons: ['Persisted evidence was below the stronger-decision contract.'],
        blockingReasons: ['candidate_evidence_missing'],
        missingEvidence: ['candidate_evidence_score'],
      },
      decisionHistory: [{ ...accepted.decision!, decisionId: 'decision-no-trade-history' }],
    };
    const watch: WorldMonitorInboxItem = {
      ...accepted,
      id: 'watch',
      headline: 'Persisted watch event',
      candidateId: undefined,
      possibleAffectedEtfs: [],
      decision: {
        ...accepted.decision!,
        decisionId: 'decision-watch',
        decision: 'WATCH',
        candidateId: undefined,
        affectedAssets: [],
        unknownAssets: true,
        missingEvidence: ['truthful_asset_mapping'],
      },
      decisionHistory: [],
    };
    vi.mocked(aiService.getWorldMonitorInbox).mockResolvedValue(response([noTrade, watch]));
    renderPage();
    expect((await screen.findAllByText('NO_TRADE')).length).toBeGreaterThan(0);
    expect(screen.getAllByText('WATCH').length).toBeGreaterThan(0);
    await user.click(itemButtons()[0]);
    expect(screen.queryByRole('link', { name: 'Open Candidate Review' })).not.toBeInTheDocument();
    await user.click(screen.getByText('Decision — NO_TRADE'));
    expect(screen.getByText('candidate_evidence_missing')).toBeVisible();
    expect(screen.getByText('candidate_evidence_score')).toBeVisible();
    await user.click(screen.getByRole('button', { name: 'WATCH' }));
    expect(screen.getByText('1–1 of 1')).toBeInTheDocument();
  });

  it('operates the accordion by keyboard and has accessible pagination', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('1–10 of 25');
    const first = itemButtons()[0];
    first.focus();
    await user.keyboard('{Enter}');
    expect(first).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('button', { name: 'Previous evidence page' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next evidence page' })).toBeEnabled();
  });

  it('distinguishes unavailable and protected inbox responses', async () => {
    vi.mocked(aiService.getWorldMonitorInbox).mockRejectedValue(
      new HttpError('Request failed: 404', 404, '404 page not found'),
    );
    const { unmount } = renderPage();
    expect(await screen.findByText(/Evidence Inbox unavailable/i)).toBeInTheDocument();
    unmount();
    vi.mocked(aiService.getWorldMonitorInbox).mockRejectedValue(
      new HttpError('Request failed: 401', 401, 'missing authorization token'),
    );
    renderPage();
    expect(await screen.findByText(/Sign in to view evidence/i)).toBeInTheDocument();
  });

  it('has no detectable accessibility violations collapsed or expanded', async () => {
    const user = userEvent.setup();
    const { container } = renderPage();
    await screen.findByText('1–10 of 25');
    expect((await axe(container)).violations).toHaveLength(0);
    await user.click(itemButtons()[0]);
    expect((await axe(container)).violations).toHaveLength(0);
  });
});
