import { useQuery } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { AlertTriangle, CircleHelp, ShieldCheck } from 'lucide-react';
import { PageIntro, GlossaryTerm } from '@/components/ui/beginner-help';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { HealthPanel, MetricsPanel, MemoryBrowserPanel } from '@/components/dashboard';
import { DashboardGrid, DashboardPanel } from '@/components/layout';
import { datasetsService } from '@/data/datasets-service';
import { eventsService } from '@/data/events-service';
import { operatorEvidenceService } from '@/data/operator-evidence-service';
import { useOperatorEvidenceOverview } from '@/hooks/useOperatorEvidenceOverview';
import { interpretOperatorSafety, type SafetyState } from '@/lib/operator-safety';
import { cn } from '@/lib/utils';

const executionKeys = [
  ['executionInstructions', 'Execution instructions'],
  ['orderIntents', 'Order intents'],
  ['brokerOrders', 'Broker orders'],
  ['trades', 'Trades'],
  ['fills', 'Fills'],
] as const;

export function SystemPage() {
  const [params] = useSearchParams();
  const candidateId = params.get('candidateId');
  const overview = useOperatorEvidenceOverview();
  const safety = interpretOperatorSafety(overview.data);
  const journey = useQuery({
    queryKey: ['operator-candidate-evidence', candidateId],
    queryFn: () => operatorEvidenceService.candidate(candidateId!),
    enabled: Boolean(candidateId),
  });
  const events = useQuery({
    queryKey: ['system-events'],
    queryFn: () => eventsService.list({ limit: 20 }),
  });
  const datasets = useQuery({
    queryKey: ['system-datasets'],
    queryFn: () => datasetsService.list({ limit: 20 }),
  });

  return (
    <div className="mx-auto flex w-full max-w-6xl min-w-0 flex-col gap-6">
      <PageIntro
        eyebrow="System Safety"
        title="System Safety"
        description="Confirm that Jax remains in paper-safe mode and review whether any execution-side records exist."
      />

      <SafetySummary
        state={overview.isPending ? 'unknown' : safety.state}
        unavailable={overview.isError}
      />

      <section aria-labelledby="critical-safety-heading">
        <div className="mb-3 flex items-end justify-between gap-3">
          <div>
            <h2 id="critical-safety-heading" className="text-xl font-semibold">
              Critical safety settings
            </h2>
            <p className="text-sm text-muted-foreground">
              Read-only runtime evidence. This page cannot change these settings.
            </p>
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {safety.checks.map(({ key, ...check }) => (
            <SafetyCard key={key} {...check} />
          ))}
        </div>
      </section>

      <section
        aria-labelledby="journey-heading"
        className="rounded-lg border border-border bg-card p-4 sm:p-5"
      >
        <h2 id="journey-heading" className="text-xl font-semibold">
          This journey
        </h2>
        {!candidateId ? (
          <p className="mt-2 text-sm text-muted-foreground">
            No specific candidate journey is selected. Global historical records are shown
            separately below.
          </p>
        ) : journey.isPending ? (
          <p className="mt-2 text-sm text-muted-foreground">Loading the selected journey.</p>
        ) : journey.isError || !journey.data ? (
          <p role="alert" className="mt-2 text-sm text-destructive">
            Selected-journey evidence is unavailable.
          </p>
        ) : (
          <ExecutionCounts counts={journey.data.selectedExecutionCounts} selected />
        )}
      </section>

      <section
        aria-labelledby="historical-heading"
        className="rounded-lg border border-border bg-card p-4 sm:p-5"
      >
        <h2 id="historical-heading" className="text-xl font-semibold">
          Historical records
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          These are database-wide historical records and may be unrelated to the candidate or
          evidence item currently being reviewed.
        </p>
        {overview.isError || !overview.data ? (
          <p role="alert" className="mt-3 text-sm text-destructive">
            Historical record totals are unavailable.
          </p>
        ) : (
          <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-7">
            <Count label="Approvals" value={overview.data.approvals} />
            <Count label="Paper tickets" value={overview.data.paperTickets} />
            <Count
              label="Execution instructions"
              value={overview.data.historicalExecutionInstructions}
            />
            <Count label="Order intents" value={overview.data.historicalOrderIntents} />
            <Count label="Broker orders" value={overview.data.historicalBrokerOrders} />
            <Count label="Trades" value={overview.data.historicalTrades} />
            <Count label="Fills" value={overview.data.historicalFills} />
          </div>
        )}
      </section>

      <section
        aria-labelledby="next-step-heading"
        className="rounded-lg border border-primary/40 bg-primary/5 p-4"
      >
        <h2 id="next-step-heading" className="font-semibold">
          Next step
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          If any setting is unsafe or unknown, stop relying on Jax and ask an administrator to
          review the runtime configuration.
        </p>
        <Button
          asChild
          variant="link"
          className="mt-2 h-auto max-w-full justify-start whitespace-normal p-0 text-left"
        >
          <Link to="/guide#system-safety">Learn how this page fits into the Jax workflow.</Link>
        </Button>
      </section>

      <details className="rounded-lg border border-border bg-card">
        <summary className="cursor-pointer px-4 py-4 text-lg font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          Technical diagnostics
        </summary>
        <div className="space-y-5 border-t border-border p-4">
          {overview.isError && (
            <p role="alert" className="text-sm text-destructive">
              Technical diagnostics could not be loaded. The page remains read-only.
            </p>
          )}
          <DiagnosticGroup title="Runtime configuration">
            <dl className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
              <Diagnostic label="JAX_RUNTIME_MODE" value={overview.data?.runtimeMode} />
              <Diagnostic
                label="ALLOW_LIVE_TRADING"
                value={formatRaw(overview.data?.allowLiveTrading)}
              />
              <Diagnostic
                label="EXECUTION_ENABLED"
                value={formatRaw(overview.data?.executionEnabled)}
              />
              <Diagnostic
                label="EXECUTION_INSTRUCTION_WORKER_ENABLED"
                value={formatRaw(overview.data?.executionWorkerEnabled)}
              />
              <Diagnostic
                label="BROKER_EXECUTION_ALLOWED"
                value={formatRaw(overview.data?.brokerExecutionAllowed)}
              />
              <Diagnostic label="MAX_LEVERAGE" value={overview.data?.maximumLeverage?.toString()} />
              <Diagnostic
                label="Last successful status check"
                value={formatDate(overview.data?.checkedAt)}
              />
            </dl>
          </DiagnosticGroup>
          <DiagnosticGroup title="Services and health">
            <HealthPanel isOpen onToggle={() => undefined} />
          </DiagnosticGroup>
          <DiagnosticGroup title="Data and events">
            <p className="text-sm text-muted-foreground">
              {datasets.isError || events.isError
                ? 'Some dataset or event diagnostics are unavailable.'
                : `${datasets.data?.datasets.length ?? 0} datasets and ${events.data?.events.length ?? 0} recent events loaded.`}
            </p>
          </DiagnosticGroup>
          <DiagnosticGroup title="Metrics and logs">
            <DashboardGrid>
              <DashboardPanel>
                <MetricsPanel isOpen onToggle={() => undefined} />
              </DashboardPanel>
              <DashboardPanel>
                <MemoryBrowserPanel isOpen onToggle={() => undefined} />
              </DashboardPanel>
            </DashboardGrid>
          </DiagnosticGroup>
          <DiagnosticGroup title="Raw evidence">
            <details className="rounded-md border border-border">
              <summary className="cursor-pointer p-3 font-medium">Raw API response</summary>
              <pre className="max-w-full overflow-x-auto border-t p-3 text-xs">
                {overview.data ? JSON.stringify(overview.data, null, 2) : 'Unavailable'}
              </pre>
            </details>
          </DiagnosticGroup>
        </div>
      </details>

      <details className="rounded-lg border border-border bg-card">
        <summary className="flex cursor-pointer items-center gap-2 px-4 py-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          <CircleHelp className="h-4 w-4" />
          What does this mean?
        </summary>
        <div className="grid gap-3 border-t p-4 text-sm text-muted-foreground sm:grid-cols-2">
          {(
            [
              'Paper mode',
              'Live trading',
              'Execution',
              'Execution worker',
              'Broker execution',
              'Leverage',
              'Selected journey',
              'Historical records',
              'Order intent',
              'Broker order',
              'Trade',
              'Fill',
            ] as const
          ).map((term) => (
            <p key={term}>
              <GlossaryTerm term={term} />
            </p>
          ))}
        </div>
      </details>
    </div>
  );
}

