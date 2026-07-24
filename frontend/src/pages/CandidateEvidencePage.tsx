import { useMemo } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, BarChart3, ExternalLink, Newspaper, ShieldCheck, Sparkles } from 'lucide-react';
import { candidatesService, type CandidateTrade } from '@/data/approvals-service';
import type { SentimentEvidence } from '@/data/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { operatorEvidenceService, type OutcomeCheckpoint } from '@/data/operator-evidence-service';

type AnyRecord = Record<string, unknown>;

interface WorldMonitorEvidence {
  headline?: string;
  summary?: string;
  eventType?: string;
  source?: string;
  sourceEventId?: string;
  sourceUrls: string[];
  sourceCount?: number;
  assetThemes: string[];
  confidenceReasons: string[];
  mappingReason?: string;
  route?: string;
}

interface ChartEvidence {
  confirmed?: boolean;
  reasonCode?: string;
  reason?: string;
  candleCount?: number;
  lastClose?: number;
  sma20?: number;
  fiveCandleChangePct?: number;
  checkedAt?: string;
}

interface SuggestedPaperSize {
  shares: number;
  notional: number;
  riskToStop: number;
  rewardToTarget?: number;
  riskReward?: number;
  source: string;
}

interface CandidateEvidence {
  worldMonitor: WorldMonitorEvidence;
  chart?: ChartEvidence;
  sentiment?: SentimentEvidence;
  suggestedSize?: SuggestedPaperSize;
}

function asRecord(value: unknown): AnyRecord | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as AnyRecord)
    : undefined;
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() !== '' ? value : undefined;
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
}

function readString(record: AnyRecord | undefined, keys: string[]): string | undefined {
  if (!record) return undefined;
  for (const key of keys) {
    const value = asString(record[key]);
    if (value) return value;
  }
  return undefined;
}

function readNumber(record: AnyRecord | undefined, keys: string[]): number | undefined {
  if (!record) return undefined;
  for (const key of keys) {
    const value = asNumber(record[key]);
    if (value !== undefined) return value;
  }
  return undefined;
}

function formatCurrency(value: number | undefined): string {
  if (value === undefined) return 'Unavailable';
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value);
}

function formatPercent(value: number | undefined): string {
  if (value === undefined) return 'Unavailable';
  const percent = Math.abs(value) <= 1 ? value * 100 : value;
  return `${percent.toFixed(2)}%`;
}

function formatDateTime(value: string | undefined): string {
  if (!value) return 'Unavailable';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toLocaleString();
}

function confidenceLabel(confidence: number | undefined): string {
  if (confidence === undefined) return 'Unknown confidence';
  const percent = confidence <= 1 ? confidence * 100 : confidence;
  if (percent >= 80) return 'High confidence';
  if (percent >= 60) return 'Medium confidence';
  return 'Low confidence';
}

function getWorldMonitorEvidence(metadata: AnyRecord | undefined): WorldMonitorEvidence {
  const worldMonitor = asRecord(metadata?.worldMonitor) ?? metadata;
  return {
    headline: readString(worldMonitor, ['headline', 'newsHeadline', 'eventTitle']),
    summary: readString(worldMonitor, ['summary', 'newsSummary', 'description']),
    eventType: readString(worldMonitor, ['eventType', 'event_type']),
    source: readString(worldMonitor, ['source', 'newsSource']),
    sourceEventId: readString(worldMonitor, [
      'sourceEventId',
      'source_event_id',
      'worldMonitorEventId',
    ]),
    sourceUrls: asStringArray(
      worldMonitor?.sourceURLs ?? worldMonitor?.sourceUrls ?? worldMonitor?.source_urls,
    ),
    sourceCount: readNumber(worldMonitor, ['sourceCount', 'source_count']),
    assetThemes: asStringArray(worldMonitor?.assetThemes ?? worldMonitor?.asset_themes),
    confidenceReasons: asStringArray(
      worldMonitor?.confidenceReasons ?? worldMonitor?.confidence_reasons,
    ),
    mappingReason: readString(worldMonitor, ['mappingReason', 'mapping_reason']),
    route: readString(worldMonitor, ['route']),
  };
}

