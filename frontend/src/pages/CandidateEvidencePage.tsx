import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, ExternalLink } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';
import { candidatesService, type CandidateTrade } from '@/data/approvals-service';
import {
  operatorEvidenceService,
  type OperatorCandidateEvidence,
} from '@/data/operator-evidence-service';
import type { SentimentEvidence } from '@/data/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

type AnyRecord = Record<string, unknown>;

interface EvidenceView {
  headline?: string;
  summary?: string;
  source?: string;
  sourceEventId?: string;
  sourceUrls: string[];
  assetThemes: string[];
  mappingReason?: string;
  chart?: {
    confirmed?: boolean;
    reason?: string;
    lastClose?: number;
    sma20?: number;
    fiveCandleChangePct?: number;
  };
  sentiment?: SentimentEvidence;
}

function asRecord(value: unknown): AnyRecord | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as AnyRecord)
    : undefined;
}

function asString(value: unknown) {
  return typeof value === 'string' && value.trim() ? value : undefined;
}

function asNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

function asBoolean(value: unknown) {
  return typeof value === 'boolean' ? value : undefined;
}

function asStrings(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string' && Boolean(item.trim()))
    : [];
}

function firstString(record: AnyRecord | undefined, keys: string[]) {
  for (const key of keys) {
    const value = asString(record?.[key]);
    if (value) return value;
  }
  return undefined;
}

function firstNumber(record: AnyRecord | undefined, keys: string[]) {
  for (const key of keys) {
    const value = asNumber(record?.[key]);
    if (value !== undefined) return value;
  }
  return undefined;
}

function buildEvidence(candidate: CandidateTrade): EvidenceView {
  const metadata = candidate.metadata;
  const world = asRecord(metadata?.worldMonitor) ?? metadata;
  const chart = asRecord(
    metadata?.chartConfirmation ?? metadata?.chart_confirmation ?? metadata?.chart,
  );
  const rawSentiment = asRecord(metadata?.sentiment ?? metadata?.sentimentSummaryStructured);
  const sentiment =
    candidate.sentiment ??
    (rawSentiment
      ? {
          label: (asString(rawSentiment.label) as SentimentEvidence['label']) ?? 'unavailable',
          state: (asString(rawSentiment.state) as SentimentEvidence['state']) ?? 'available',
          score: asNumber(rawSentiment.score),
          confidence: asNumber(rawSentiment.confidence),
          summary: asString(rawSentiment.summary),
          limitations: asStrings(rawSentiment.limitations),
          topDrivers: asStrings(rawSentiment.topDrivers ?? rawSentiment.top_drivers),
        }
      : undefined);

  return {
    headline: firstString(world, ['headline', 'newsHeadline', 'eventTitle']),
    summary: firstString(world, ['summary', 'newsSummary', 'description']),
    source: firstString(world, ['source', 'newsSource']),
    sourceEventId: firstString(world, ['sourceEventId', 'source_event_id', 'worldMonitorEventId']),
    sourceUrls: asStrings(world?.sourceURLs ?? world?.sourceUrls ?? world?.source_urls),
    assetThemes: asStrings(world?.assetThemes ?? world?.asset_themes),
    mappingReason: firstString(world, ['mappingReason', 'mapping_reason']),
    chart: chart
      ? {
          confirmed: asBoolean(chart.confirmed),
          reason: asString(chart.reason),
          lastClose: firstNumber(chart, ['lastClose', 'last_close']),
          sma20: firstNumber(chart, ['sma20', 'SMA20']),
          fiveCandleChangePct: firstNumber(chart, [
            'fiveCandleChangePct',
            'five_candle_change_pct',
          ]),
        }
      : undefined,
    sentiment,
  };
}

function money(value: number | undefined) {
  return value === undefined
    ? 'Unavailable'
    : new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value);
}

function percent(value: number | undefined) {
  if (value === undefined) return 'Unavailable';
  return `${(Math.abs(value) <= 1 ? value * 100 : value).toFixed(2)}%`;
}

function dateTime(value: string | undefined) {
  if (!value) return 'Unavailable';
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
}

function humanize(value: string | undefined) {
  if (!value) return 'Not recorded';
  return value
    .replace(/_/g, ' ')
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, (c) => c.toUpperCase());
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded border bg-muted/20 p-2">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="break-words text-sm font-semibold">{value}</dd>
    </div>
  );
}

