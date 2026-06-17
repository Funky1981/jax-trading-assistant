import { describe, expect, it, vi } from 'vitest';
import { tradingModesService } from './trading-modes-service';
import { apiClient } from './http-client';

vi.mock('./http-client', () => ({
  apiClient: {
    get: vi.fn(),
  },
}));

describe('tradingModesService', () => {
  it('adds Swing display metadata without filtering existing modes', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      modes: [
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
      ],
    });

    const modes = await tradingModesService.list();

    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/trading-modes');
    expect(modes).toHaveLength(2);
    expect(modes[0].name).toBe('Manual Trading');
    expect(modes[1]).toMatchObject({
      id: 'etf_swing_paper',
      horizonLabel: 'Swing',
      displayCopy:
        'Swing Trading researches multi-day ETF setups. It creates approval-gated paper candidates only after evidence, chart history, and daily revalidation checks pass.',
    });
  });
});
