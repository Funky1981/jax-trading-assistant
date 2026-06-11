import type { StrategyInstance } from '@/data/types';

export interface GuidedTemplate {
  id: string;
  title: string;
  description: string;
  strategyKeywords: string[];
  defaultMarketId: string;
}

export interface GuidedMarket {
  id: string;
  label: string;
  symbols: string[];
}

export interface GuidedBacktestRequest {
  instanceId: string;
  from: string;
  to: string;
  symbolsOverride: string[];
  datasetId: string;
}

export const GUIDED_TEMPLATES: GuidedTemplate[] = [
  {
    id: 'trend_breakout',
    title: 'Trend breakout',
    description: 'Find momentum breakouts with confirmation signals.',
    strategyKeywords: ['opening_range', 'orb', 'breakout', 'momentum'],
    defaultMarketId: 'spy_sample',
  },
  {
    id: 'mean_reversion',
    title: 'Pullback reversion',
    description: 'Look for pullbacks that revert toward the recent trend.',
    strategyKeywords: ['reversion', 'mean', 'pullback'],
    defaultMarketId: 'us_index_etfs',
  },
  {
    id: 'risk_off_rotation',
    title: 'Risk-off rotation',
    description: 'Track defensive rotation and downside pressure setups.',
    strategyKeywords: ['defensive', 'rotation', 'risk'],
    defaultMarketId: 'macro_rotation',
  },
];

export const GUIDED_MARKETS: GuidedMarket[] = [
  {
    id: 'spy_sample',
    label: 'SPY only (available sample dataset)',
    symbols: ['SPY'],
  },
  {
    id: 'us_index_etfs',
    label: 'US index ETFs (SPY, QQQ, IWM)',
    symbols: ['SPY', 'QQQ', 'IWM'],
  },
  {
    id: 'macro_rotation',
    label: 'Macro rotation basket (SPY, XLF, XLE, TLT, GLD)',
    symbols: ['SPY', 'XLF', 'XLE', 'TLT', 'GLD'],
  },
  {
    id: 'growth_vs_value',
    label: 'Growth vs value (QQQ, IWM, DIA)',
    symbols: ['QQQ', 'IWM', 'DIA'],
  },
];

export function resolveGuidedInstance(instances: StrategyInstance[], template: GuidedTemplate): StrategyInstance | null {
  const keywords = template.strategyKeywords.map((keyword) => keyword.toLowerCase());

  const matches = instances.filter((instance) => {
    const target = `${instance.strategyId ?? ''} ${instance.strategyTypeId ?? ''} ${instance.name ?? ''}`.toLowerCase();
    return keywords.some((keyword) => target.includes(keyword));
  });

  return matches.find((instance) => instance.enabled) ?? matches[0] ?? null;
}

export function resolveGuidedMarket(marketId: string): GuidedMarket {
  return GUIDED_MARKETS.find((market) => market.id === marketId) ?? GUIDED_MARKETS[0];
}

export function buildGuidedBacktestRequest(input: {
  instanceId: string;
  from: string;
  to: string;
  marketId: string;
  datasetId: string;
}): GuidedBacktestRequest {
  const market = resolveGuidedMarket(input.marketId);

  return {
    instanceId: input.instanceId,
    from: input.from,
    to: input.to,
    symbolsOverride: market.symbols,
    datasetId: input.datasetId,
  };
}
