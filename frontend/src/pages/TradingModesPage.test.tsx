import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TradingModesPage } from './TradingModesPage';
import { tradingModesService } from '../data/trading-modes-service';

vi.mock('../data/trading-modes-service', () => ({
  tradingModesService: {
    list: vi.fn(),
  },
}));

describe('TradingModesPage', () => {
  beforeEach(() => {
    vi.mocked(tradingModesService.list).mockResolvedValue([
      {
        id: 'manual',
        name: 'Manual Trading',
        description: 'Manual paper workflow.',
        assetClass: 'MULTI',
        runtimeMode: 'paper',
        executionPolicy: 'manual',
        universe: [],
        requiredData: [],
        riskDefaults: {
          maxTradesPerDay: 1,
          maxOpenPositions: 1,
          riskPerTradePct: 0.1,
          minConfidence: 0,
          flattenBy: '',
          approvalRequired: false,
        },
        strategies: [],
      },
      {
        id: 'etf_swing_paper',
        name: 'ETF Swing Paper',
        description: 'Backend description.',
        displayCopy:
          'Swing Trading researches multi-day ETF setups. It creates approval-gated paper candidates only after evidence, chart history, and daily revalidation checks pass.',
        horizonLabel: 'Swing',
        assetClass: 'ETF',
        runtimeMode: 'paper',
        executionPolicy: 'candidate_approval_only',
        universe: ['QQQ'],
        requiredData: ['daily_candles'],
        riskDefaults: {
          maxTradesPerDay: 1,
          maxOpenPositions: 1,
          riskPerTradePct: 0.15,
          minConfidence: 0.7,
          flattenBy: 'daily_revalidation',
          approvalRequired: true,
        },
        strategies: [],
      },
      {
        id: 'etf_news_paper',
        name: 'ETF News Paper',
        description: 'Intraday ETF news workflow.',
        assetClass: 'ETF',
        runtimeMode: 'paper',
        executionPolicy: 'candidate_approval_only',
        universe: ['SPY'],
        requiredData: ['candles_1m'],
        riskDefaults: {
          maxTradesPerDay: 3,
          maxOpenPositions: 1,
          riskPerTradePct: 0.25,
          minConfidence: 0.65,
          flattenBy: '15:55',
          approvalRequired: true,
        },
        strategies: [],
      },
    ]);
  });

  it('renders Swing Trading separately while keeping Manual and ETF news modes', async () => {
    render(<TradingModesPage />);

    expect(await screen.findByText('ETF Swing Paper')).toBeInTheDocument();
    expect(screen.getByText(/Swing Trading researches multi-day ETF setups/)).toBeInTheDocument();
    expect(screen.getByText('Swing')).toBeInTheDocument();
    expect(screen.getByText('Manual Trading')).toBeInTheDocument();
    expect(screen.getByText('ETF News Paper')).toBeInTheDocument();
    expect(screen.getAllByText('Universe').length).toBeGreaterThan(0);
  });
});
