import { AlertCircle, TrendingUp, Shield, AlertTriangle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import {
  type UXMode,
  formatConfidenceForBeginners,
  formatPrice,
  ETF_DESCRIPTIONS,
} from '@/utils/beginner-helpers';
import type { CandidateTrade } from '@/data/approvals-service';

export interface CandidateTradeData {
  candidate: CandidateTrade;
  newsHeadline?: string;
  newsSource?: string;
  priceHistory?: {
    beforeNews: number;
    afterNews: number;
    percentChange: number;
  };
  pricedInVerdictScore?: number; // 0-100, higher = more priced in
  pricedInExplanation?: string;
  confounders?: string[];
  conflictingNews?: string[];
  riskControlsSummary?: string;
}

interface TradeStatusIndicatorProps {
  status: string;
}

function TradeStatusIndicator({ status }: TradeStatusIndicatorProps) {
  const statusMap: Record<
    string,
    { color: string; icon: React.ReactNode; label: string; description: string }
  > = {
    detected: {
      color: 'bg-blue-100 text-blue-800',
      icon: <AlertCircle className="w-4 h-4" />,
      label: 'Detected',
      description: 'Signal found by Jax',
    },
    qualified: {
      color: 'bg-green-100 text-green-800',
      icon: <TrendingUp className="w-4 h-4" />,
      label: 'Qualified',
      description: 'Passes all checks',
    },
    blocked: {
      color: 'bg-red-100 text-red-800',
      icon: <AlertTriangle className="w-4 h-4" />,
      label: 'Blocked',
      description: 'Did not pass checks',
    },
    approved: {
      color: 'bg-green-100 text-green-800',
      icon: <TrendingUp className="w-4 h-4" />,
      label: 'Approved',
      description: 'Ready to trade',
    },
    submitted: {
      color: 'bg-purple-100 text-purple-800',
      icon: <TrendingUp className="w-4 h-4" />,
      label: 'Submitted',
      description: 'Sent to broker',
    },
    filled: {
      color: 'bg-green-100 text-green-800',
      icon: <TrendingUp className="w-4 h-4" />,
      label: 'Filled',
      description: 'Trade executed',
    },
  };

  const info = statusMap[status] || statusMap['detected'];
  return (
    <div className={`${info.color} rounded-full px-3 py-1 flex items-center gap-2`}>
      {info.icon}
      <span className="text-sm font-semibold">{info.label}</span>
    </div>
  );
}

export interface CandidateTradeSummaryProps {
  data: CandidateTradeData;
  mode: UXMode;
  onApprove?: () => void;
  onReject?: () => void;
  onSnooze?: () => void;
  loading?: boolean;
}

/**
 * Candidate Trade Summary Screen - Shows a single candidate trade in beginner-friendly terms
 * Part of Step 9: Beginner UX
 */
export function CandidateTradeSummary({
  data,
  mode,
  onApprove,
  onReject,
  onSnooze,
  loading = false,
}: CandidateTradeSummaryProps) {
  const { candidate, newsHeadline, newsSource, priceHistory, pricedInVerdictScore, pricedInExplanation, confounders, conflictingNews } = data;

  const etfInfo = ETF_DESCRIPTIONS[candidate.symbol];
  const signalEmoji = candidate.signalType === 'BUY' ? '📈' : '📉';
  const signalColor = candidate.signalType === 'BUY' ? 'text-green-600' : 'text-red-600';
  const badgeVariant = candidate.signalType === 'BUY' ? 'default' : 'destructive';

  // Calculate movement from entry price
  const entryToStop = candidate.stopLoss && candidate.entryPrice ? 
    (((candidate.stopLoss - candidate.entryPrice) / candidate.entryPrice) * 100).toFixed(2) : null;
  const entryToTarget = candidate.takeProfit && candidate.entryPrice ?
    (((candidate.takeProfit - candidate.entryPrice) / candidate.entryPrice) * 100).toFixed(2) : null;

  return (
    <div className="space-y-6 p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <h1 className={`text-4xl font-bold ${signalColor}`}>{signalEmoji}</h1>
              <div>
                <h2 className="text-3xl font-bold">{candidate.symbol}</h2>
                <p className="text-muted-foreground">{etfInfo?.name || 'ETF'}</p>
              </div>
            </div>
          </div>
          <TradeStatusIndicator status={candidate.status} />
        </div>

        <div className="flex items-center gap-3">
          <Badge variant={badgeVariant} className="text-base px-3 py-1">
            {candidate.signalType}
          </Badge>
          <span className="text-lg font-semibold">
            Confidence: {formatConfidenceForBeginners(candidate.confidence, mode)}
          </span>
        </div>
      </div>

      {/* Plain-English Summary */}
      {mode === 'simple' && (
        <Card className="bg-blue-50 border-blue-200">
          <CardHeader>
            <CardTitle className="text-blue-900">What Jax Found</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-blue-800">
            <p className="text-base">
              {candidate.reasoning ||
                `${candidate.signalType === 'BUY' ? 'Positive news pushed' : 'Negative news pushed'} ${candidate.symbol} ${candidate.signalType === 'BUY' ? 'up' : 'down'}. Jax thinks the move is real and the momentum might continue.`}
            </p>

            {newsHeadline && (
              <div className="bg-white rounded p-3 border border-blue-100">
                <p className="text-xs font-semibold text-blue-600 mb-1">NEWS TRIGGER</p>
                <p className="font-medium">{newsHeadline}</p>
                {newsSource && <p className="text-xs text-muted-foreground mt-1">Source: {newsSource}</p>}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Price Analysis */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="w-5 h-5" />
            Price Analysis
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {priceHistory && (
            <div className="grid grid-cols-3 gap-4 text-center">
              <div className="bg-gray-50 rounded p-3">
                <p className="text-xs font-semibold text-muted-foreground mb-1">BEFORE NEWS</p>
                <p className="text-lg font-bold">{formatPrice(priceHistory.beforeNews, mode)}</p>
              </div>
              <div className="bg-blue-50 rounded p-3 flex items-center justify-center">
                <p className={`text-2xl font-bold ${priceHistory.percentChange > 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {priceHistory.percentChange > 0 ? '+' : ''}
                  {priceHistory.percentChange.toFixed(2)}%
                </p>
              </div>
              <div className="bg-gray-50 rounded p-3">
                <p className="text-xs font-semibold text-muted-foreground mb-1">AFTER NEWS</p>
                <p className="text-lg font-bold">{formatPrice(priceHistory.afterNews, mode)}</p>
              </div>
            </div>
          )}

          <div className="grid grid-cols-3 gap-3 pt-3 border-t">
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-2">ENTRY</p>
              <p className="text-2xl font-bold">{formatPrice(candidate.entryPrice, mode)}</p>
              <p className="text-xs text-muted-foreground mt-1">Buy here</p>
            </div>
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-2">STOP-LOSS</p>
              <p className="text-lg font-bold text-red-600">{formatPrice(candidate.stopLoss, mode)}</p>
              {entryToStop && <p className="text-xs text-red-600 mt-1">{entryToStop}% down</p>}
              <p className="text-xs text-muted-foreground">Sell if drops</p>
            </div>
            <div>
              <p className="text-xs font-semibold text-muted-foreground mb-2">TARGET</p>
              <p className="text-lg font-bold text-green-600">{formatPrice(candidate.takeProfit, mode)}</p>
              {entryToTarget && <p className="text-xs text-green-600 mt-1">{entryToTarget}% up</p>}
              <p className="text-xs text-muted-foreground">Sell if rises</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Priced-In Analysis */}
      {pricedInVerdictScore !== undefined && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertCircle className="w-5 h-5" />
              Is This Already Priced In?
            </CardTitle>
            <CardDescription>How much of the news is already reflected in the price</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-4">
              <div className="flex-1">
                <div className="flex items-end gap-2 mb-2">
                  <span className="text-3xl font-bold">{pricedInVerdictScore}%</span>
                  <span className="text-sm text-muted-foreground mb-1">Priced In</span>
                </div>
                <Progress
                  value={pricedInVerdictScore}
                  indicatorClassName={
                    pricedInVerdictScore > 70
                      ? 'bg-red-500'
                      : pricedInVerdictScore > 40
                        ? 'bg-yellow-500'
                        : 'bg-green-500'
                  }
                />
              </div>
            </div>

            {pricedInExplanation && (
              <div className="bg-amber-50 border border-amber-200 rounded p-3 text-sm text-amber-800">
                {pricedInExplanation}
              </div>
            )}

            {pricedInVerdictScore <= 40 && (
              <p className="text-sm text-green-700 font-medium">✓ Good news! The market hasn't fully reacted yet. Room to profit.</p>
            )}
            {pricedInVerdictScore > 40 && pricedInVerdictScore <= 70 && (
              <p className="text-sm text-yellow-700 font-medium">⚠ Medium: The market has partially reacted. Some profit potential but limited.</p>
            )}
            {pricedInVerdictScore > 70 && (
              <p className="text-sm text-red-700 font-medium">✗ Most of this move is already done. Limited profit potential remains.</p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Risk Analysis */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="w-5 h-5" />
            Risk Controls
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {data.riskControlsSummary && <p className="text-sm">{data.riskControlsSummary}</p>}

          <div className="space-y-2">
            <div className="flex items-start gap-2">
              <span className="text-green-600 font-bold">✓</span>
              <span className="text-sm">Stop-loss is set. If you're wrong, losses are limited.</span>
            </div>
            <div className="flex items-start gap-2">
              <span className="text-green-600 font-bold">✓</span>
              <span className="text-sm">Paper trading only. No real money is at risk.</span>
            </div>
            <div className="flex items-start gap-2">
              <span className="text-green-600 font-bold">✓</span>
              <span className="text-sm">Position will auto-flatten at market close for safety.</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Confounders & Warnings */}
      {(confounders && confounders.length > 0) || (conflictingNews && conflictingNews.length > 0) ? (
        <Card className="border-yellow-200 bg-yellow-50">
          <CardHeader>
            <CardTitle className="text-yellow-900 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5" />
              Other Things to Know
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {confounders && confounders.length > 0 && (
              <div>
                <p className="font-semibold text-sm text-yellow-900 mb-2">Other News Out There:</p>
                <ul className="space-y-1">
                  {confounders.map((confounder, idx) => (
                    <li key={idx} className="text-sm text-yellow-800">
                      • {confounder}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {conflictingNews && conflictingNews.length > 0 && (
              <div>
                <p className="font-semibold text-sm text-yellow-900 mb-2">Conflicting Signals:</p>
                <ul className="space-y-1">
                  {conflictingNews.map((news, idx) => (
                    <li key={idx} className="text-sm text-yellow-800">
                      • {news}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </CardContent>
        </Card>
      ) : null}

      {/* Approval Buttons */}
      <div className="flex gap-3 pt-4 border-t">
        <Button
          variant="default"
          size="lg"
          className="flex-1"
          onClick={onApprove}
          disabled={loading}
        >
          ✓ Approve Trade
        </Button>
        <Button
          variant="outline"
          size="lg"
          className="flex-1"
          onClick={onReject}
          disabled={loading}
        >
          ✗ Reject
        </Button>
        <Button
          variant="ghost"
          size="lg"
          className="flex-1"
          onClick={onSnooze}
          disabled={loading}
        >
          ⏸ Snooze 4h
        </Button>
      </div>

      {mode !== 'simple' && (
        <div className="text-xs text-muted-foreground pt-4 border-t">
          <p>Trade ID: {candidate.id}</p>
          <p>Instance: {candidate.strategyInstanceId}</p>
        </div>
      )}
    </div>
  );
}
