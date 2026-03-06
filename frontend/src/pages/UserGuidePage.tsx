import { BookOpen, CheckCircle2, ListChecks, Wrench } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { HelpHint } from '@/components/ui/help-hint';

export function UserGuidePage() {
  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">GETTING STARTED</p>
        <h1 className="flex items-center gap-2 text-2xl font-bold md:text-3xl">
          User Guide
          <HelpHint text="Step-by-step instructions for trading, backtesting, analysis, and testing workflows." />
        </h1>
        <p className="text-muted-foreground mt-1">
          Practical steps for placing and managing paper trades, running real-data backtests, reviewing results, and validating trust gates.
        </p>
      </div>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <BookOpen className="h-5 w-5" />
          <div>
            <CardTitle>Place and Manage a Trade</CardTitle>
            <CardDescription>Use the trading UI as an operator workflow, not just a dashboard.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ol className="list-decimal pl-5 space-y-1 text-foreground">
            <li>Open `Trading` and pick a symbol from Watchlist or the chart.</li>
            <li>Use `Order Ticket` for a market or limit entry.</li>
            <li>Add `Stop Loss` and optional `Take Profit` before submitting if you want bracket protection.</li>
            <li>Use `Trade Blotter` to cancel any working broker order that has not filled.</li>
            <li>Use `Portfolio` or `Positions` to close or re-protect an open position.</li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <CheckCircle2 className="h-5 w-5" />
          <div>
            <CardTitle>Close or Protect Open Exposure</CardTitle>
            <CardDescription>Position management happens from the Positions panel.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ul className="list-disc pl-5 space-y-1 text-foreground">
            <li>`Close` submits a market or limit exit for all or part of a position.</li>
            <li>`Protect` submits a stop loss and optional take profit on the broker for the chosen quantity.</li>
            <li>When you submit new protection, existing UI-created protection for that symbol is replaced.</li>
            <li>Use `Trade Blotter` to cancel working orders; filled history remains visible but read only.</li>
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <BookOpen className="h-5 w-5" />
          <div>
            <CardTitle>Dataset Snapshots</CardTitle>
            <CardDescription>Backtests require dataset snapshots (CSV OHLCV).</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ol className="list-decimal pl-5 space-y-1 text-foreground">
            <li>Open `System` → `Dataset Snapshots`.</li>
            <li>Confirm at least one snapshot exists.</li>
            <li>If empty, add a dataset under `data/datasets` and restart `jax-research`.</li>
          </ol>
          <img
            className="rounded-md border border-border"
            src="/user-guide/system-datasets.png"
            alt="System page showing dataset snapshots"
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <ListChecks className="h-5 w-5" />
          <div>
            <CardTitle>Run a Backtest</CardTitle>
            <CardDescription>Use a strategy instance with a dataset snapshot.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ol className="list-decimal pl-5 space-y-1 text-foreground">
            <li>Go to `Backtesting` → `Research`.</li>
            <li>Select a Strategy Instance in the table.</li>
            <li>Pick a Dataset Snapshot in the editor below.</li>
            <li>Click `Run Backtest`.</li>
          </ol>
          <img
            className="rounded-md border border-border"
            src="/user-guide/research-backtest.png"
            alt="Research page showing dataset selection and Run Backtest button"
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <ListChecks className="h-5 w-5" />
          <div>
            <CardTitle>Review Results</CardTitle>
            <CardDescription>Open the run in Analysis to inspect trades and timeline.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ol className="list-decimal pl-5 space-y-1 text-foreground">
            <li>Go to `Backtesting` → `Backtests` tab.</li>
            <li>Click a run row to open `Analysis`.</li>
            <li>Check dataset hash and provenance.</li>
          </ol>
          <img
            className="rounded-md border border-border"
            src="/user-guide/backtest-runs.png"
            alt="Backtests list showing completed runs"
          />
          <img
            className="rounded-md border border-border"
            src="/user-guide/analysis-run.png"
            alt="Analysis page with run details"
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <Wrench className="h-5 w-5" />
          <div>
            <CardTitle>Testing and Trust Gates</CardTitle>
            <CardDescription>Validate data integrity and safety checks.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ul className="list-disc pl-5 space-y-1 text-foreground">
            <li>Open `Testing` and run the reconciliation jobs in paper mode.</li>
            <li>Each gate produces an artifact report under `/reports`.</li>
            <li>Review failed gates before enabling live trading.</li>
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-col items-start gap-2 sm:flex-row sm:items-center">
          <CheckCircle2 className="h-5 w-5" />
          <div>
            <CardTitle>Daily Workflow</CardTitle>
            <CardDescription>Suggested routine for paper trading.</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="space-y-2 text-sm text-muted-foreground">
          <ol className="list-decimal pl-5 space-y-1 text-foreground">
            <li>Check `System` health and datasets.</li>
            <li>Run backtests for new ideas in `Research`.</li>
            <li>Inspect results in `Analysis` and document observations.</li>
            <li>Run `Testing` gates and archive artifacts.</li>
          </ol>
        </CardContent>
      </Card>
    </div>
  );
}
