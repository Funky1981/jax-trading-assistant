import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { candidatesService } from '@/data/approvals-service';
import { CandidateEvidencePage } from './CandidateEvidencePage';

vi.mock('@/data/approvals-service', async () => {
  const actual = await vi.importActual<typeof import('@/data/approvals-service')>('@/data/approvals-service');
  return {
    ...actual,
    candidatesService: {
      get: vi.fn(),
    },
  };
});

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return render(
    <MemoryRouter initialEntries={['/candidates/candidate-1/evidence']}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route path="/candidates/:candidateId/evidence" element={<CandidateEvidencePage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('CandidateEvidencePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(candidatesService.get).mockResolvedValue({
      id: 'candidate-1',
      strategyInstanceId: 'instance-1',
      symbol: 'SPY',
      signalType: 'BUY',
      status: 'awaiting_approval',
      entryPrice: 530,
      stopLoss: 527.5,
      takeProfit: 536,
      confidence: 0.84,
      reasoning: 'Softer inflation news supports a tactical SPY paper long while price holds above trend.',
      sessionDate: '2026-06-12',
      detectedAt: '2026-06-12T10:30:00Z',
      expiresAt: '2026-06-12T11:00:00Z',
      dataProvenance: 'world-monitor',
      metadata: {
        worldMonitor: {
          source: 'world-monitor',
          sourceEventId: 'wm-1',
          headline: 'Inflation cools more than expected',
          summary: 'Treasury yields moved lower after the inflation print.',
          eventType: 'macro_rates',
          sourceURLs: ['https://example.com/inflation', 'https://example.com/yields'],
          sourceCount: 2,
          assetThemes: ['rates', 'growth'],
          confidenceReasons: ['trusted macro source', 'mapped to SPY'],
          mappingReason: 'SPY was selected because broad US equities often react to lower-rate surprises.',
          route: 'approval_required',
        },
        chartConfirmation: {
          confirmed: true,
          reasonCode: 'above_sma20',
          reason: 'SPY held above the 20-period moving average and the last five candles were positive.',
          candleCount: 30,
          lastClose: 531.25,
          sma20: 528.1,
          fiveCandleChangePct: 0.012,
          checkedAt: '2026-06-12T10:31:00Z',
        },
      },
      sentiment: {
        label: 'positive',
        state: 'available',
        score: 0.68,
        confidence: 0.74,
        window: '24h',
        sourceCount: 2,
        priceAgreement: 'agreeing',
        topDrivers: ['Lower yields supported equity risk appetite.'],
        limitations: ['Only two trusted sources were available.'],
        summary: 'News tone is positive for broad US equity exposure.',
        snapshotAt: '2026-06-12T10:32:00Z',
      },
    });
  });

  it('shows the complete trade evidence needed before approval or manual review', async () => {
    renderPage();

    expect(await screen.findByRole('heading', { name: 'SPY trade setup' })).toBeInTheDocument();
    expect(screen.getByText(/Softer inflation news supports a tactical SPY paper long/i)).toBeInTheDocument();
    expect(screen.getByText(/SPY was selected because broad US equities/i)).toBeInTheDocument();

    expect(screen.getByRole('heading', { name: /News articles/i })).toBeInTheDocument();
    expect(screen.getByText('Inflation cools more than expected')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /example.com\/inflation/i })).toHaveAttribute('href', 'https://example.com/inflation');
    expect(screen.getByRole('link', { name: /example.com\/yields/i })).toHaveAttribute('href', 'https://example.com/yields');

    expect(screen.getByRole('heading', { name: /What the charts are saying/i })).toBeInTheDocument();
    expect(screen.getByText('Chart confirmed')).toBeInTheDocument();
    expect(screen.getByText(/held above the 20-period moving average/i)).toBeInTheDocument();
    expect(screen.getByText('$531.25')).toBeInTheDocument();

    expect(screen.getByRole('heading', { name: 'Sentiment and source' })).toBeInTheDocument();
    expect(screen.getByText('News tone is positive for broad US equity exposure.')).toBeInTheDocument();
    expect(screen.getByText('Lower yields supported equity risk appetite.')).toBeInTheDocument();
    expect(screen.getByText('Only two trusted sources were available.')).toBeInTheDocument();

    expect(screen.getByRole('heading', { name: 'Suggested paper trade amount' })).toBeInTheDocument();
    expect(screen.getByText('40 shares')).toBeInTheDocument();
    expect(screen.getByText('$21,200.00 estimated notional')).toBeInTheDocument();
    expect(screen.getByText('$100.00')).toBeInTheDocument();
    expect(screen.getByText('2.40R')).toBeInTheDocument();
  });
});