function getChartEvidence(metadata: AnyRecord | undefined): ChartEvidence | undefined {
  const chart = asRecord(
    metadata?.chartConfirmation ?? metadata?.chart_confirmation ?? metadata?.chart,
  );
  if (!chart) return undefined;
  return {
    confirmed: asBoolean(chart.confirmed),
    reasonCode: readString(chart, ['reasonCode', 'reason_code']),
    reason: readString(chart, ['reason']),
    candleCount: readNumber(chart, ['candleCount', 'candle_count']),
    lastClose: readNumber(chart, ['lastClose', 'last_close']),
    sma20: readNumber(chart, ['sma20', 'SMA20']),
    fiveCandleChangePct: readNumber(chart, ['fiveCandleChangePct', 'five_candle_change_pct']),
    checkedAt: readString(chart, ['checkedAt', 'checked_at']),
  };
}

function getSentimentEvidence(
  candidate: CandidateTrade,
  metadata: AnyRecord | undefined,
): SentimentEvidence | undefined {
  if (candidate.sentiment) return candidate.sentiment;
  const sentiment = asRecord(
    metadata?.sentiment ??
      metadata?.sentimentSummaryStructured ??
      metadata?.sentiment_summary_structured,
  );
  if (!sentiment) return undefined;
  return {
    label: (asString(sentiment.label) as SentimentEvidence['label']) ?? 'unavailable',
    state: (asString(sentiment.state) as SentimentEvidence['state']) ?? 'available',
    score: asNumber(sentiment.score),
    confidence: asNumber(sentiment.confidence),
    window: asString(sentiment.window),
    sourceCount: readNumber(sentiment, ['sourceCount', 'source_count']),
    sourceGroups: asRecord(
      sentiment.sourceGroups ?? sentiment.source_groups,
    ) as SentimentEvidence['sourceGroups'],
    priceAgreement: sentiment.priceAgreement as SentimentEvidence['priceAgreement'],
    topDrivers: asStringArray(sentiment.topDrivers ?? sentiment.top_drivers),
    limitations: asStringArray(sentiment.limitations),
    summary: asString(sentiment.summary),
    snapshotAt: readString(sentiment, ['snapshotAt', 'snapshot_at']),
    intendedUse: readString(sentiment, ['intendedUse', 'intended_use']),
  };
}

function getSuggestedPaperSize(
  candidate: CandidateTrade,
  metadata: AnyRecord | undefined,
): SuggestedPaperSize | undefined {
  const sizing = asRecord(
    metadata?.sizing ?? metadata?.suggestedSizing ?? metadata?.positionSizing,
  );
  const explicitShares = readNumber(sizing, ['shares', 'quantity', 'suggestedQuantity']);
  const entry = candidate.entryPrice;

  if (explicitShares && entry && explicitShares > 0 && entry > 0) {
    const stop = candidate.stopLoss;
    const target = candidate.takeProfit;
    const riskPerShare = stop ? Math.abs(entry - stop) : 0;
    const rewardPerShare = target ? Math.abs(target - entry) : undefined;
    return {
      shares: Math.floor(explicitShares),
      notional: Math.floor(explicitShares) * entry,
      riskToStop: riskPerShare * Math.floor(explicitShares),
      rewardToTarget: rewardPerShare ? rewardPerShare * Math.floor(explicitShares) : undefined,
      riskReward: rewardPerShare && riskPerShare > 0 ? rewardPerShare / riskPerShare : undefined,
      source: 'Attached sizing metadata',
    };
  }

  return undefined;
}

function buildEvidence(candidate: CandidateTrade): CandidateEvidence {
  const metadata = candidate.metadata;
  return {
    worldMonitor: getWorldMonitorEvidence(metadata),
    chart: getChartEvidence(metadata),
    sentiment: getSentimentEvidence(candidate, metadata),
    suggestedSize: getSuggestedPaperSize(candidate, metadata),
  };
}

function EvidenceList({ items, emptyText }: { items: string[]; emptyText: string }) {
  if (!items.length) return <p className="text-sm text-muted-foreground">{emptyText}</p>;
  return (
    <ul className="space-y-2 text-sm text-foreground">
      {items.map((item) => (
        <li key={item} className="rounded-md border border-border bg-muted/30 px-3 py-2">
          {item}
        </li>
      ))}
    </ul>
  );
}

