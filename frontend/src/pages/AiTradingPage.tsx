import { useEffect, useMemo, useRef } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, ArrowRight, Bot, Clock, RefreshCw, ShieldCheck, Sparkles } from 'lucide-react';
import { approvalsService, candidatesService } from '@/data/approvals-service';
import { signalsService } from '@/data/signals-service';
import { toOpportunitySummaries } from '@/data/opportunity-adapter';
import type { OpportunityRoute, OpportunitySummary, ScannerSettings } from '@/data/types';
import { ScannerSettingsCard } from '@/components/trading/ScannerSettingsCard';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { emitAnalyticsEvent } from '@/lib/analytics';

const REFRESH_INTERVAL_MS = 30_000;
const STALE_DATA_MS = 5 * 60_000;

const phaseOneScannerSettings: ScannerSettings = {
  enabled: true,
  assetScope: 'ETF pilot',
  symbols: ['SPY', 'QQQ', 'IWM'],
  universePreset: 'Phase 1 ETF universe',
  intervalSeconds: 30,
  minimumConfidence: 0.65,
  connected: false,
  sentiment: {
    enabled: true,
    sourceScope: 'Trusted news and filings',
    timeWindow: 'Last 24 hours',
    minimumThresholdLabel: 'Positive or better',
    minimumSourceCount: 2,
    sourceTrustMode: 'trust_weighted',
    mode: 'rank_boost',
    supported: false,
    connected: false,
    unsupportedReason:
      'Sentiment is shown as boost/filter guidance in Phase 1. Required-feature routing needs Phase 2 backend support.',
  },
};

const routeLabels: Record<OpportunityRoute, string> = {
  manual_allowed: 'Manual review',
  approval_required: 'Approval required',
  blocked: 'Blocked',
};

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
      label: 'Open blocked-state guidance',
      path:
        opportunity.sourceType === 'candidate' || opportunity.sourceType === 'approval'
          ? `/candidates/${opportunity.sourceId}/evidence`
          : '/research',
    };
  }

  if (opportunity.route === 'approval_required') {
    return { label: 'Send to approval', path: '/etf/approvals' };
  }

  return { label: 'Review order', path: '/manual-trading' };
}

function OpportunityCard({ opportunity }: { opportunity: OpportunitySummary }) {
  const action = primaryAction(opportunity);
  const expired = isExpired(opportunity.expiresAt);

  useEffect(() => {
    if (!opportunity.sentimentSummary) {
      return;
    }

    emitAnalyticsEvent('opportunity_sentiment_viewed', {
      source_surface: 'ai_trading',
      opportunity_id: opportunity.id,
      route_type: opportunity.route,
      sentiment_mode: phaseOneScannerSettings.sentiment.mode,
    });
  }, [opportunity.id, opportunity.route, opportunity.sentimentSummary]);

  return (
    <Card className="overflow-hidden">
      <CardHeader className="gap-3 md:flex-row md:items-start md:justify-between">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <CardTitle className="text-xl">{opportunity.symbol}</CardTitle>
            <Badge variant={opportunity.route === 'blocked' ? 'destructive' : opportunity.route === 'approval_required' ? 'warning' : 'default'}>
              {routeLabels[opportunity.route]}
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

        {opportunity.sentimentSummary && (
          <div className="rounded-md border border-border bg-muted px-3 py-2 text-sm text-muted-foreground">
            {opportunity.sentimentSummary}
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Button asChild>
            <Link to={action.path}>
              {action.label}
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
          <Button type="button" variant="outline">
            Watch
          </Button>
          <Button type="button" variant="ghost">
            Dismiss
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

export function AiTradingPage() {
  const trackedSetupRef = useRef(false);

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

  const opportunities = useMemo(
    () =>
      toOpportunitySummaries({
        signals: signalsQuery.data?.signals ?? [],
        candidates: candidatesQuery.data ?? [],
        approvals: approvalsQuery.data ?? [],
      }),
    [approvalsQuery.data, candidatesQuery.data, signalsQuery.data]
  );

  const queries = [signalsQuery, candidatesQuery, approvalsQuery];
  const isLoading = queries.some((query) => query.isPending);
  const isError = queries.some((query) => query.isError);
  const isFetching = queries.some((query) => query.isFetching);
  const latestUpdate = Math.max(...queries.map((query) => query.dataUpdatedAt).filter(Boolean), 0);
  const staleData = latestUpdate > 0 && Date.now() - latestUpdate > STALE_DATA_MS;
  const scannerStatus = isError ? 'Degraded' : isFetching ? 'Scanning' : 'Ready';

  useEffect(() => {
    emitAnalyticsEvent('page_viewed', { source_surface: 'ai_trading' });
  }, []);

  useEffect(() => {
    if (trackedSetupRef.current) {
      return;
    }

    trackedSetupRef.current = true;
    emitAnalyticsEvent('ai_scanner_enabled', {
      source_surface: 'ai_trading',
      enabled: phaseOneScannerSettings.enabled,
      sentiment_mode: phaseOneScannerSettings.sentiment.mode,
    });
    emitAnalyticsEvent('sentiment_settings_opened', {
      source_surface: 'ai_trading',
      sentiment_mode: phaseOneScannerSettings.sentiment.mode,
      enabled: phaseOneScannerSettings.sentiment.enabled,
    });
  }, []);

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">
      <section className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">AI Trading</p>
        <div className="max-w-3xl space-y-2">
          <h1 className="text-2xl font-bold md:text-3xl">AI Trading</h1>
          <p className="text-base text-muted-foreground">
            Review AI-backed Opportunities in one queue, then choose the right manual, approval, watch, or blocked-state path.
          </p>
        </div>
      </section>

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
            <p className="text-2xl font-semibold">{opportunities.length}</p>
            <p className="mt-1 text-sm text-muted-foreground">Unified across signals, candidates, and approvals.</p>
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

      <ScannerSettingsCard settings={phaseOneScannerSettings} />

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
            <p className="text-sm text-muted-foreground">Every item includes a visible next action.</p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              signalsQuery.refetch();
              candidatesQuery.refetch();
              approvalsQuery.refetch();
            }}
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
        </div>

        {isLoading && <p className="rounded-md border border-border p-6 text-muted-foreground">Loading Opportunity queue...</p>}

        {!isLoading && opportunities.length === 0 && (
          <Card>
            <CardContent className="py-10 text-center text-muted-foreground">
              No Opportunities are available right now. Scanner results will appear here when signals, candidates, or approvals are returned.
            </CardContent>
          </Card>
        )}

        {!isLoading && opportunities.map((opportunity) => <OpportunityCard key={opportunity.id} opportunity={opportunity} />)}
      </section>
    </div>
  );
}
