import { Shield } from 'lucide-react';
import { useRiskSummary } from '@/hooks/useRisk';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Progress } from '@/components/ui/progress';
import { cn, formatCurrency, formatPercent } from '@/lib/utils';

interface RiskSummaryPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function RiskSummaryPanel({ isOpen, onToggle }: RiskSummaryPanelProps) {
  const { data: summary, metrics, isLoading } = useRiskSummary();

  const summaryText = summary ? (
    <div className="flex items-center gap-3">
      <span>{formatCurrency(summary.exposure)}</span>
      <span className="text-muted-foreground">
        ({summary.utilizationPercent.toFixed(1)}% utilized)
      </span>
    </div>
  ) : null;

  return (
    <CollapsiblePanel
      title="Risk Summary"
      icon={<Shield className="h-4 w-4" />}
      summary={summaryText}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      {metrics && (
        <div className="space-y-4">
          {/* Exposure */}
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span>Exposure</span>
              <span className="font-mono">
                {formatCurrency(metrics.exposure.current)} / {formatCurrency(metrics.exposure.limit)}
              </span>
            </div>
            <Progress
              value={metrics.exposure.utilizationPercent}
              className="h-2"
              indicatorClassName={cn(
                metrics.exposure.utilizationPercent > 80
                  ? 'bg-destructive'
                  : metrics.exposure.utilizationPercent > 60
                  ? 'bg-warning'
                  : 'bg-success'
              )}
            />
          </div>

          {/* Daily P&L */}
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span>Daily P&L</span>
              <span
                className={cn(
                  'font-mono',
                  metrics.dailyPnl.current > 0 ? 'text-success' : metrics.dailyPnl.current < 0 ? 'text-destructive' : ''
                )}
              >
                {formatCurrency(metrics.dailyPnl.current)}
              </span>
            </div>
            <Progress
              value={Math.abs(metrics.dailyPnl.utilizationPercent)}
              className="h-2"
              indicatorClassName={cn(
                metrics.dailyPnl.current >= 0 ? 'bg-success' : 'bg-destructive'
              )}
            />
          </div>

          {/* Drawdown */}
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span>Drawdown</span>
              <span className="font-mono">
                {formatPercent(-metrics.drawdown.current, 1)} / {formatPercent(-metrics.drawdown.limit, 1)} max
              </span>
            </div>
            <Progress
              value={metrics.drawdown.utilizationPercent}
              className="h-2"
              indicatorClassName={cn(
                metrics.drawdown.utilizationPercent > 70
                  ? 'bg-destructive'
                  : metrics.drawdown.utilizationPercent > 50
                  ? 'bg-warning'
                  : 'bg-success'
              )}
            />
          </div>

          {/* Position Count */}
          <div className="flex justify-between text-sm border-t border-border pt-3">
            <span>Positions</span>
            <span className="font-mono">
              {metrics.positionCount.current} / {metrics.positionCount.limit}
            </span>
          </div>

          {/* Largest Position */}
          <div className="flex justify-between text-sm">
            <span>Largest Position</span>
            <span className="font-mono">
              {metrics.largestPosition.symbol} ({metrics.largestPosition.percentOfPortfolio.toFixed(1)}%)
            </span>
          </div>

          {/* Sector Exposure */}
          <div className="border-t border-border pt-3">
            <p className="text-sm font-medium mb-2">Sector Exposure</p>
            <div className="space-y-1">
              {metrics.sectorExposure.map((sector) => (
                <div key={sector.sector} className="flex justify-between text-xs">
                  <span className="text-muted-foreground">{sector.sector}</span>
                  <span className="font-mono">{sector.percent.toFixed(1)}%</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </CollapsiblePanel>
  );
}