function Overview({
  candidate,
  journey,
  evidence,
}: {
  candidate: CandidateTrade;
  journey?: OperatorCandidateEvidence;
  evidence: EvidenceView;
}) {
  const completed = journey?.checkpoints.filter((item) => item.status === 'completed').length ?? 0;
  const next =
    completed > 0
      ? `${completed} hypothetical checkpoint${completed === 1 ? '' : 's'} completed.`
      : journey?.checkpoints.length
        ? 'Hypothetical checkpoints are pending or missing data.'
        : 'No hypothetical outcomes are available yet.';
  const decision =
    journey?.decisionProvenance === 'non_human'
      ? 'Historical non-human record'
      : journey?.approvalDecision
        ? `Human ${humanize(journey.approvalDecision).toLowerCase()}`
        : 'No human decision';

  return (
    <Card>
      <CardHeader>
        <CardTitle>Overview</CardTitle>
        <CardDescription>Why Jax created this candidate and what happened next.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <Fact label="Symbol" value={candidate.symbol} />
          <Fact label="Setup type" value={candidate.strategyId || candidate.signalType} />
          <Fact label="Lifecycle state" value={humanize(candidate.status)} />
          <Fact label="Human-decision state" value={decision} />
          <Fact label="Paper-plan state" value={humanize(journey?.paperTicketStatus)} />
          <Fact label="Outcome availability" value={next} />
          <Fact label="Created" value={dateTime(candidate.detectedAt)} />
          <Fact label="Expires" value={dateTime(candidate.expiresAt)} />
        </dl>
        <div className="grid gap-3 lg:grid-cols-2">
          <div className="rounded-md border p-3 text-sm">
            <p className="font-semibold">Why Jax created it</p>
            <p className="mt-1 text-muted-foreground">
              {candidate.reasoning || evidence.summary || 'No concise reason was recorded.'}
            </p>
          </div>
          <div className="rounded-md border p-3 text-sm">
            <p className="font-semibold">What happened next</p>
            <p className="mt-1 text-muted-foreground">{next}</p>
          </div>
        </div>
        <p className="rounded-md border border-primary/40 bg-primary/5 p-3 text-sm font-semibold">
          NO FILL OCCURRED. The selected candidate journey created no order, position or fill.
        </p>
        <p className="text-sm text-muted-foreground">
          Use the Evidence, Paper Plan and Outcomes tabs for the persisted detail.
        </p>
      </CardContent>
    </Card>
  );
}

