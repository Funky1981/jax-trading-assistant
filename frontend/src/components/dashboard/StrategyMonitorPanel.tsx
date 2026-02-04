import { Cpu, Play, Pause } from 'lucide-react';
import { useStrategiesSummary, Strategy } from '@/hooks/useStrategies';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn, formatCurrency, formatTime } from '@/lib/utils';

interface StrategyMonitorPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function StrategyMonitorPanel({ isOpen, onToggle }: StrategyMonitorPanelProps) {
  const { data: summary, strategies, isLoading } = useStrategiesSummary();

  const summaryText = summary ? (
    <span>
      {summary.active} active • {formatCurrency(summary.totalPnl)} P&L
      {summary.recentSignal && ` • Signal: ${summary.recentSignal.symbol}`}
    </span>
  ) : null;

  return (
    <CollapsiblePanel
      title="Strategy Monitor"
      icon={<Cpu className="h-4 w-4" />}
      summary={summaryText}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-3">
        {strategies?.map((strategy) => (
          <StrategyCard key={strategy.id} strategy={strategy} />
        ))}
      </div>
    </CollapsiblePanel>
  );
}

function StrategyCard({ strategy }: { strategy: Strategy }) {
  const isActive = strategy.status === 'active';
  
  return (
    <div className="rounded-md border border-border bg-muted/30 p-4">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="flex items-center gap-2">
            <h4 className="font-semibold">{strategy.name}</h4>
            <Badge
              variant={
                strategy.status === 'active'
                  ? 'success'
                  : strategy.status === 'paused'
                  ? 'warning'
                  : 'secondary'
              }
              className="text-xs"
            >
              {strategy.status}
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            {strategy.description}
          </p>
        </div>
        <Button variant="ghost" size="icon" className="h-8 w-8">
          {isActive ? (
            <Pause className="h-4 w-4" />
          ) : (
            <Play className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Performance Metrics */}
      <div className="grid grid-cols-4 gap-4 text-center mb-3">
        <div>
          <p className="text-xs text-muted-foreground">P&L</p>
          <p
            className={cn(
              'font-mono font-semibold',
              strategy.performance.totalPnl > 0
                ? 'text-success'
                : strategy.performance.totalPnl < 0
                ? 'text-destructive'
                : ''
            )}
          >
            {formatCurrency(strategy.performance.totalPnl)}
          </p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Win Rate</p>
          <p className="font-mono font-semibold">
            {strategy.performance.winRate.toFixed(1)}%
          </p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Trades</p>
          <p className="font-mono font-semibold">{strategy.performance.trades}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Sharpe</p>
          <p className="font-mono font-semibold">
            {strategy.performance.sharpe.toFixed(2)}
          </p>
        </div>
      </div>

      {/* Last Signal */}
      {strategy.lastSignal && (
        <div className="flex items-center justify-between text-xs border-t border-border pt-2">
          <span className="text-muted-foreground">Last Signal</span>
          <div className="flex items-center gap-2">
            <span
              className={cn(
                'font-semibold uppercase',
                strategy.lastSignal.action === 'buy'
                  ? 'text-success'
                  : 'text-destructive'
              )}
            >
              {strategy.lastSignal.action} {strategy.lastSignal.symbol}
            </span>
            <span className="text-muted-foreground">
              ({(strategy.lastSignal.confidence * 100).toFixed(0)}% conf)
            </span>
            <span className="text-muted-foreground">
              {formatTime(strategy.lastSignal.timestamp)}
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
