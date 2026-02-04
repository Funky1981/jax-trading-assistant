import { useState } from 'react';
import { Brain, TrendingUp, TrendingDown, Minus, Eye, AlertTriangle, Target, Shield, Loader2 } from 'lucide-react';
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
  
  const { data: health, isError: healthError } = useAIHealth();
  const { data: config } = useAIConfig();
  const { mutate: getSuggestion, data: suggestionData, isPending, error } = useAISuggestion();

  const handleGetSuggestion = () => {
    if (!symbol.trim()) return;
    getSuggestion({ symbol: symbol.toUpperCase(), context: context || undefined });
  };

  const isHealthy = !healthError && health?.status === 'healthy';

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
              placeholder="Symbol (e.g., AAPL)"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value.toUpperCase())}
              className="w-32"
            />
            <Input
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
