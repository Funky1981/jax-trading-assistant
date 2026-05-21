import { useState } from 'react';
import { Info } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ETF_DESCRIPTIONS, type UXMode, explainRiskLevel } from '@/utils/beginner-helpers';

interface ETFCardProps {
  symbol: string;
  mode: UXMode;
}

function ETFCard({ symbol, mode }: ETFCardProps) {
  const [expanded, setExpanded] = useState(false);
  const etf = ETF_DESCRIPTIONS[symbol];

  if (!etf) return null;

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
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <CardTitle className="text-lg">{symbol}</CardTitle>
            <CardDescription>{etf.name}</CardDescription>
          </div>
          <Badge className={riskColors[etf.riskLevel]}>
            {riskEmoji[etf.riskLevel]} {etf.riskLevel}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm font-medium text-foreground">
          {mode === 'simple' ? etf.simple : etf.detailed}
        </p>

        {expanded && (
          <div className="space-y-3 pt-2 border-t">
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-1">Risk Level Explained</p>
              <p className="text-sm">{explainRiskLevel(etf.riskLevel, mode)}</p>
            </div>

            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-1">Affected By</p>
              <div className="flex flex-wrap gap-1">
                {etf.categories.map((cat) => (
                  <Badge key={cat} variant="outline" className="text-xs">
                    {cat.replace(/_/g, ' ')}
                  </Badge>
                ))}
              </div>
            </div>
          </div>
        )}

        <Button
          variant="outline"
          size="sm"
          className="w-full"
          onClick={() => setExpanded(!expanded)}
        >
          <Info className="w-4 h-4 mr-2" />
          {expanded ? 'Hide Details' : 'Learn More'}
        </Button>
      </CardContent>
    </Card>
  );
}

export interface ETFUniverseScreenProps {
  mode: UXMode;
  onSelectETF?: (symbol: string) => void;
}

/**
 * ETF Universe Screen - Shows all approved ETFs with simple explanations
 * Part of Step 9: Beginner UX
 */
export function ETFUniverseScreen({ mode, onSelectETF }: ETFUniverseScreenProps) {
  const approvedETFs = ['SPY', 'QQQ', 'DIA', 'IWM', 'XLK', 'XLF', 'XLE', 'SMH', 'SOXX', 'TLT', 'GLD'];

  const categoryMap = new Map<string, string[]>();
  approvedETFs.forEach((symbol) => {
    const etf = ETF_DESCRIPTIONS[symbol];
    if (etf) {
      etf.categories.forEach((cat) => {
        if (!categoryMap.has(cat)) {
          categoryMap.set(cat, []);
        }
        categoryMap.get(cat)!.push(symbol);
      });
    }
  });

  return (
    <div className="space-y-8 p-6">
      <div>
        <h1 className="text-3xl font-bold mb-2">ETF Universe</h1>
        <p className="text-muted-foreground">
          {mode === 'simple'
            ? 'These are the ETFs Jax can trade. Each one represents a different part of the market.'
            : 'Approved ETF catalog for phase-1 paper trading. All holdings are diversified instruments.'}
        </p>
      </div>

      {/* ETF Grid */}
      <div>
        <h2 className="text-xl font-semibold mb-4">All Approved ETFs</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {approvedETFs.map((symbol) => (
            <div key={symbol} onClick={() => onSelectETF?.(symbol)} className="cursor-pointer">
              <ETFCard symbol={symbol} mode={mode} />
            </div>
          ))}
        </div>
      </div>

      {/* Category Guide */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-semibold text-blue-900 mb-3">How to Read the Impact Categories</h3>
        <div className="space-y-2 text-sm text-blue-800">
          <p>
            <strong>AI & Tech:</strong> ETFs that move when there's news about artificial intelligence, semiconductors,
            or software.
          </p>
          <p>
            <strong>Macro & Rates:</strong> ETFs sensitive to interest rate decisions, inflation reports, or broad
            economic changes.
          </p>
          <p>
            <strong>Bonds & Commodities:</strong> Non-equity instruments that often move opposite to stocks.
          </p>
        </div>
      </div>

      {/* Risk Levels Explained */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="border-green-200 bg-green-50">
          <CardHeader>
            <CardTitle className="text-green-900">🛡️ Low Risk</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-green-800">
            Smaller daily moves. Good for beginners. Examples: TLT, GLD
          </CardContent>
        </Card>

        <Card className="border-yellow-200 bg-yellow-50">
          <CardHeader>
            <CardTitle className="text-yellow-900">⚠️ Medium Risk</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-yellow-800">
            Normal market moves. Standard for most traders. Examples: SPY, DIA, XLF
          </CardContent>
        </Card>

        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="text-red-900">⚡ High Risk</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-red-800">
            Big swings. For experienced traders. Examples: QQQ, IWM, SMH, SOXX
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