export function CandidateEvidencePage() {
  const { candidateId } = useParams<{ candidateId: string }>();

  const candidateQuery = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => candidatesService.get(candidateId ?? ''),
    enabled: Boolean(candidateId),
  });
  const operatorQuery = useQuery({
    queryKey: ['operator-candidate-evidence', candidateId],
    queryFn: () => operatorEvidenceService.candidate(candidateId ?? ''),
    enabled: Boolean(candidateId),
  });

  const evidence = useMemo(
    () => (candidateQuery.data ? buildEvidence(candidateQuery.data) : null),
    [candidateQuery.data],
  );

  if (!candidateId) {
    return <div className="p-6">Missing candidate id.</div>;
  }

  if (candidateQuery.isLoading) {
    return <div className="p-6">Loading candidate evidence...</div>;
  }

  if (candidateQuery.isError || !candidateQuery.data || !evidence) {
    return (
      <div className="space-y-4 p-6">
        <p className="text-sm text-destructive">Failed to load candidate evidence.</p>
        <Button asChild variant="outline">
          <Link to="/etf/approvals">
            <ArrowLeft className="mr-2 h-4 w-4" /> Back to Candidates
          </Link>
        </Button>
      </div>
    );
  }

  const candidate = candidateQuery.data;
  const sourceCount = evidence.worldMonitor.sourceCount ?? evidence.worldMonitor.sourceUrls.length;
  const sentimentSources = evidence.sentiment?.sourceCount ?? sourceCount;

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Button asChild variant="outline" size="sm">
          <Link to="/etf/approvals">
            <ArrowLeft className="mr-2 h-4 w-4" /> Back to Candidates
          </Link>
        </Button>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline" size="sm">
            <Link to="/monitor/inbox">Evidence Inbox</Link>
          </Button>
          <Button asChild variant="outline" size="sm">
            <Link to="/outcomes">Hypothetical Outcomes</Link>
          </Button>
        </div>
      </div>

      <header className="space-y-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant="secondary">{candidate.strategyId || candidate.signalType}</Badge>
          <Badge variant={candidate.status === 'blocked' ? 'destructive' : 'outline'}>
            {humanize(candidate.status)}
          </Badge>
          <Badge variant="outline">{confidenceLabel(candidate.confidence)}</Badge>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {candidate.symbol} candidate
          </p>
          <h1 className="text-3xl font-bold">Candidate Review</h1>
          <p className="mt-2 max-w-4xl text-base text-muted-foreground">
            Review why Jax created this candidate, what a human decided and what the hypothetical
            paper plan assumed.
          </p>
          <p className="mt-3 rounded-md border border-primary/40 bg-primary/5 p-3 text-sm font-semibold">
            Review only — this page cannot place an order or create a fill.
          </p>
        </div>
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5" /> Plain-English overview
          </CardTitle>
          <CardDescription>
            Start here to understand why this candidate exists and what happened next.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 lg:grid-cols-[2fr_1fr]">
          <div className="space-y-3">
            <p className="text-sm leading-6 text-foreground">
              {candidate.reasoning ||
                evidence.worldMonitor.summary ||
                'No plain-language reason was attached to this candidate.'}
            </p>
            {evidence.worldMonitor.mappingReason && (
              <div className="rounded-md border border-border bg-muted/30 p-3 text-sm">
                <span className="font-semibold">Why this symbol: </span>
                {evidence.worldMonitor.mappingReason}
              </div>
            )}
            {candidate.blockReason && (
              <div className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
                <span className="font-semibold">Current blocker: </span>
                {candidate.blockReason}
              </div>
            )}
          </div>
          <dl className="grid grid-cols-2 gap-3 text-sm">
            <div>
              <dt className="text-muted-foreground">Candidate created</dt>
              <dd className="font-semibold">{formatDateTime(candidate.detectedAt)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Lifecycle state</dt>
              <dd className="font-semibold">{humanize(candidate.status)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Setup type</dt>
              <dd className="font-semibold">{candidate.strategyId || candidate.signalType}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Expires</dt>
              <dd className="font-semibold">{formatDateTime(candidate.expiresAt)}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <div className="grid gap-5 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Newspaper className="h-5 w-5" /> Why Jax considered it
            </CardTitle>
            <CardDescription>
              {sourceCount > 0
                ? `${sourceCount} monitor source${sourceCount === 1 ? '' : 's'} attached.`
                : 'No source links attached.'}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="font-semibold">
                {evidence.worldMonitor.headline || 'No monitor headline attached'}
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                {evidence.worldMonitor.summary || 'No monitor summary attached.'}
              </p>
            </div>
            {evidence.worldMonitor.sourceUrls.length > 0 ? (
              <ul className="space-y-2 text-sm">
                {evidence.worldMonitor.sourceUrls.map((url) => (
                  <li key={url}>
                    <a
                      href={url}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex max-w-full items-center gap-2 break-all text-primary underline-offset-4 hover:underline"
                    >
                      <ExternalLink className="h-4 w-4 shrink-0" />
                      {url}
                    </a>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
                No article URLs were supplied with this candidate. Treat the evidence as incomplete
                until the Monitor payload includes source links.
              </p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5" /> What the charts are saying
            </CardTitle>
            <CardDescription>
              Persisted chart evidence is one review input; absence never implies a pass.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {evidence.chart ? (
              <>
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={evidence.chart.confirmed ? 'success' : 'warning'}>
                    {evidence.chart.confirmed ? 'Chart confirmed' : 'Chart not confirmed'}
                  </Badge>
                  {evidence.chart.reasonCode && (
                    <Badge variant="outline">{evidence.chart.reasonCode}</Badge>
                  )}
                </div>
                <p className="text-sm text-foreground">
                  {evidence.chart.reason || 'No chart explanation was attached.'}
                </p>
                <dl className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <dt className="text-muted-foreground">Last close</dt>
                    <dd className="font-semibold">{formatCurrency(evidence.chart.lastClose)}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">SMA 20</dt>
                    <dd className="font-semibold">{formatCurrency(evidence.chart.sma20)}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">5-candle move</dt>
                    <dd className="font-semibold">
                      {formatPercent(evidence.chart.fiveCandleChangePct)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Candles checked</dt>
                    <dd className="font-semibold">{evidence.chart.candleCount ?? 'Unavailable'}</dd>
                  </div>
                </dl>
                <p className="text-xs text-muted-foreground">
                  Checked {formatDateTime(evidence.chart.checkedAt)}
                </p>
              </>
            ) : (
              <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
                Chart confirmation was not available.
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-5 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Sentiment and source</CardTitle>
            <CardDescription>
              Shows whether the news tone supports, weakens, or conflicts with the setup.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {evidence.sentiment ? (
              <>
                <div className="flex flex-wrap gap-2">
                  <Badge variant="outline">{evidence.sentiment.label}</Badge>
                  <Badge variant="outline">{evidence.sentiment.state}</Badge>
                  <Badge variant="outline">{sentimentSources} sources</Badge>
                </div>
                <p className="text-sm text-foreground">
                  {evidence.sentiment.summary || 'No sentiment summary was attached.'}
                </p>
                <dl className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <dt className="text-muted-foreground">Score</dt>
                    <dd className="font-semibold">{formatPercent(evidence.sentiment.score)}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Confidence</dt>
                    <dd className="font-semibold">
                      {formatPercent(evidence.sentiment.confidence)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Window</dt>
                    <dd className="font-semibold">{evidence.sentiment.window || 'Unavailable'}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Price agreement</dt>
                    <dd className="font-semibold">
                      {evidence.sentiment.priceAgreement || 'Unavailable'}
                    </dd>
                  </div>
                </dl>
                <EvidenceList
                  items={evidence.sentiment.topDrivers ?? []}
                  emptyText="No sentiment drivers attached."
                />
                <EvidenceList
                  items={evidence.sentiment.limitations ?? []}
                  emptyText="No sentiment limitations attached."
                />
              </>
            ) : (
              <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
                No structured sentiment evidence is attached to this candidate yet.
              </p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Persisted paper sizing evidence</CardTitle>
            <CardDescription>
              Financial values are shown only when attached to persisted sizing evidence.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {evidence.suggestedSize ? (
              <>
                <div className="rounded-md border border-border bg-muted/30 p-4">
                  <p className="text-2xl font-bold">{evidence.suggestedSize.shares} shares</p>
                  <p className="text-sm text-muted-foreground">
                    {formatCurrency(evidence.suggestedSize.notional)} estimated notional
                  </p>
                </div>
                <dl className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <dt className="text-muted-foreground">Risk to stop</dt>
                    <dd className="font-semibold">
                      {formatCurrency(evidence.suggestedSize.riskToStop)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Reward to target</dt>
                    <dd className="font-semibold">
                      {formatCurrency(evidence.suggestedSize.rewardToTarget)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Risk/reward</dt>
                    <dd className="font-semibold">
                      {evidence.suggestedSize.riskReward
                        ? `${evidence.suggestedSize.riskReward.toFixed(2)}R`
                        : 'Unavailable'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Sizing source</dt>
                    <dd className="font-semibold">{evidence.suggestedSize.source}</dd>
                  </div>
                </dl>
              </>
            ) : (
              <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
                No persisted sizing evidence available
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      {operatorQuery.isError && (
        <p role="alert" className="text-destructive">
          Jax could not load this evidence. Your data has not been changed.
        </p>
      )}
      {operatorQuery.data && (
        <PersistedJourney evidence={operatorQuery.data} candidate={candidate} />
      )}

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" /> Quick review checklist
          </CardTitle>
          <CardDescription>
            Read-only evidence review. No action on this page changes runtime or trading state.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 text-sm md:grid-cols-3">
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <p className="font-semibold">1. Evidence</p>
              <p className="text-muted-foreground">
                Headline, source links, sentiment, and chart check are present and credible.
              </p>
            </div>
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <p className="font-semibold">2. Risk</p>
              <p className="text-muted-foreground">
                Review only the persisted paper-plan assumptions.
              </p>
            </div>
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <p className="font-semibold">3. Execution</p>
              <p className="text-muted-foreground">
                No fill occurred. Hypothetical paper results are not orders, positions, or realised
                P&amp;L.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function PersistedJourney({
  evidence,
  candidate,
}: {
  evidence: Awaited<ReturnType<typeof operatorEvidenceService.candidate>>;
  candidate: CandidateTrade;
}) {
  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>What supported or blocked it</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Fact
            label="Evidence score"
            value={
              evidence.evidenceScore === undefined
                ? 'Missing persisted evidence'
                : `${(evidence.evidenceScore * 100).toFixed(1)}%`
            }
          />
          <Fact
            label="Evidence state"
            value={
              evidence.evidenceStatus
                ? humanize(evidence.evidenceStatus)
                : 'Missing — no pass inferred'
            }
          />
          <Fact
            label="Trust checks"
            value={
              evidence.gateStatus ? humanize(evidence.gateStatus) : 'Missing — no pass inferred'
            }
          />
          <Fact
            label="Risk review"
            value={
              evidence.riskStatus ? humanize(evidence.riskStatus) : 'Missing — no pass inferred'
            }
          />
          <Fact
            label="Human decision"
            value={
              evidence.decisionProvenance === 'non_human'
                ? 'Historical non-human record'
                : evidence.approvalDecision
                  ? `Human ${humanize(evidence.approvalDecision)}`
                  : 'No human decision'
            }
          />
          <Fact label="Approver" value={evidence.approvedBy || 'Not applicable'} />
          <Fact label="Decision reason" value={evidence.approvalReason || 'Not supplied'} />
          <Fact label="Decision timestamp" value={formatDateTime(evidence.approvalAt)} />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Hypothetical paper plan</CardTitle>
          <CardDescription>
            HYPOTHETICAL PAPER PLAN — NOT AN ORDER OR FILL. Every financial value below is a
            persisted paper-plan assumption.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Fact
            label="Paper-ticket ID"
            value={evidence.paperTicketId || 'No paper plan was created for this candidate'}
          />
          <Fact
            label="Paper-plan state"
            value={
              evidence.paperTicketStatus ? humanize(evidence.paperTicketStatus) : 'Not applicable'
            }
          />
          <Fact
            label="Hypothetical entry — persisted assumption"
            value={formatCurrency(evidence.entry)}
          />
          <Fact label="Stop — persisted assumption" value={formatCurrency(evidence.stop)} />
          <Fact label="Target — persisted assumption" value={formatCurrency(evidence.target)} />
          <Fact
            label="Quantity — persisted assumption"
            value={
              evidence.quantity === undefined
                ? 'No persisted sizing evidence available'
                : evidence.quantity.toString()
            }
          />
          <Fact label="Notional — persisted assumption" value={formatCurrency(evidence.notional)} />
          <Fact
            label="Planned risk — persisted assumption"
            value={formatCurrency(evidence.plannedRisk)}
          />
          <Fact label="Planned reward" value={formatCurrency(evidence.plannedReward)} />
          <Fact
            label="Reward/risk"
            value={
              evidence.rewardRisk === undefined
                ? 'Not supplied'
                : `${evidence.rewardRisk.toFixed(2)}R`
            }
          />
          <Fact
            label="Leverage"
            value={
              evidence.leverage === undefined ? 'Not supplied' : `${evidence.leverage.toFixed(2)}x`
            }
          />
          <Fact
            label="Account-equity assumption"
            value={formatCurrency(evidence.accountEquityAssumption)}
          />
          <Fact label="Fill state" value="NO FILL OCCURRED" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Hypothetical outcome checkpoints</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {evidence.checkpoints.length ? (
            evidence.checkpoints.map((c) => <Checkpoint key={c.name} checkpoint={c} />)
          ) : (
            <p className="text-sm text-muted-foreground">No outcome checkpoints persisted.</p>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>No-fill evidence</CardTitle>
          <CardDescription>
            Selected journey counts are separate from unrelated historical database records.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="font-semibold">
            No fill occurred. This candidate journey created no execution records.
          </p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            {Object.entries(evidence.selectedExecutionCounts).map(([key, value]) => (
              <Fact key={key} label={humanize(key)} value={String(value)} />
            ))}
          </div>
          <p className="text-sm text-muted-foreground">
            Historical database records exist but are unrelated to this candidate:{' '}
            {Object.entries(evidence.historicalExecutionCounts)
              .map(([key, value]) => `${humanize(key)} ${value}`)
              .join(', ')}
            .
          </p>
        </CardContent>
      </Card>
      <details className="rounded-lg border bg-card">
        <summary className="cursor-pointer px-4 py-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          Audit details
        </summary>
        <div className="space-y-3 border-t p-4 text-sm">
          <Fact label="Candidate ID" value={candidate.id} />
          <Fact label="Approval ID" value={evidence.approvalId || 'Not applicable'} />
          <Fact label="Paper-ticket ID" value={evidence.paperTicketId || 'Not applicable'} />
          <Fact label="Original candidate status" value={candidate.status} />
          <pre className="max-w-full overflow-auto whitespace-pre-wrap break-all rounded bg-muted p-3">
            {JSON.stringify(candidate.metadata ?? {}, null, 2)}
          </pre>
        </div>
      </details>
    </div>
  );
}

function Checkpoint({ checkpoint: c }: { checkpoint: OutcomeCheckpoint }) {
  const label =
    c.status === 'pending_not_due'
      ? 'PENDING — NOT DUE'
      : c.status === 'ambiguous_same_candle'
        ? 'AMBIGUOUS SAME CANDLE'
        : ['pending_market_data', 'insufficient_data'].includes(c.status)
          ? 'MISSING MARKET DATA'
          : 'COMPLETED';
  const facts = [
    ['Tracking start', formatDateTime(c.trackingStartedAt)],
    ['Due', formatDateTime(c.dueAt)],
    ['Status', label],
    ['Checkpoint price', formatCurrency(c.checkpointPrice)],
    ['Observation', formatDateTime(c.observationAt)],
    ['Market-data source', c.marketDataSource || 'Not supplied'],
    ['Hypothetical return', formatPercent(c.percentageReturn)],
    ['Hypothetical P&L', formatCurrency(c.hypotheticalPnl)],
    ['Maximum favourable excursion', formatCurrency(c.maximumFavourableExcursion)],
    ['Maximum adverse excursion', formatCurrency(c.maximumAdverseExcursion)],
    ['Stop touched', c.stopTouched ? 'Yes' : 'No'],
    ['Target touched', c.targetTouched ? 'Yes' : 'No'],
    ['First stop touch', formatDateTime(c.firstStopTouchAt)],
    ['First target touch', formatDateTime(c.firstTargetTouchAt)],
    ['Created', formatDateTime(c.createdAt)],
    ['Updated', formatDateTime(c.updatedAt)],
  ];
  return (
    <div className="rounded-md border p-4">
      <div className="mb-3 flex flex-wrap gap-2">
        <Badge variant="outline">{c.name}</Badge>
        <Badge variant={label === 'COMPLETED' ? 'success' : 'warning'}>{label}</Badge>
        <Badge variant="secondary">HYPOTHETICAL — NOT A FILL</Badge>
      </div>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {facts.map(([l, v]) => (
          <Fact key={l} label={l} value={v} />
        ))}
      </div>
    </div>
  );
}
function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border bg-muted/20 p-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="break-words text-sm font-semibold">{value}</div>
    </div>
  );
}
function humanize(value: string) {
  return value
    .replace(/_/g, ' ')
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, (c) => c.toUpperCase());
}
