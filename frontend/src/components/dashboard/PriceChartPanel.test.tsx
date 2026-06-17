import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PriceChartPanel } from './PriceChartPanel';
import { apiClient } from '@/data/http-client';

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addCandlestickSeries: vi.fn(() => ({
      setData: vi.fn(),
    })),
    timeScale: vi.fn(() => ({
      fitContent: vi.fn(),
    })),
    applyOptions: vi.fn(),
    remove: vi.fn(),
  })),
}));

vi.mock('@/data/http-client', async () => {
  const actual = await vi.importActual<typeof import('@/data/http-client')>('@/data/http-client');
  return {
    ...actual,
    apiClient: {
      get: vi.fn(),
    },
  };
});

vi.mock('@/hooks/useWatchlist', () => ({
  useWatchlist: () => ({
    data: [],
  }),
}));

vi.mock('@/hooks/useTradingPilotStatus', () => ({
  useTradingPilotStatus: () => ({
    data: {
      brokerConnected: true,
      readOnly: false,
      reasons: [],
    },
  }),
}));

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <PriceChartPanel isOpen onToggle={() => undefined} />
    </QueryClientProvider>
  );
}

describe('PriceChartPanel', () => {
  beforeEach(() => {
    vi.mocked(apiClient.get).mockReset();
  });

  it('shows honest zero-candle diagnostics with source and checked timestamp', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      symbol: 'AAPL',
      timeframe: '15m',
      requestedTimeframe: '15m',
      marketDataMode: 'delayed',
      paperTrading: true,
      source: 'ib-bridge',
      checkedAt: '2026-06-12T16:20:00Z',
      candles: [],
    });

    renderPanel();

    expect(await screen.findByText(/No usable candles returned for AAPL/i)).toBeInTheDocument();
    expect(screen.getByText(/At least 2 candles are required/i)).toBeInTheDocument();
    expect(screen.getAllByText(/Source: ib-bridge/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Last checked:/i).length).toBeGreaterThan(0);
  });

  it('shows degraded-data metadata when intraday candles fall back to daily candles', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      symbol: 'AAPL',
      timeframe: '1d',
      requestedTimeframe: '15m',
      degraded: true,
      message: 'intraday candles unavailable; showing daily fallback',
      marketDataMode: 'delayed',
      paperTrading: true,
      source: 'ib-bridge',
      checkedAt: '2026-06-12T16:25:00Z',
      candles: [
        { timestamp: '2026-06-11T20:00:00Z', open: 190, high: 195, low: 189, close: 194 },
        { timestamp: '2026-06-12T20:00:00Z', open: 194, high: 196, low: 193, close: 195 },
      ],
    });

    renderPanel();

    expect(await screen.findByText(/intraday candles unavailable; showing daily fallback/i)).toBeInTheDocument();
    expect(screen.getByText(/Candles: 2/i)).toBeInTheDocument();
    expect(screen.getAllByText(/Source: ib-bridge/i).length).toBeGreaterThan(0);
  });
});
