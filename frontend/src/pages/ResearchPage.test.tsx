import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ResearchPage } from './ResearchPage';
import { instancesService } from '@/data/instances-service';
import { backtestService } from '@/data/backtest-service';
import { researchService } from '@/data/research-service';
import { datasetsService } from '@/data/datasets-service';
import { emitAnalyticsEvent } from '@/lib/analytics';

vi.mock('@/data/instances-service', () => ({
  instancesService: {
    list: vi.fn(),
    listStrategyTypes: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    enable: vi.fn(),
    disable: vi.fn(),
  },
}));

vi.mock('@/data/backtest-service', () => ({
  backtestService: {
    run: vi.fn(),
    list: vi.fn(),
  },
}));

vi.mock('@/data/research-service', () => ({
  researchService: {
    listProjects: vi.fn(),
    listProjectRuns: vi.fn(),
    createProject: vi.fn(),
    runProject: vi.fn(),
  },
}));

vi.mock('@/data/datasets-service', () => ({
  datasetsService: {
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
      mutations: { retry: false },
    },
  });

  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <ResearchPage />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('ResearchPage guided wizard', () => {
  beforeEach(() => {
    vi.mocked(emitAnalyticsEvent).mockClear();
    vi.mocked(instancesService.list).mockResolvedValue([
      {
        id: 'instance-orb',
        name: 'ORB paper setup',
        strategyTypeId: 'orb_v1',
        strategyId: 'orb',
        enabled: true,
        sessionTimezone: 'America/New_York',
        flattenByCloseTime: '15:55',
        configJson: { symbols: ['SPY'] },
      },
    ] as never);
    vi.mocked(instancesService.listStrategyTypes).mockResolvedValue([] as never);
    vi.mocked(backtestService.list).mockResolvedValue([] as never);
    vi.mocked(backtestService.run).mockResolvedValue({ runId: 'run-1', status: 'queued' } as never);
    vi.mocked(researchService.listProjects).mockResolvedValue([] as never);
    vi.mocked(researchService.listProjectRuns).mockResolvedValue([] as never);
    vi.mocked(datasetsService.list).mockResolvedValue({
      datasets: [
        {
          datasetId: 'dataset-1',
          datasetHash: 'hash-1',
          name: 'ETF baseline',
          symbol: 'SPY',
        },
      ],
      limit: 200,
      offset: 0,
    } as never);
  });

  it('shows wizard steps and builds guided backtest request', async () => {
    const user = userEvent.setup();

    renderPage();

    expect(await screen.findByText('Guided Research Wizard')).toBeInTheDocument();
    expect(screen.getByText('Step 1: Strategy template')).toBeInTheDocument();
    expect(screen.getByText('Step 2: Market scope')).toBeInTheDocument();
    expect(screen.getByText('Step 3: Backtest period')).toBeInTheDocument();
    expect(screen.getByText('Step 4: Sentiment feature')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Run guided backtest' }));

    await waitFor(() => {
      expect(backtestService.run).toHaveBeenCalledWith(
        expect.objectContaining({
          instanceId: 'instance-orb',
          strategyId: 'orb',
          datasetId: 'dataset-1',
          symbolsOverride: ['SPY'],
        })
      );
    });

    expect(vi.mocked(emitAnalyticsEvent)).toHaveBeenCalledWith(
      'backtest_sentiment_enabled',
      expect.objectContaining({ source_surface: 'research', sentiment_mode: 'boost' })
    );

    await user.click(screen.getByRole('link', { name: 'Teach me sentiment' }));
    expect(vi.mocked(emitAnalyticsEvent)).toHaveBeenCalledWith(
      'teach_me_sentiment_opened',
      expect.objectContaining({ source_surface: 'research', sentiment_mode: 'boost' })
    );
  });

  it('shows missing-data guidance when no dataset snapshots exist', async () => {
    vi.mocked(datasetsService.list).mockResolvedValue({ datasets: [], limit: 200, offset: 0 } as never);

    renderPage();

    expect(await screen.findByText(/Research needs a prepared dataset snapshot before running/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'System' })).toHaveAttribute('href', '/system');
  });
});
