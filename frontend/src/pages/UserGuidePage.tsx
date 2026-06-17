import { BookOpen, CheckCircle2, ListChecks, Wrench } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { HelpHint } from '@/components/ui/help-hint';
import { PilotStatusBanner } from '@/components/ui/PilotStatusBanner';
import { useTradingPilotStatus } from '@/hooks/useTradingPilotStatus';

export function UserGuidePage() {
  const { data: pilotStatus } = useTradingPilotStatus();

  return (
    <div className="space-y-6">
      <div>
        <p className="mb-1 text-xs font-semibold uppercase tracking-widest text-primary">Getting started</p>
        <h1 className="flex items-center gap-2 text-2xl font-bold md:text-3xl">
          User Guide
          <HelpHint text="A practical first-run workflow for research, AI review, and paper trading." />
        </h1>
        <p className="mt-1 text-muted-foreground">
          Use this to prove the local stack is healthy before trusting any research result or paper-trading workflow.
        </p>
      </div>

      {pilotStatus ? (
        <PilotStatusBanner
          title="Pilot trading policy"
          readOnly={pilotStatus.readOnly}
          reasons={pilotStatus.reasons}
          checklist={pilotStatus.checklist}
        />
      ) : null}

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <ListChecks className="h-5 w-5" />
          <div>
            <CardTitle>First-Run Checklist</CardTitle>
            <CardDescription>Run these checks in order. Stop at the first failed check and use the linked diagnostic page.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <ol className="list-decimal space-y-1 pl-5 text-foreground">
            <li>Start the stack with `.\start.ps1`. It applies migrations, syncs dataset snapshots, and rebuilds stale service images.</li>
            <li>Confirm services: open `System` and check runtime, providers, market data, and dataset snapshots.</li>
            <li>Confirm Monitor ingestion: open `Monitor Inbox`. Empty means no payload has arrived; route/auth/schema errors are shown separately.</li>
            <li>Run one dataset-backed research check from `Research`. A dataset ID and hash must be visible before running.</li>
            <li>Open the completed run in `Analysis` and verify the dataset hash, source provider, and trade table are present.</li>
            <li>Only after those checks pass, use `Manual Trading` for a tiny non-ETF paper order. ETF entries must come from a real candidate approval.</li>
          </ol>
          <div className="rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
            Troubleshooting: use `System` for service health, `Monitor Inbox` for payload diagnostics, `Testing` for gate checks, and `Docs/DEBUGGING.md` for container logs.
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <BookOpen className="h-5 w-5" />
          <div>
            <CardTitle>What Each Main Page Does</CardTitle>
            <CardDescription>Use these pages in order while learning the app.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <ul className="list-disc space-y-1 pl-5 text-foreground">
            <li>`Research` runs dataset-backed backtests. It does not place trades.</li>
            <li>`AI Trading` shows queued opportunities and can ask the local AI for a symbol view.</li>
            <li>`Manual Trading` is where protected paper orders are submitted and managed.</li>
            <li>`Approvals` is for approval-required ETF ideas; manual ETF entries are blocked by policy.</li>
            <li>`Macro Events` shows ingested research events, including World Monitor triggers.</li>
            <li>`Analysis` opens completed backtests so you can inspect results and provenance.</li>
            <li>`System` shows runtime health, datasets, and diagnostics.</li>
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <CheckCircle2 className="h-5 w-5" />
          <div>
            <CardTitle>Dataset Snapshots</CardTitle>
            <CardDescription>Backtests run from registered CSV snapshots, not from whatever file happens to exist.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <ol className="list-decimal space-y-1 pl-5 text-foreground">
            <li>Datasets live in `data/datasets` and are listed in `data/datasets/catalog.json`.</li>
            <li>`.\start.ps1` syncs that catalog into Postgres after migrations.</li>
            <li>`jax-research` loads the catalog on startup, so restart it after adding new dataset files by hand.</li>
            <li>For the beginner research path, use the SPY 1-minute snapshot because opening-range strategies need intraday candles.</li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <BookOpen className="h-5 w-5" />
          <div>
            <CardTitle>Paper Orders</CardTitle>
            <CardDescription>Paper trading is still a broker mutation, so keep every test tiny.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <ol className="list-decimal space-y-1 pl-5 text-foreground">
            <li>Use `Manual Trading` for non-ETF manual paper orders such as AAPL.</li>
            <li>Start with quantity `1`, add `Stop Loss`, and add optional `Take Profit`.</li>
            <li>Confirm the IB/TWS checkbox before submitting.</li>
            <li>Use `Trade Blotter` to inspect or cancel working stop/target orders.</li>
            <li>Use `Positions` to close or protect open exposure after a fill.</li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <Wrench className="h-5 w-5" />
          <div>
            <CardTitle>News Monitor Flow</CardTitle>
            <CardDescription>World Monitor input is research context, not an order instruction.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <ul className="list-disc space-y-1 pl-5 text-foreground">
            <li>The Jax endpoint is `POST /api/v1/research/events/world-monitor`.</li>
            <li>Accepted payloads become Monitor Inbox rows and normalized research events with original source URLs retained.</li>
            <li>Promotion to a candidate is a separate step and requires evidence, mapping, and chart confirmation.</li>
            <li>Payload receipt must not create broker orders or execution instructions directly.</li>
            <li>Use `Monitor Inbox` for raw payload provenance and `Macro Events` for normalized event context.</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
