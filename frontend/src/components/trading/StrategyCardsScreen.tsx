import { useState } from 'react';
import { ChevronDown, TrendingUp, AlertCircle, Clock } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { type UXMode } from '@/utils/beginner-helpers';

export interface StrategyCardData {
  id: string;
  name: string;
  description: string;
  whatItWatches: string;
  whenItTrades: string;
  whenItWalksAway: string;
  relatedETFs: string[];
  typicalHoldingTime: string;
  riskLevel: 'low' | 'medium' | 'high';
  exampleTrade?: {
    etf: string;
    scenario: string;
    entryReason: string;
    exitReason: string;
  };
  successRateEstimate?: string;
}

// ETF News Strategies (Step 9 requirement)
const ETF_NEWS_STRATEGIES: StrategyCardData[] = [
  {
    id: 'etf_news_001_market_panic_reversal',
    name: 'Market Panic Reversal',
    description: 'Trades broad market rebounds after sudden sell-offs triggered by negative news.',
    whatItWatches: 'Sudden big market drops (sell-offs) and panic headlines. Looks for reversals when the fear subsides.',
    whenItTrades:
      'Right after a market panic event (bad earnings, geopolitical shock). Enters on the rebound when selling pressure eases.',
    whenItWalksAway:
      'If the panic continues (market keeps falling). If no reversal shows up within 1-2 hours. If conflicting bad news keeps coming.',
    relatedETFs: ['SPY', 'QQQ', 'DIA', 'IWM'],
    typicalHoldingTime: '30 min - 2 hours',
    riskLevel: 'high',
    exampleTrade: {
      etf: 'SPY',
      scenario: 'Market falls 2% on bad earnings. After 15 minutes, selling stops and prices stabilize.',
      entryReason: 'Reversal signal detected — buyers are stepping in, panic is fading.',
      exitReason: 'Price recovered 1%, or 1 hour passed, or another negative headline drops.',
    },
    successRateEstimate: '~60% win rate when reversal occurs',
  },
  {
    id: 'etf_news_002_sector_momentum',
    name: 'Sector Momentum',
    description: 'Rides momentum in a specific sector after positive news (AI breakthroughs, earnings beats, M&A).',
    whatItWatches: 'Sector-specific positive news (AI chip announcements, energy policy, tech deals). Watches for continuing momentum.',
    whenItTrades:
      'After sector-specific good news breaks. Enters on strength, riding the momentum wave as money flows into that sector.',
    whenItWalksAway:
      'When momentum fades (lower highs). When conflicting negative news about the sector emerges. When the overall market turns down.',
    relatedETFs: ['XLK', 'XLF', 'XLE', 'SMH', 'SOXX', 'QQQ'],
    typicalHoldingTime: '1-4 hours',
    riskLevel: 'medium',
    exampleTrade: {
      etf: 'SMH',
      scenario: 'AI company announces breakthrough. Semiconductor stocks start climbing on the news.',
      entryReason: 'Sector ETF (SMH) is breaking to new highs. Money is flowing in. Momentum is positive.',
      exitReason: 'Gains slow down or other tech stocks lag. Stop-loss hit if sellers come back.',
    },
    successRateEstimate: '~55% win rate on sector continuation',
  },
  {
    id: 'etf_news_003_rates_bonds_rotation',
    name: 'Rates & Bonds Rotation',
    description: 'Trades bond (TLT) and rate-sensitive moves when central banks signal rate decisions or inflation data surprises.',
    whatItWatches: 'Fed announcements, inflation reports (CPI), employment data. Watches for big moves in bond prices and interest rates.',
    whenItTrades:
      'When rate-moving news (inflation report, Fed decision) hits. Bonds fall when rates rise, climb when rates fall. Rotates between stocks and bonds.',
    whenItWalksAway:
      'When no clear rate move happens. When conflicting macro signals cloud the picture. When bonds stabilize after the initial move.',
    relatedETFs: ['TLT', 'XLF', 'SPY'],
    typicalHoldingTime: '1-8 hours',
    riskLevel: 'medium',
    exampleTrade: {
      etf: 'TLT',
      scenario: 'CPI report shows inflation falling. Investors think Fed might cut rates soon.',
      entryReason: 'Bond prices (TLT) jump up because falling rates = higher bond values. Momentum is clear.',
      exitReason: 'Fed officials push back on rate cuts. Bond rally fades. Take profit on gains.',
    },
    successRateEstimate: '~50% win rate on rate moves',
  },
];

interface StrategyCardProps {
  strategy: StrategyCardData;
  onSelectStrategy?: (strategy: StrategyCardData) => void;
}

