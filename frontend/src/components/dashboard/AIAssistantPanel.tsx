import { useEffect, useMemo, useState } from 'react';
import { Brain, TrendingUp, TrendingDown, Minus, Eye, AlertTriangle, Target, Shield, Loader2, Bell, BellOff } from 'lucide-react';
import { useAISuggestion, useAIHealth, useAIConfig, type Action, type AISuggestion } from '@/hooks/useAISuggestion';
import { CollapsiblePanel, StatusDot } from './CollapsiblePanel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

interface AIAssistantPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

type ScanAlert = {
  id: string;
  symbol: string;
  action: Action;
  confidence: number;
  timestamp: string;
  reason: string;
};

const actionConfig: Record<Action, { icon: React.ReactNode; color: string; bgColor: string }> = {
  BUY: {
    icon: <TrendingUp className="h-5 w-5" />,
    color: 'text-emerald-500',
    bgColor: 'bg-emerald-500/10',
  },
  SELL: {
    icon: <TrendingDown className="h-5 w-5" />,
    color: 'text-red-500',
    bgColor: 'bg-red-500/10',
  },
  HOLD: {
    icon: <Minus className="h-5 w-5" />,
    color: 'text-yellow-500',
    bgColor: 'bg-yellow-500/10',
  },
  WATCH: {
    icon: <Eye className="h-5 w-5" />,
    color: 'text-blue-500',
    bgColor: 'bg-blue-500/10',
  },
};

function ConfidenceBarFill({ percentage }: { percentage: number }) {
  return (
    <div
      className={cn(
        'h-full rounded-full transition-all duration-500',
        percentage >= 70 ? 'bg-emerald-500' : percentage >= 40 ? 'bg-yellow-500' : 'bg-red-500'
      )}
      style={{ width: `${percentage}%` }}
    />
  );
}

function ConfidenceBar({ confidence }: { confidence: number }) {
  const percentage = Math.round(confidence * 100);
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden">
        <ConfidenceBarFill percentage={percentage} />
      </div>
      <span className="text-xs font-medium">{percentage}%</span>
    </div>
  );
}

function SuggestionCard({ suggestion, provider, model }: { suggestion: AISuggestion; provider: string; model: string }) {
  const config = actionConfig[suggestion.action];

  return (
    <div className="space-y-4">
      {/* Action Header */}
      <div className={cn('flex items-center gap-3 p-4 rounded-lg', config.bgColor)}>
        <div className={cn('p-2 rounded-full', config.bgColor, config.color)}>
          {config.icon}
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className={cn('text-2xl font-bold', config.color)}>
              {suggestion.action}
            </span>
            <Badge variant="outline" className="text-xs">
              {suggestion.symbol}
            </Badge>
          </div>
          <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
            <Brain className="h-3 w-3" />
            <span>{provider}/{model}</span>
          </div>
        </div>
      </div>

      {/* Confidence */}
      <div className="space-y-1">
        <div className="flex items-center justify-between text-sm">
          <span className="font-medium">Confidence</span>
        </div>
        <ConfidenceBar confidence={suggestion.confidence} />
      </div>

      {/* Price Targets */}
      {(suggestion.entry_price || suggestion.target_price || suggestion.stop_loss) && (
        <div className="grid grid-cols-3 gap-2">
          {suggestion.entry_price && (
            <div className="p-2 rounded-md bg-muted/50 text-center">
              <p className="text-xs text-muted-foreground">Entry</p>
              <p className="text-sm font-semibold">${suggestion.entry_price.toFixed(2)}</p>
            </div>
          )}
          {suggestion.target_price && (
            <div className="p-2 rounded-md bg-emerald-500/10 text-center">
              <p className="text-xs text-muted-foreground flex items-center justify-center gap-1">
                <Target className="h-3 w-3" /> Target
              </p>
              <p className="text-sm font-semibold text-emerald-500">${suggestion.target_price.toFixed(2)}</p>
            </div>
          )}
          {suggestion.stop_loss && (
            <div className="p-2 rounded-md bg-red-500/10 text-center">
              <p className="text-xs text-muted-foreground flex items-center justify-center gap-1">
                <Shield className="h-3 w-3" /> Stop
              </p>
              <p className="text-sm font-semibold text-red-500">${suggestion.stop_loss.toFixed(2)}</p>
            </div>
          )}
        </div>
      )}

      {/* Position Size */}
      {suggestion.position_size_pct && (
        <div className="flex items-center justify-between p-2 rounded-md bg-muted/50">
          <span className="text-sm">Position Size</span>
          <Badge variant="secondary">{suggestion.position_size_pct}% of portfolio</Badge>
        </div>
      )}

      {/* Reasoning */}
      <div className="space-y-2">
        <p className="text-sm font-medium">Analysis</p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          {suggestion.reasoning}
        </p>
      </div>

      {/* Risk Assessment */}
      <div className="space-y-2">
        <p className="text-sm font-medium flex items-center gap-1">
          <AlertTriangle className="h-4 w-4 text-yellow-500" />
          Risk Assessment
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          {suggestion.risk_assessment}
        </p>
      </div>

      {/* Timestamp */}
      <p className="text-xs text-muted-foreground text-right">
        Generated: {new Date(suggestion.timestamp).toLocaleString()}
      </p>
    </div>
  );
}

