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
          Use this as the first full local test: confirm data, run research, ask the local AI, then place a tiny paper trade.
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
            <CardTitle>First Full Test</CardTitle>
            <CardDescription>The shortest path that proves the local stack is useful.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <ol className="list-decimal space-y-1 pl-5 text-foreground">
            <li>Start everything from this repo with `.\start.ps1`.</li>
            <li>Open `Research` and confirm `SPY_2026-06-09_2026-06-11_1m` is selected.</li>
            <li>Click `Run guided backtest`; the first beginner setup should use `or-spy-paper-v1`.</li>
            <li>Open `AI Trading`, enter `AAPL` or `SPY` in `Ask Jax`, and click `Ask`.</li>
            <li>Open `Manual Trading`, use 1 share, add a stop loss, add a take profit, and submit only in paper mode.</li>
            <li>Check `Trade Blotter` for the parent entry and attached stop/target orders.</li>
          </ol>
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
            <li>Accepted payloads become `research_trigger` inbox rows and normalized events.</li>
            <li>They must not create candidates, approvals, broker orders, or execution instructions directly.</li>
            <li>Use `Macro Events` to inspect accepted research events after ingestion.</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