function SafetySummary({ state, unavailable }: { state: SafetyState; unavailable: boolean }) {
  const safe = state === 'safe';
  const title = safe
    ? 'Paper-safe mode is on'
    : state === 'unsafe'
      ? 'Safety warning'
      : 'Safety is unknown';
  const copy = unavailable
    ? 'Jax could not load the current runtime state. Treat safety as unknown.'
    : safe
      ? 'Paper-safe mode is on. Live trading, execution and broker activity are disabled.'
      : state === 'unsafe'
        ? 'Jax is not in a confirmed paper-safe state. Review the settings below before relying on this system.'
        : 'Jax could not confirm every required safety setting. Treat the system state as unknown until the missing values are reviewed.';
  return (
    <section
      role="status"
      aria-live="polite"
      className={cn(
        'rounded-lg border p-4 sm:p-5',
        safe ? 'border-success/60 bg-success/5' : 'border-destructive/60 bg-destructive/5',
      )}
    >
      <div className="flex gap-3">
        {safe ? (
          <ShieldCheck className="mt-0.5 h-5 w-5 text-success" />
        ) : (
          <AlertTriangle className="mt-0.5 h-5 w-5 text-destructive" />
        )}
        <div>
          <h2 className="font-semibold">{title}</h2>
          <p className="mt-1 text-sm text-muted-foreground">{copy}</p>
        </div>
      </div>
    </section>
  );
}