export function AIAssistantPanel({ isOpen, onToggle }: AIAssistantPanelProps) {
  const [symbol, setSymbol] = useState('AAPL');
  const [context, setContext] = useState('');
  const [scanSymbolsText, setScanSymbolsText] = useState('SPY,QQQ,SMH');
  const [autoScanEnabled, setAutoScanEnabled] = useState(false);
  const [scanIntervalSeconds, setScanIntervalSeconds] = useState(60);
  const [confidenceThreshold, setConfidenceThreshold] = useState(0.65);
  const [alerts, setAlerts] = useState<ScanAlert[]>([]);
  const [isAutoScanning, setIsAutoScanning] = useState(false);
  const [scanError, setScanError] = useState<string | null>(null);
  
  const { data: health, isError: healthError } = useAIHealth();
  const { data: config } = useAIConfig();
  const { mutate: getSuggestion, mutateAsync: getSuggestionAsync, data: suggestionData, isPending, error } = useAISuggestion();

  const scanSymbols = useMemo(
    () =>
      scanSymbolsText
        .split(',')
        .map((value) => value.trim().toUpperCase())
        .filter(Boolean),
    [scanSymbolsText]
  );

  const canUseDesktopNotifications = typeof window !== 'undefined' && 'Notification' in window;
  const desktopNotificationsGranted = canUseDesktopNotifications && Notification.permission === 'granted';
  const isHealthy = !healthError && health?.status === 'healthy';

  const handleGetSuggestion = () => {
    if (!symbol.trim()) return;
    getSuggestion({ symbol: symbol.toUpperCase(), context: context || undefined });
  };

  const handleEnableDesktopNotifications = async () => {
    if (!canUseDesktopNotifications) {
      return;
    }
    await Notification.requestPermission();
  };

  useEffect(() => {
    if (!autoScanEnabled || !isHealthy || scanSymbols.length === 0) {
      return;
    }

    let cancelled = false;

    const runScanCycle = async () => {
      if (cancelled) {
        return;
      }

      setIsAutoScanning(true);
      setScanError(null);

      try {
        for (const scanSymbol of scanSymbols) {
          if (cancelled) {
            return;
          }

          const response = await getSuggestionAsync({
            symbol: scanSymbol,
            context: context || 'Auto scan for news+chart opportunities.',
          });

          const candidate = response.suggestion;
          const actionable = (candidate.action === 'BUY' || candidate.action === 'SELL') && candidate.confidence >= confidenceThreshold;

          if (!actionable) {
            continue;
          }

          const alert: ScanAlert = {
            id: `${candidate.symbol}-${candidate.timestamp}`,
            symbol: candidate.symbol,
            action: candidate.action,
            confidence: candidate.confidence,
            timestamp: candidate.timestamp,
            reason: candidate.reasoning,
          };

          setAlerts((prev) => [alert, ...prev].slice(0, 20));

          if (canUseDesktopNotifications && Notification.permission === 'granted') {
            new Notification(`Jax ${candidate.action} ${candidate.symbol}`, {
              body: `Confidence ${(candidate.confidence * 100).toFixed(0)}%`,
            });
          }
        }
      } catch (err) {
        setScanError(err instanceof Error ? err.message : 'Auto scan failed.');
      } finally {
        if (!cancelled) {
          setIsAutoScanning(false);
        }
      }
    };

    void runScanCycle();
    const timer = window.setInterval(() => {
      void runScanCycle();
    }, Math.max(scanIntervalSeconds, 20) * 1000);

    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [
    autoScanEnabled,
    isHealthy,
    scanSymbols,
    getSuggestionAsync,
    context,
    confidenceThreshold,
    scanIntervalSeconds,
    canUseDesktopNotifications,
  ]);

  const summary = suggestionData?.suggestion ? (
    <div className="flex items-center gap-2">
      <StatusDot status={isHealthy ? 'healthy' : 'unhealthy'} />
      <Badge
        variant={
          suggestionData.suggestion.action === 'BUY' ? 'success' :
          suggestionData.suggestion.action === 'SELL' ? 'destructive' :
          'secondary'
        }
      >
        {suggestionData.suggestion.action} {suggestionData.suggestion.symbol}
      </Badge>
      <span className="text-xs">
        {Math.round(suggestionData.suggestion.confidence * 100)}% confidence
      </span>
    </div>
  ) : (
    <div className="flex items-center gap-2">
      <StatusDot status={isHealthy ? 'healthy' : 'unhealthy'} />
      <span className="text-xs">
        {config?.provider || 'AI'} ready
      </span>
    </div>
  );

  return (
    <CollapsiblePanel
      title="AI Trading Assistant"
      icon={<Brain className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <div className="space-y-4">
        <div className="rounded-md border border-border bg-muted/20 px-3 py-3 text-xs text-muted-foreground">
          <p className="font-semibold uppercase tracking-wide text-foreground">AI Opportunity Scanning</p>
          <p className="mt-2">
            This panel can run continuous symbol scans using AI + chart context and raise in-app alerts. Mobile push delivery depends on your backend approval pipeline/Telegram bridge.
          </p>
        </div>

        {/* Service Status */}
        <div className="flex items-center justify-between p-2 rounded-md bg-muted/50">
          <div className="flex items-center gap-2">
            <StatusDot status={isHealthy ? 'healthy' : 'unhealthy'} />
            <span className="text-sm">
              {isHealthy ? 'AI Service Online' : 'AI Service Offline'}
            </span>
          </div>
          {config && (
            <Badge variant="outline" className="text-xs">
              {config.provider}/{config.model}
            </Badge>
          )}
        </div>

        {/* Input Form */}
        <div className="space-y-2">
          <div className="flex gap-2">
            <Input
              id="ai-symbol"
              name="symbol"
              aria-label="Trading symbol"
              placeholder="Symbol (e.g., AAPL)"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value.toUpperCase())}
              className="w-32"
            />
            <Input
              id="ai-context"
              name="context"
              aria-label="Additional trading context"
              placeholder="Additional context (optional)"
              value={context}
              onChange={(e) => setContext(e.target.value)}
              className="flex-1"
            />
            <Button
              onClick={handleGetSuggestion}
              disabled={isPending || !symbol.trim() || !isHealthy}
            >
              {isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Analyzing
                </>
              ) : (
                <>
                  <Brain className="h-4 w-4 mr-2" />
                  Get Suggestion
                </>
              )}
            </Button>
          </div>
        </div>

        <div className="space-y-3 rounded-md border border-border bg-muted/20 p-3">
          <div className="flex items-center justify-between gap-2">
            <p className="text-sm font-medium">Auto Scan & Notifications</p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant={desktopNotificationsGranted ? 'default' : 'outline'}
                size="sm"
                onClick={handleEnableDesktopNotifications}
                disabled={!canUseDesktopNotifications || desktopNotificationsGranted}
                title={!canUseDesktopNotifications ? 'Desktop notifications are unavailable in this browser.' : undefined}
              >
                {desktopNotificationsGranted ? <Bell className="h-4 w-4" /> : <BellOff className="h-4 w-4" />}
                {desktopNotificationsGranted ? 'Desktop Alerts On' : 'Enable Desktop Alerts'}
              </Button>
              <Button
                type="button"
                variant={autoScanEnabled ? 'destructive' : 'default'}
                size="sm"
                onClick={() => setAutoScanEnabled((prev) => !prev)}
                disabled={!isHealthy || scanSymbols.length === 0}
              >
                {autoScanEnabled ? 'Stop Auto Scan' : 'Start Auto Scan'}
              </Button>
            </div>
          </div>

          <div className="grid gap-2 md:grid-cols-3">
            <Input
              aria-label="Auto scan symbols"
              placeholder="SPY,QQQ,SMH"
              value={scanSymbolsText}
              onChange={(event) => setScanSymbolsText(event.target.value.toUpperCase())}
            />
            <Input
              aria-label="Auto scan interval seconds"
              type="number"
              min={20}
              step={5}
              value={scanIntervalSeconds}
              onChange={(event) => setScanIntervalSeconds(Number(event.target.value) || 60)}
            />
            <Input
              aria-label="Auto scan confidence threshold"
              type="number"
              min={0.4}
              max={0.95}
              step={0.05}
              value={confidenceThreshold}
              onChange={(event) => setConfidenceThreshold(Number(event.target.value) || 0.65)}
            />
          </div>

          <p className="text-xs text-muted-foreground">
            Status: {autoScanEnabled ? (isAutoScanning ? 'scanning now' : 'monitoring') : 'stopped'} • Symbols: {scanSymbols.join(', ') || 'none'}
          </p>

          {scanError && (
            <p className="text-xs text-destructive">{scanError}</p>
          )}
        </div>

        {/* Error Display */}
        {error && (
          <div className="p-3 rounded-md bg-destructive/10 border border-destructive/20">
            <p className="text-sm text-destructive flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {error instanceof Error ? error.message : 'Failed to get suggestion'}
            </p>
          </div>
        )}

        {/* Suggestion Display */}
        {suggestionData && (
          <SuggestionCard
            suggestion={suggestionData.suggestion}
            provider={suggestionData.provider}
            model={suggestionData.model}
          />
        )}

        {/* Empty State */}
        {!suggestionData && !isPending && !error && (
          <div className="p-8 text-center text-muted-foreground">
            <Brain className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p className="text-sm">Enter a symbol and click "Get Suggestion"</p>
            <p className="text-xs mt-1">
              The AI will analyze market data and provide trading recommendations
            </p>
          </div>
        )}

        {alerts.length > 0 && (
          <div className="space-y-2 rounded-md border border-border bg-muted/20 p-3">
            <p className="text-sm font-medium">Recent Scan Alerts</p>
            <div className="space-y-2">
              {alerts.slice(0, 5).map((alert) => (
                <div key={alert.id} className="rounded-md border border-border bg-background px-2.5 py-2 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold">{alert.action} {alert.symbol}</span>
                    <span className="text-muted-foreground">{Math.round(alert.confidence * 100)}%</span>
                  </div>
                  <p className="mt-1 line-clamp-2 text-muted-foreground">{alert.reason}</p>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Provider Info */}
        {config && (
          <p className="text-xs text-muted-foreground text-center">
            Using {config.provider === 'ollama' ? 'Ollama (FREE - Local)' : config.provider} • {config.model}
          </p>
        )}
      </div>
    </CollapsiblePanel>
  );
}