function StrategyCard({ strategy, onSelectStrategy }: StrategyCardProps) {
  const [expanded, setExpanded] = useState(false);

  const riskColors = {
    low: 'bg-green-100 text-green-800',
    medium: 'bg-yellow-100 text-yellow-800',
    high: 'bg-red-100 text-red-800',
  };

  const riskEmoji = {
    low: '🛡️',
    medium: '⚠️',
    high: '⚡',
  };

  return (
    <Card className="hover:shadow-lg transition-shadow">
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <CardTitle className="text-lg">{strategy.name}</CardTitle>
            <CardDescription className="mt-1">{strategy.description}</CardDescription>
          </div>
          <Badge className={riskColors[strategy.riskLevel]}>
            {riskEmoji[strategy.riskLevel]} {strategy.riskLevel}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div className="flex items-start gap-2">
            <Clock className="w-4 h-4 mt-0.5 text-muted-foreground flex-shrink-0" />
            <div>
              <p className="font-semibold text-xs text-muted-foreground">HOLD TIME</p>
              <p className="font-medium">{strategy.typicalHoldingTime}</p>
            </div>
          </div>

          <div className="flex items-start gap-2">
            <TrendingUp className="w-4 h-4 mt-0.5 text-muted-foreground flex-shrink-0" />
            <div>
              <p className="font-semibold text-xs text-muted-foreground">SUCCESS RATE</p>
              <p className="font-medium">{strategy.successRateEstimate || '—'}</p>
            </div>
          </div>
        </div>

        <div className="space-y-3 pt-2 border-t">
          <div>
            <p className="font-semibold text-xs text-muted-foreground mb-1">WHAT IT WATCHES</p>
            <p className="text-sm">{strategy.whatItWatches}</p>
          </div>

          <div>
            <p className="font-semibold text-xs text-muted-foreground mb-1">WHEN IT TRADES</p>
            <p className="text-sm">{strategy.whenItTrades}</p>
          </div>

          <div>
            <p className="font-semibold text-xs text-muted-foreground mb-1">TRADES THESE ETFs</p>
            <div className="flex flex-wrap gap-1">
              {strategy.relatedETFs.map((etf) => (
                <Badge key={etf} variant="outline">
                  {etf}
                </Badge>
              ))}
            </div>
          </div>
        </div>

        {expanded && (
          <div className="space-y-3 pt-3 border-t">
            <div className="bg-amber-50 border border-amber-200 rounded p-3">
              <div className="flex gap-2 items-start">
                <AlertCircle className="w-4 h-4 text-amber-600 mt-0.5 flex-shrink-0" />
                <div className="text-sm">
                  <p className="font-semibold text-amber-900 mb-1">When It Walks Away</p>
                  <p className="text-amber-800">{strategy.whenItWalksAway}</p>
                </div>
              </div>
            </div>

            {strategy.exampleTrade && (
              <div className="bg-blue-50 border border-blue-200 rounded p-3">
                <p className="font-semibold text-sm text-blue-900 mb-2">Example Trade: {strategy.exampleTrade.etf}</p>
                <div className="space-y-2 text-sm text-blue-800">
                  <p>
                    <strong>Scenario:</strong> {strategy.exampleTrade.scenario}
                  </p>
                  <p>
                    <strong>Why enter:</strong> {strategy.exampleTrade.entryReason}
                  </p>
                  <p>
                    <strong>Why exit:</strong> {strategy.exampleTrade.exitReason}
                  </p>
                </div>
              </div>
            )}
          </div>
        )}

        <Button
          variant="outline"
          size="sm"
          className="w-full"
          onClick={() => setExpanded(!expanded)}
        >
          <ChevronDown className={`w-4 h-4 mr-2 transition-transform ${expanded ? 'rotate-180' : ''}`} />
          {expanded ? 'Hide Details' : 'Learn More'}
        </Button>

        {onSelectStrategy && (
          <Button
            variant="default"
            size="sm"
            className="w-full"
            onClick={() => onSelectStrategy(strategy)}
          >
            Enable This Strategy
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

export interface StrategyCardsProps {
  mode: UXMode;
  onSelectStrategy?: (strategy: StrategyCardData) => void;
  showIntroduction?: boolean;
}

/**
 * Strategy Cards Screen - Shows beginner-friendly explanations of ETF trading strategies
 * Part of Step 9: Beginner UX
 */
export function StrategyCardsScreen({ mode, onSelectStrategy, showIntroduction = true }: StrategyCardsProps) {
  return (
    <div className="space-y-8 p-6">
      {showIntroduction && (
        <div>
          <h1 className="text-3xl font-bold mb-2">Trading Strategies</h1>
          <p className="text-muted-foreground">
            {mode === 'simple'
              ? 'Each strategy watches for specific market patterns and trades when the right conditions appear. Pick one or enable all to see suggestions.'
              : 'Strategy definitions for ETF news event trading. Each strategy specifies watch criteria, entry conditions, and exit rules.'}
          </p>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {ETF_NEWS_STRATEGIES.map((strategy) => (
          <StrategyCard
            key={strategy.id}
            strategy={strategy}
            onSelectStrategy={onSelectStrategy}
          />
        ))}
      </div>

      {mode === 'simple' && (
        <Card className="bg-blue-50 border-blue-200">
          <CardHeader>
            <CardTitle className="text-blue-900">How to Use Strategies</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-blue-800">
            <p>
              <strong>Enable all strategies</strong> to get suggestions whenever any pattern appears.
            </p>
            <p>
              <strong>Each strategy sends you alerts</strong> when it detects the pattern. You don't have to watch all day.
            </p>
            <p>
              <strong>You approve each trade</strong> before it happens. You stay in control.
            </p>
            <p>
              <strong>Paper trading only</strong> means no real money is at risk. This is practice.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
