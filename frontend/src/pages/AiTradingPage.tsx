import { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, ArrowRight, Bot, Clock, RefreshCw, ShieldCheck, Sparkles } from 'lucide-react';
import { approvalsService, candidatesService } from '@/data/approvals-service';
import { aiService } from '@/data/ai-service';
import { signalsService } from '@/data/signals-service';
import { toOpportunitySummaries } from '@/data/opportunity-adapter';
import type { AIScannerApiState, OpportunityRoute, OpportunitySummary, ScannerSettings } from '@/data/types';
import { ScannerSettingsCard } from '@/components/trading/ScannerSettingsCard';
import { useAISuggestion } from '@/hooks/useAISuggestion';
import { SentimentEvidencePanel } from '@/components/trading/SentimentEvidencePanel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useBeginnerMode } from '@/context/BeginnerUXContextValue';
import { emitAnalyticsEvent } from '@/lib/analytics';

const REFRESH_INTERVAL_MS = 30_000;
const STALE_DATA_MS = 5 * 60_000;
const WATCHED_STORAGE_KEY = 'jax-ai-trading-watched-opportunities';
const DISMISSED_STORAGE_KEY = 'jax-ai-trading-dismissed-opportunities';

const defaultScannerSettings: ScannerSettings = {
  enabled: true,
  assetScope: 'etf',
  symbols: ['SPY', 'QQQ', 'IWM'],
  universePreset: 'etf-core',
  intervalSeconds: 300,
  minimumConfidence: 0.7,
  connected: true,
  sentiment: {
    enabled: false,
    sourceScope: 'news',
    timeWindow: '24h',
    minimumThresholdLabel: '60%',
    minimumSourceCount: 3,
    sourceTrustMode: 'equal',
    mode: 'filter',
    supported: true,
    connected: true,
  },
};

function mapScannerToSettings(scanner?: AIScannerApiState): ScannerSettings {
  if (!scanner) {
    return defaultScannerSettings;
  }

  return {
    enabled: scanner.enabled,
    assetScope: scanner.assetScope,
    symbols: scanner.symbols,
    universePreset: scanner.universePreset,
    intervalSeconds: scanner.intervalSeconds,
    minimumConfidence: scanner.minimumConfidence,
    connected: true,
    sentiment: {
      enabled: scanner.sentiment.enabled,
      sourceScope: scanner.sentiment.sourceScope,
      timeWindow: scanner.sentiment.window,
      minimumThresholdLabel: `${Math.round(scanner.sentiment.threshold * 100)}%`,
      minimumSourceCount: scanner.sentiment.minimumSourceCount,
      sourceTrustMode: scanner.sentiment.sourceTrustWeightingMode,
      mode: scanner.sentiment.mode,
      supported: true,
      connected: true,
    },
  };
}

const routeLabels: Record<OpportunityRoute, string> = {
  manual_allowed: 'Manual review',
  approval_required: 'Approval required',
  execution_ready: 'Execution chain',
  blocked: 'Blocked',
};

function routeLabel(opportunity: OpportunitySummary) {
  if (opportunity.routeReasonCode === 'no_chart_confirmation') {
    return 'Needs chart confirmation';
  }
  return routeLabels[opportunity.route];
}

function formatConfidence(band: OpportunitySummary['confidenceBand']) {
  if (band === 'unknown') return 'Confidence unknown';
  return `${band[0].toUpperCase()}${band.slice(1)} confidence`;
}