function SafetyCard({
  label,
  value,
  state,
  explanation,
}: {
  label: string;
  value: string;
  state: SafetyState;
  explanation: string;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between gap-2">
          <p className="text-xl font-bold">{value}</p>
          <span
            className={cn(
              'rounded-full border px-2 py-1 text-xs font-semibold',
              state === 'safe'
                ? 'border-success/50 text-success'
                : state === 'unsafe'
                  ? 'border-destructive/50 text-destructive'
                  : 'border-warning/50 text-warning',
            )}
          >
            {state === 'safe' ? 'Safe' : state === 'unsafe' ? 'Unsafe' : 'Unknown'}
          </span>
        </div>
        <p className="mt-2 text-xs text-muted-foreground">{explanation}</p>
      </CardContent>
    </Card>
  );
}

function ExecutionCounts({
  counts,
  selected,
}: {
  counts: Record<string, number>;
  selected?: boolean;
}) {
  const present = executionKeys.filter(([key]) => (counts[key] ?? 0) > 0);
  return (
    <div className="mt-4">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        {executionKeys.map(([key, label]) => (
          <Count key={key} label={label} value={counts[key]} />
        ))}
      </div>
      <p
        className={cn(
          'mt-4 text-sm font-semibold',
          present.length ? 'text-destructive' : 'text-success',
        )}
      >
        {present.length
          ? `Warning: this journey created ${present.map(([, label]) => label.toLowerCase()).join(', ')}.`
          : selected
            ? 'This journey created no execution records.'
            : ''}
      </p>
    </div>
  );
}

function Count({ label, value }: { label: string; value: number | null | undefined }) {
  return (
    <div className="min-w-0 rounded-md border border-border p-3">
      <dt className="break-words text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-xl font-bold">
        {value === null || value === undefined ? 'Unavailable' : value}
      </dd>
    </div>
  );
}
function DiagnosticGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-2 font-semibold">{title}</h3>
      {children}
    </section>
  );
}
function Diagnostic({ label, value }: { label: string; value: string | null | undefined }) {
  return (
    <div className="min-w-0 rounded-md border p-3">
      <dt className="break-all text-xs text-muted-foreground">{label}</dt>
      <dd className="mt-1 break-words font-medium">{value || 'Unknown'}</dd>
    </div>
  );
}
function formatRaw(value: boolean | null | undefined) {
  return value === null || value === undefined ? undefined : String(value);
}
function formatDate(value: string | undefined) {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
