import { describe, expect, it } from 'vitest';
import { buildGuidedBacktestRequest, GUIDED_MARKETS, GUIDED_TEMPLATES, resolveGuidedInstance } from './research-guided-wizard';
import type { StrategyInstance } from '@/data/types';

describe('research guided wizard helpers', () => {
  it('maps guided request to backend run payload', () => {
    const payload = buildGuidedBacktestRequest({
      instanceId: 'instance-1',
      from: '2026-05-01',
      to: '2026-05-22',
      marketId: 'macro_rotation',
      datasetId: 'dataset-1',
    });

    expect(payload).toEqual({
      instanceId: 'instance-1',
      from: '2026-05-01',
      to: '2026-05-22',
      symbolsOverride: ['SPY', 'XLF', 'XLE', 'TLT', 'GLD'],
      datasetId: 'dataset-1',
    });
  });

  it('resolves strategy instances by template keywords', () => {
    const instances: StrategyInstance[] = [
      {
        id: 'instance-reversion',
        name: 'Pullback Reversion setup',
        strategyTypeId: 'mean_reversion_v1',
        strategyId: 'reversion',
        enabled: true,
        sessionTimezone: 'America/New_York',
        flattenByCloseTime: '15:55',
        configJson: { symbols: ['SPY'] },
      },
      {
        id: 'instance-orb',
        name: 'ORB trend setup',
        strategyTypeId: 'orb_v1',
        strategyId: 'orb',
        enabled: true,
        sessionTimezone: 'America/New_York',
        flattenByCloseTime: '15:55',
        configJson: { symbols: ['QQQ'] },
      },
    ];

    const matched = resolveGuidedInstance(instances, GUIDED_TEMPLATES[0]);

    expect(matched?.id).toBe('instance-orb');
  });

  it('keeps guided catalogs non-empty', () => {
    expect(GUIDED_TEMPLATES.length).toBeGreaterThan(0);
    expect(GUIDED_MARKETS.length).toBeGreaterThan(0);
  });
});