function formatDateTime(value?: string) {
  if (!value) return 'Not set';
  const time = new Date(value);
  if (Number.isNaN(time.getTime())) return value;
  return time.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function isExpired(value?: string) {
  if (!value) return false;
  const time = new Date(value).getTime();
  return Number.isFinite(time) && time < Date.now();
}

function primaryAction(opportunity: OpportunitySummary) {
  if (opportunity.route === 'blocked') {
    return {
      label: opportunity.routeReasonCode === 'no_chart_confirmation' ? 'Review chart evidence' : 'Open blocked-state guidance',
      path:
        opportunity.sourceType === 'candidate' || opportunity.sourceType === 'approval'
          ? `/candidates/${opportunity.sourceId}/evidence`
          : '/research',
    };
  }

  if (opportunity.route === 'approval_required') {
    return { label: 'Send to approval', path: '/etf/approvals' };
  }

  if (opportunity.route === 'execution_ready') {
    return { label: 'View execution chain', path: '/approvals' };
  }

  return { label: 'Review order', path: '/manual-trading' };
}

function loadStoredIds(key: string): string[] {
  try {
    const raw = window.localStorage.getItem(key);
    const parsed = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((id): id is string => typeof id === 'string') : [];
  } catch {
    return [];
  }
}

function saveStoredIds(key: string, ids: string[]) {
  try {
    window.localStorage.setItem(key, JSON.stringify(ids));
  } catch {
    // Keep the interaction working in memory when storage is unavailable.
  }
}

function OpportunityCard({
  opportunity,
  watched,
  onWatch,
  onDismiss,
}: {
  opportunity: OpportunitySummary;
  watched: boolean;
  onWatch: (id: string) => void;
  onDismiss: (id: string) => void;
}) {
  const action = primaryAction(opportunity);
  const expired = isExpired(opportunity.expiresAt);

  useEffect(() => {
    if (!opportunity.sentiment) {
      return;
    }

    emitAnalyticsEvent('opportunity_sentiment_viewed', {
      source_surface: 'ai_trading',
      opportunity_id: opportunity.id,
      route_type: opportunity.route,
      sentiment_mode: 'api_scanner',
    });
  }, [opportunity.id, opportunity.route, opportunity.sentiment]);

  return (
    <Card className="overflow-hidden">
      <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <CardTitle className="text-xl">{opportunity.symbol}</CardTitle>
            <Badge variant={opportunity.route === 'blocked' ? 'destructive' : opportunity.route === 'approval_required' ? 'warning' : 'default'}>
              {routeLabel(opportunity)}
            </Badge>
            <Badge variant="outline">{opportunity.signalType}</Badge>
            <Badge variant="secondary">{formatConfidence(opportunity.confidenceBand)}</Badge>
          </div>
          <CardDescription>{opportunity.summary}</CardDescription>
        </div>
        <Badge variant={expired ? 'destructive' : 'outline'}>{expired ? 'Expired' : opportunity.status}</Badge>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 text-sm md:grid-cols-3">
          <div>
            <p className="font-medium text-foreground">Route reason</p>
            <p className="text-muted-foreground">{opportunity.routeReason}</p>
          </div>
          <div>
            <p className="font-medium text-foreground">Detected</p>
            <p className="text-muted-foreground">{formatDateTime(opportunity.detectedAt)}</p>
          </div>
          <div>
            <p className="font-medium text-foreground">Expiry</p>
            <p className="text-muted-foreground">{formatDateTime(opportunity.expiresAt)}</p>
          </div>
        </div>

        <SentimentEvidencePanel sentiment={opportunity.sentiment} />

        <div className="flex flex-wrap gap-2">
          <Button asChild>
            <Link to={action.path}>
              {action.label}
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
          <Button type="button" variant="outline" onClick={() => onWatch(opportunity.id)}>
            {watched ? 'Watching' : 'Watch'}
          </Button>
          <Button type="button" variant="ghost" onClick={() => onDismiss(opportunity.id)}>
            Dismiss
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

export function AiTradingPage() {
  const trackedSetupRef = useRef(false);
  const queryClient = useQueryClient();
  const { mode } = useBeginnerMode();
  const [watchedIds, setWatchedIds] = useState<string[]>(() => loadStoredIds(WATCHED_STORAGE_KEY));
  const [dismissedIds, setDismissedIds] = useState<string[]>(() => loadStoredIds(DISMISSED_STORAGE_KEY));
  const [aiSymbol, setAiSymbol] = useState('SPY');
  const aiSuggestion = useAISuggestion();

  const overviewQuery = useQuery({
    queryKey: ['ai-trading', 'overview'],
    queryFn: () => aiService.getOverview(),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: 60_000,
  });

  const signalsQuery = useQuery({
    queryKey: ['ai-trading', 'signals'],
    queryFn: () => signalsService.list({ status: 'pending', limit: 12 }),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: 60_000,
  });
  const candidatesQuery = useQuery({
    queryKey: ['ai-trading', 'candidates'],
    queryFn: () => candidatesService.list({ limit: 12 }),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: 60_000,
  });
  const approvalsQuery = useQuery({
    queryKey: ['ai-trading', 'approval-queue'],
    queryFn: () => approvalsService.getQueue(12),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: 60_000,
  });
  const monitorStatusQuery = useQuery({
    queryKey: ['ai-trading', 'world-monitor-status'],
    queryFn: () => aiService.getWorldMonitorStatus(),
    refetchInterval: REFRESH_INTERVAL_MS,
    staleTime: 60_000,
  });

  const allOpportunities = useMemo(
    () =>
      toOpportunitySummaries({
        signals: signalsQuery.data?.signals ?? [],
        candidates: candidatesQuery.data ?? [],
        approvals: approvalsQuery.data ?? [],
      }),
    [approvalsQuery.data, candidatesQuery.data, signalsQuery.data]
  );
  const opportunities = useMemo(
    () => allOpportunities.filter((opportunity) => !dismissedIds.includes(opportunity.id)),
    [allOpportunities, dismissedIds]
  );

  const scannerSettings = useMemo(
    () => mapScannerToSettings(overviewQuery.data?.scanner),
    [overviewQuery.data?.scanner]
  );

  const scannerMutation = useMutation({
    mutationFn: (state: AIScannerApiState) => aiService.updateScanner(state),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ai-trading', 'overview'] });
    },
  });

  const promoteSuggestionMutation = useMutation({
    mutationFn: () => {
      const suggestion = aiSuggestion.data?.suggestion;
      if (!suggestion || (suggestion.action !== 'BUY' && suggestion.action !== 'SELL')) {
        throw new Error('Only BUY or SELL suggestions can be promoted.');
      }
      return aiService.promoteSuggestion({
        symbol: suggestion.symbol,
        action: suggestion.action,
        confidence: suggestion.confidence,
        reasoning: suggestion.reasoning,
        risk: suggestion.risk_assessment,
        source: 'agent0_manual_review',
      });
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['ai-trading', 'overview'] }),
        queryClient.invalidateQueries({ queryKey: ['ai-trading', 'candidates'] }),
        queryClient.invalidateQueries({ queryKey: ['ai-trading', 'approval-queue'] }),
      ]);
    },
  });

  const queries = [overviewQuery, signalsQuery, candidatesQuery, approvalsQuery];
  const isLoading = queries.some((query) => query.isPending);
  const isError = queries.some((query) => query.isError);
  const isFetching = queries.some((query) => query.isFetching);
  const latestUpdate = Math.max(...queries.map((query) => query.dataUpdatedAt).filter(Boolean), 0);
  const staleData = latestUpdate > 0 && Date.now() - latestUpdate > STALE_DATA_MS;
  const rawScannerStatus = overviewQuery.data?.scanner.status;
  const scannerStatus = rawScannerStatus
    ? `${rawScannerStatus.charAt(0).toUpperCase()}${rawScannerStatus.slice(1)}`
    : isError
      ? 'Degraded'
      : isFetching
        ? 'Scanning'
        : 'Ready';
  const opportunitiesCount =
    (overviewQuery.data?.opportunityCounts.signalsPending ?? 0) +
    (overviewQuery.data?.opportunityCounts.candidates ?? 0) +
    (overviewQuery.data?.opportunityCounts.approvals ?? 0);
  const watchedCount = watchedIds.filter((id) => allOpportunities.some((opportunity) => opportunity.id === id)).length;
  const suggestionAction = aiSuggestion.data?.suggestion.action;
  const suggestionCanPromote = suggestionAction === 'BUY' || suggestionAction === 'SELL';
  const suggestionPromoteLabel =
    aiSuggestion.data && scannerSettings.symbols.includes(aiSuggestion.data.suggestion.symbol.toUpperCase())
      ? 'Send to approval queue'
      : 'Send to opportunity queue';

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'ai_trading' });
  }, []);

  useEffect(() => {
    if (trackedSetupRef.current) {
      return;
    }

    trackedSetupRef.current = true;
    if (!overviewQuery.data?.scanner) {
      return;
    }

    emitAnalyticsEvent('ai_scanner_enabled', {
      source_surface: 'ai_trading',
      enabled: overviewQuery.data.scanner.enabled,
      sentiment_mode: overviewQuery.data.scanner.sentiment.mode,
    });
    emitAnalyticsEvent('sentiment_settings_opened', {
      source_surface: 'ai_trading',
      sentiment_mode: overviewQuery.data.scanner.sentiment.mode,
      enabled: overviewQuery.data.scanner.sentiment.enabled,
    });
  }, [overviewQuery.data?.scanner]);

  const toggleScanner = () => {
    const scanner = overviewQuery.data?.scanner;
    if (!scanner) {
      return;
    }

    scannerMutation.mutate({
      ...scanner,
      enabled: !scanner.enabled,
    });
  };

  const watchOpportunity = (id: string) => {
    setWatchedIds((current) => {
      const next = current.includes(id) ? current.filter((currentId) => currentId !== id) : [id, ...current];
      saveStoredIds(WATCHED_STORAGE_KEY, next);
      return next;
    });
  };

  const dismissOpportunity = (id: string) => {
    setDismissedIds((current) => {
      const next = current.includes(id) ? current : [id, ...current];
      saveStoredIds(DISMISSED_STORAGE_KEY, next);
      return next;
    });
  };

  const resetDismissed = () => {
    setDismissedIds([]);
    saveStoredIds(DISMISSED_STORAGE_KEY, []);
  };

  const askAI = () => {
    const symbol = aiSymbol.trim().toUpperCase();
    if (!symbol || aiSuggestion.isPending) {
      return;
    }

    setAiSymbol(symbol);
    aiSuggestion.mutate({
      symbol,
      context: 'Beginner AI Trading page check. Explain the paper-trading idea, risk, and next safe action.',
    });
  };

  const isSimple = mode === 'simple';

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">AI Trading</p>
        <div className="max-w-3xl space-y-2">
          <h1 className="text-2xl font-bold md:text-3xl">{isSimple ? 'Find Trade Ideas' : 'AI Trading'}</h1>
          <p className="text-base text-muted-foreground">
            {isSimple
              ? 'Jax collects possible trade ideas here. Your job is to review each one, then send it to the safe next step.'
              : 'Review AI-backed Opportunities in one queue, then choose the right manual, approval, watch, or blocked-state path.'}
          </p>
        </div>
      </section>

      {isSimple && (
        <Card>
          <CardHeader>
            <CardTitle>Start Here</CardTitle>
            <CardDescription>Use this page as a triage desk. It does not place live trades.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm md:grid-cols-3">
            <div>
              <p className="font-semibold text-foreground">1. Read the idea</p>
              <p className="text-muted-foreground">Check the symbol, confidence, and why Jax found it.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">2. Choose the route</p>
              <p className="text-muted-foreground">Use Review order, Send to approval, Watch, or Dismiss.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">3. Keep it paper-safe</p>
              <p className="text-muted-foreground">Approval and broker confirmation stay separate from this screen.</p>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="h-4 w-4" />
              Ask Jax
            </CardTitle>
            <CardDescription>Check that the local AI is connected and get a paper-trading view of one symbol.</CardDescription>
          </div>
          <div className="flex w-full gap-2 md:w-auto">
            <input
              aria-label="AI symbol"
              className="min-h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground md:w-28"
              maxLength={12}
              onChange={(event) => setAiSymbol(event.target.value.toUpperCase())}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  askAI();
                }
              }}
              value={aiSymbol}
            />
            <Button disabled={aiSuggestion.isPending || !aiSymbol.trim()} onClick={askAI} type="button">
              {aiSuggestion.isPending ? 'Asking...' : 'Ask'}
            </Button>
          </div>
        </CardHeader>
        {(aiSuggestion.data || aiSuggestion.error || aiSuggestion.isPending) && (
          <CardContent className="space-y-3 text-sm">
            {aiSuggestion.isPending && <p className="text-muted-foreground">Jax is checking market data, recent news, and risk context.</p>}
            {aiSuggestion.error && <p className="text-destructive">AI request failed: {aiSuggestion.error.message}</p>}
            {aiSuggestion.data && (
              <div className="space-y-3">
                <div className="grid gap-3 md:grid-cols-[180px_1fr]">
                  <div className="rounded-md border border-border p-3">
                    <p className="text-xs uppercase text-muted-foreground">Suggestion</p>
                    <p className="mt-1 text-xl font-semibold">{aiSuggestion.data.suggestion.action}</p>
                    <p className="text-muted-foreground">
                      {aiSuggestion.data.suggestion.symbol} - {Math.round(aiSuggestion.data.suggestion.confidence * 100)}% confidence
                    </p>
                  </div>
                  <div className="rounded-md border border-border p-3">
                    <p className="font-semibold text-foreground">Reason</p>
                    <p className="mt-1 text-muted-foreground">{aiSuggestion.data.suggestion.reasoning || 'No reasoning returned.'}</p>
                    <p className="mt-3 font-semibold text-foreground">Risk</p>
                    <p className="mt-1 text-muted-foreground">{aiSuggestion.data.suggestion.risk_assessment || 'No risk summary returned.'}</p>
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <Button
                    disabled={!suggestionCanPromote || promoteSuggestionMutation.isPending}
                    onClick={() => promoteSuggestionMutation.mutate()}
                    type="button"
                  >
                    {promoteSuggestionMutation.isPending ? 'Sending...' : suggestionPromoteLabel}
                  </Button>
                  {!suggestionCanPromote && (
                    <p className="text-sm text-muted-foreground">Watch-only suggestions are advisory and cannot be sent to approval.</p>
                  )}
                  {promoteSuggestionMutation.data && (
                    <p className="text-sm text-muted-foreground">
                      Sent to {promoteSuggestionMutation.data.route === 'approval_required' ? 'Approvals' : 'Opportunities'} as candidate{' '}
                      {promoteSuggestionMutation.data.candidateId}.
                    </p>
                  )}
                  {promoteSuggestionMutation.error && (
                    <p className="text-sm text-destructive">Promotion failed: {promoteSuggestionMutation.error.message}</p>
                  )}
                </div>
              </div>
            )}
          </CardContent>
        )}
      </Card>

      <Card>
        <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            <CardTitle>News Monitor</CardTitle>
            <CardDescription>Shows whether Jax has received news from Jax World News Monitor.</CardDescription>
          </div>
          <Badge variant={monitorStatusQuery.data?.connected ? 'success' : monitorStatusQuery.isError ? 'destructive' : 'secondary'}>
            {monitorStatusQuery.data?.connected ? 'Receiving news' : monitorStatusQuery.isError ? 'Status unavailable' : 'No news yet'}
          </Badge>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm md:grid-cols-[1fr_180px]">
          <div>
            <p className="font-semibold text-foreground">
              {monitorStatusQuery.data?.lastHeadline ?? (monitorStatusQuery.isError ? 'Monitor status could not be loaded.' : 'No Monitor trigger received yet.')}
            </p>
            <p className="mt-1 text-muted-foreground">
              {monitorStatusQuery.data?.lastReceivedAt
                ? `Last received ${formatDateTime(monitorStatusQuery.data.lastReceivedAt)}; status ${monitorStatusQuery.data.lastStatus ?? 'unknown'}.`
                : 'Start Jax World News Monitor and post a trigger to populate this pipeline.'}
            </p>
            {monitorStatusQuery.data?.lastSymbols?.length ? (
              <p className="mt-1 text-muted-foreground">Mapped symbols: {monitorStatusQuery.data.lastSymbols.join(', ')}</p>
            ) : null}
            {monitorStatusQuery.data?.lastCandidateId ? (
              <p className="mt-1 text-muted-foreground">Latest candidate: {monitorStatusQuery.data.lastCandidateId}</p>
            ) : null}
          </div>
          <div className="rounded-md border border-border p-3">
            <p className="text-xs font-semibold uppercase text-muted-foreground">Inbox</p>
            <p className="mt-2 text-2xl font-semibold">{monitorStatusQuery.data?.counts.total ?? 0}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {monitorStatusQuery.data?.counts.pending ?? 0} pending / {monitorStatusQuery.data?.counts.candidatesCreated ?? 0} candidates
            </p>
          </div>
        </CardContent>
      </Card>

      <section className="grid gap-4 md:grid-cols-3" aria-label="Scanner state">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Bot className="h-4 w-4" />
              Scanner status
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{scannerStatus}</p>
            <p className="mt-1 text-sm text-muted-foreground">Signals, candidates, and approval queue data refresh every 30 seconds.</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Sparkles className="h-4 w-4" />
              Opportunities
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{Math.max(opportunities.length, opportunitiesCount)}</p>
            <p className="mt-1 text-sm text-muted-foreground">Unified across signals, candidates, approvals, and API overview counts.</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ShieldCheck className="h-4 w-4" />
              Guardrails
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">Route-aware</p>
            <p className="mt-1 text-sm text-muted-foreground">Blocked and approval-required items keep their policy path visible.</p>
          </CardContent>
        </Card>
      </section>

      {!isSimple && (
        <ScannerSettingsCard isSaving={scannerMutation.isPending} onToggleScanner={toggleScanner} settings={scannerSettings} />
      )}
      {isSimple && (
        <Card>
          <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
            <div>
              <CardTitle>Scanner</CardTitle>
              <CardDescription>Jax is watching {scannerSettings.symbols.join(', ')} for new ideas.</CardDescription>
            </div>
            <Button disabled={scannerMutation.isPending} onClick={toggleScanner} type="button" variant="outline">
              {scannerMutation.isPending ? 'Saving...' : scannerSettings.enabled ? 'Pause scanner' : 'Resume scanner'}
            </Button>
          </CardHeader>
        </Card>
      )}

      {staleData && (
        <div className="flex items-center gap-2 rounded-md border border-warning bg-warning/10 px-4 py-3 text-sm text-foreground">
          <Clock className="h-4 w-4" />
          Opportunity data is stale. Refresh before acting.
        </div>
      )}

      {isError && (
        <div className="flex items-center gap-2 rounded-md border border-destructive bg-destructive/10 px-4 py-3 text-sm text-foreground">
          <AlertTriangle className="h-4 w-4" />
          Failed to load one or more Opportunity sources. Available data is shown below.
        </div>
      )}

      <section className="space-y-3" aria-label="Opportunity queue">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-xl font-semibold">Opportunity queue</h2>
            <p className="text-sm text-muted-foreground">
              Every item includes a visible next action. {watchedCount > 0 ? `${watchedCount} currently watched.` : ''}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {dismissedIds.length > 0 && (
              <Button type="button" variant="ghost" onClick={resetDismissed}>
                Show dismissed
              </Button>
            )}
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                signalsQuery.refetch();
                candidatesQuery.refetch();
                approvalsQuery.refetch();
                overviewQuery.refetch();
                monitorStatusQuery.refetch();
              }}
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
          </div>
        </div>

        {isLoading && <p className="rounded-md border border-border p-6 text-muted-foreground">Loading Opportunity queue...</p>}

        {!isLoading && opportunities.length === 0 && (
          <Card>
            <CardContent className="py-10 text-center text-muted-foreground">
              No Opportunities are available right now. Scanner results will appear here when signals, candidates, or approvals are returned.
            </CardContent>
          </Card>
        )}

        {!isLoading && opportunities.map((opportunity) => (
          <OpportunityCard
            key={opportunity.id}
            opportunity={opportunity}
            watched={watchedIds.includes(opportunity.id)}
            onWatch={watchOpportunity}
            onDismiss={dismissOpportunity}
          />
        ))}
      </section>
    </div>
  );
}