function EvidencePanel({
  evidence,
  journey,
  confidence,
}: {
  evidence: EvidenceView;
  journey?: OperatorCandidateEvidence;
  confidence?: number;
}) {
  const missing = [
    !evidence.chart && 'chart confirmation',
    !evidence.sentiment && 'sentiment analysis',
  ].filter(Boolean) as string[];
  return (
    <Card>
      <CardHeader>
        <CardTitle>Evidence</CardTitle>
        <CardDescription>
          Persisted inputs only. Missing evidence never implies a pass.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="font-semibold">
            {evidence.headline || 'No triggering evidence headline recorded'}
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            {evidence.summary || 'No evidence summary recorded.'}
          </p>
          <p className="mt-2 text-sm">
            <span className="font-semibold">Source: </span>
            {evidence.source || 'Not recorded'}
          </p>
          {evidence.sourceUrls.map((url) => (
            <a
              key={url}
              href={url}
              target="_blank"
              rel="noreferrer"
              className="mt-1 flex max-w-full items-center gap-2 break-all text-sm text-primary underline"
            >
              <ExternalLink className="h-4 w-4 shrink-0" />
              {url}
            </a>
          ))}
        </div>
        <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <Fact
            label="Asset mapping"
            value={evidence.assetThemes.length ? evidence.assetThemes.join(', ') : 'Not recorded'}
          />
          <Fact label="Mapping reason" value={evidence.mappingReason || 'Not recorded'} />
          <Fact
            label="Evidence score"
            value={
              journey?.evidenceScore === undefined
                ? 'Missing persisted evidence'
                : percent(journey.evidenceScore)
            }
          />
          <Fact label="Confidence" value={percent(confidence)} />
          <Fact
            label="Trust review"
            value={
              journey?.gateStatus ? humanize(journey.gateStatus) : 'Missing — no pass inferred'
            }
          />
          <Fact
            label="Risk review"
            value={
              journey?.riskStatus ? humanize(journey.riskStatus) : 'Missing — no pass inferred'
            }
          />
        </dl>
        {evidence.chart && (
          <div className="rounded-md border p-3 text-sm">
            <p className="font-semibold">
              Chart evidence: {evidence.chart.confirmed ? 'confirmed' : 'not confirmed'}
            </p>
            <p>{evidence.chart.reason || 'No chart explanation recorded.'}</p>
            <p className="text-muted-foreground">
              Last close {money(evidence.chart.lastClose)} · SMA 20 {money(evidence.chart.sma20)} ·
              5-candle move {percent(evidence.chart.fiveCandleChangePct)}
            </p>
          </div>
        )}
        {evidence.sentiment && (
          <div className="rounded-md border p-3 text-sm">
            <p className="font-semibold">Sentiment evidence: {evidence.sentiment.label}</p>
            <p>{evidence.sentiment.summary || 'No sentiment summary recorded.'}</p>
            {evidence.sentiment.limitations?.length ? (
              <p className="mt-1 text-muted-foreground">
                Limitations: {evidence.sentiment.limitations.join(' ')}
              </p>
            ) : null}
          </div>
        )}
        {missing.length > 0 && (
          <p className="rounded-md border border-warning/50 bg-warning/10 p-3 text-sm">
            Additional evidence not recorded: {missing.join(' and ')}.
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function PaperPlan({ journey }: { journey?: OperatorCandidateEvidence }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Paper Plan</CardTitle>
        <CardDescription>HYPOTHETICAL PAPER PLAN — NOT AN ORDER OR FILL</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <Fact label="Hypothetical entry" value={money(journey?.entry)} />
          <Fact label="Stop" value={money(journey?.stop)} />
          <Fact label="Target" value={money(journey?.target)} />
          <Fact
            label="Quantity"
            value={
              journey?.quantity === undefined
                ? 'No persisted sizing evidence available.'
                : String(journey.quantity)
            }
          />
          <Fact label="Planned maximum loss" value={money(journey?.plannedRisk)} />
          <Fact label="Planned reward" value={money(journey?.plannedReward)} />
          <Fact
            label="Reward/risk"
            value={
              journey?.rewardRisk === undefined
                ? 'Not supplied'
                : `${journey.rewardRisk.toFixed(2)}R`
            }
          />
          <Fact
            label="Leverage"
            value={
              journey?.leverage === undefined ? 'Not supplied' : `${journey.leverage.toFixed(2)}x`
            }
          />
        </dl>
        <details className="rounded-md border">
          <summary className="cursor-pointer px-3 py-2 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
            Show all paper-plan details
          </summary>
          <dl className="grid gap-2 border-t p-3 sm:grid-cols-2 lg:grid-cols-4">
            <Fact label="Notional" value={money(journey?.notional)} />
            <Fact
              label="Account-equity assumption"
              value={money(journey?.accountEquityAssumption)}
            />
            <Fact label="Paper-ticket ID" value={journey?.paperTicketId || 'Not applicable'} />
            <Fact label="Ticket state" value={humanize(journey?.paperTicketStatus)} />
          </dl>
        </details>
      </CardContent>
    </Card>
  );
}

function OutcomesPanel({
  candidateId,
  journey,
}: {
  candidateId: string;
  journey?: OperatorCandidateEvidence;
}) {
  const checkpoints = journey?.checkpoints ?? [];
  const completed = checkpoints.filter((item) => item.status === 'completed').length;
  const pending = checkpoints.filter((item) => item.status === 'pending_not_due').length;
  const missing = checkpoints.filter((item) =>
    ['pending_market_data', 'insufficient_data'].includes(item.status),
  ).length;
  const latest =
    [...checkpoints].reverse().find((item) => item.status === 'completed') ?? checkpoints[0];
  return (
    <Card>
      <CardHeader>
        <CardTitle>Outcomes</CardTitle>
        <CardDescription>Compact persisted checkpoint summary.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <Fact label="Completed checkpoints" value={String(completed)} />
          <Fact label="Pending checkpoints" value={String(pending)} />
          <Fact label="Missing-data checkpoints" value={String(missing)} />
          <Fact label="Latest checkpoint state" value={humanize(latest?.status)} />
          <Fact label="Stop touched" value={latest?.stopTouched ? 'Yes' : 'No'} />
          <Fact label="Target touched" value={latest?.targetTouched ? 'Yes' : 'No'} />
        </dl>
        <Button asChild variant="outline">
          <Link to="/outcomes">Open Hypothetical Outcomes</Link>
        </Button>
        <p className="text-xs text-muted-foreground">
          Candidate {candidateId}; hypothetical only, not a fill and no realised profit or loss.
        </p>
      </CardContent>
    </Card>
  );
}

function Audit({
  candidate,
  evidence,
  journey,
}: {
  candidate: CandidateTrade;
  evidence: EvidenceView;
  journey?: OperatorCandidateEvidence;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Audit</CardTitle>
        <CardDescription>
          Internal identifiers and historical context, shown only on request.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <Fact label="Candidate ID" value={candidate.id} />
          <Fact label="Source-event IDs" value={evidence.sourceEventId || 'Not recorded'} />
          <Fact label="Approval IDs" value={journey?.approvalId || 'Not applicable'} />
          <Fact label="Paper-ticket ID" value={journey?.paperTicketId || 'Not applicable'} />
          <Fact label="Original candidate status" value={candidate.status} />
          <Fact
            label="Original paper-ticket status"
            value={journey?.paperTicketStatus || 'Not applicable'}
          />
        </dl>
        <div>
          <p className="text-sm font-semibold">Selected-journey execution counts</p>
          <dl className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
            {Object.entries(journey?.selectedExecutionCounts ?? {}).map(([key, value]) => (
              <Fact key={key} label={humanize(key)} value={String(value)} />
            ))}
          </dl>
        </div>
        <p className="text-sm text-muted-foreground">
          Global historical execution counts:{' '}
          {Object.entries(journey?.historicalExecutionCounts ?? {})
            .map(([key, value]) => `${humanize(key)} ${value}`)
            .join(', ') || 'none recorded'}
          . These records are unrelated to this selected journey.
        </p>
        <details className="rounded-md border">
          <summary className="cursor-pointer px-3 py-2 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
            Show raw metadata
          </summary>
          <pre className="max-w-full overflow-auto whitespace-pre-wrap break-all border-t p-3 text-xs">
            {JSON.stringify(candidate.metadata ?? {}, null, 2)}
          </pre>
        </details>
      </CardContent>
    </Card>
  );
}

export function CandidateEvidencePage() {
  const { candidateId } = useParams<{ candidateId: string }>();
  const candidateQuery = useQuery({
    queryKey: ['candidate', candidateId],
    queryFn: () => candidatesService.get(candidateId ?? ''),
    enabled: Boolean(candidateId),
  });
  const journeyQuery = useQuery({
    queryKey: ['operator-candidate-evidence', candidateId],
    queryFn: () => operatorEvidenceService.candidate(candidateId ?? ''),
    enabled: Boolean(candidateId),
  });
  const evidence = useMemo(
    () => (candidateQuery.data ? buildEvidence(candidateQuery.data) : undefined),
    [candidateQuery.data],
  );

  if (!candidateId) return <div className="p-6">Missing candidate id.</div>;
  if (candidateQuery.isLoading) return <div className="p-6">Loading candidate evidence...</div>;
  if (candidateQuery.isError || !candidateQuery.data || !evidence)
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

  const candidate = candidateQuery.data;
  return (
    <div className="mx-auto flex w-full max-w-6xl min-w-0 flex-col gap-4">
      <Button asChild variant="outline" size="sm" className="self-start">
        <Link to="/etf/approvals">
          <ArrowLeft className="mr-2 h-4 w-4" /> Back to Candidates
        </Link>
      </Button>
      <header>
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {candidate.symbol} candidate
        </p>
        <h1 className="text-3xl font-bold">Candidate Review</h1>
        <p className="mt-2 max-w-3xl text-muted-foreground">
          Review persisted evidence and hypothetical follow-up without changing runtime state.
        </p>
        <p className="mt-3 rounded-md border border-primary/40 bg-primary/5 p-3 text-sm font-semibold">
          Review only — no order or fill exists.
        </p>
      </header>
      {journeyQuery.isError && (
        <p role="alert" className="text-destructive">
          Jax could not load the persisted journey. Candidate evidence remains read-only.
        </p>
      )}
      <h2 className="sr-only">Candidate review sections</h2>
      <Tabs defaultValue="overview" className="min-w-0">
        <TabsList
          className="h-auto w-full justify-start overflow-x-auto p-1"
          aria-label="Candidate Review sections"
        >
          {['Overview', 'Evidence', 'Paper Plan', 'Outcomes', 'Audit'].map((label) => (
            <TabsTrigger key={label} value={label.toLowerCase().replace(' ', '-')}>
              {label}
            </TabsTrigger>
          ))}
        </TabsList>
        <TabsContent value="overview">
          <Overview candidate={candidate} journey={journeyQuery.data} evidence={evidence} />
        </TabsContent>
        <TabsContent value="evidence">
          <EvidencePanel
            evidence={evidence}
            journey={journeyQuery.data}
            confidence={candidate.confidence}
          />
        </TabsContent>
        <TabsContent value="paper-plan">
          <PaperPlan journey={journeyQuery.data} />
        </TabsContent>
        <TabsContent value="outcomes">
          <OutcomesPanel candidateId={candidateId} journey={journeyQuery.data} />
        </TabsContent>
        <TabsContent value="audit">
          <Audit candidate={candidate} evidence={evidence} journey={journeyQuery.data} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
