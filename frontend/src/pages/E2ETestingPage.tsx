import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { MonitorCheck, Play, Loader2, AlertTriangle, RotateCcw } from 'lucide-react';
import { e2eTestService, type PlaywrightRunResult } from '@/data/e2e-test-service';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { HelpHint } from '@/components/ui/help-hint';

const SPECS = [
  { id: 'auth',      label: 'Auth', description: 'Login, session, anonymous bypass' },
  { id: 'signals',   label: 'Signals', description: 'Queue display, approve, reject' },
  { id: 'portfolio', label: 'Portfolio', description: 'Positions, protect, close dialogs' },
  { id: 'research',  label: 'Research', description: 'Backtesting, instances, run history' },
  { id: 'system',    label: 'System', description: 'Dataset snapshots, system events' },
  { id: 'settings',  label: 'Settings', description: 'Settings controls rendering' },
  { id: 'testing',   label: 'Testing', description: 'Trust gates, trigger endpoints' },
];

function statusBadge(status: PlaywrightRunResult['status']) {
  if (status === 'passed') {
    return <span className="inline-flex rounded bg-emerald-500/15 px-2 py-1 text-xs font-medium text-emerald-400">passed</span>;
  }
  if (status === 'failed') {
    return <span className="inline-flex rounded bg-red-500/15 px-2 py-1 text-xs font-medium text-red-400">failed</span>;
  }
  if (status === 'running') {
    return <span className="inline-flex items-center gap-1 rounded bg-yellow-500/15 px-2 py-1 text-xs font-medium text-yellow-400"><Loader2 className="h-3 w-3 animate-spin" />running</span>;
  }
  return <span className="inline-flex rounded bg-muted px-2 py-1 text-xs font-medium text-muted-foreground">idle</span>;
}

export function E2ETestingPage() {
  const queryClient = useQueryClient();

  const resultsQuery = useQuery({
    queryKey: ['e2e-results'],
    queryFn: () => e2eTestService.getResults(),
    refetchInterval: (query) => {
      return query.state.data?.status === 'running' ? 1500 : false;
    },
  });

  const result = resultsQuery.data;
  const isRunning = result?.status === 'running';

  const runMutation = useMutation({
    mutationFn: (spec?: string) => e2eTestService.run(spec),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['e2e-results'] });
    },
  });

  const handleRun = (spec?: string) => {
    runMutation.mutate(spec);
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">UI TESTS</p>
        <h1 className="flex items-center gap-2 text-2xl font-bold md:text-3xl">
          E2E Test Runner
          <HelpHint text="Run Playwright end-to-end tests against the app. Tests use stubbed API responses and run in a local browser." />
        </h1>
        <p className="text-muted-foreground mt-1">
          Trigger Playwright browser tests for each feature area. Tests are isolated with API stubs — no live trading required.
        </p>
      </div>

      {/* Run Controls */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MonitorCheck className="h-5 w-5" />
            Test Suites
            <HelpHint text="Each suite maps to one Playwright spec file in frontend/e2e/." />
          </CardTitle>
          <CardDescription>Click a suite to run scoped tests, or Run All to execute the full suite.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Run All */}
          <div className="flex items-center gap-3 flex-wrap">
            <Button
              onClick={() => handleRun()}
              disabled={isRunning || runMutation.isPending}
              variant="secondary"
              className="min-w-[120px]"
            >
              {isRunning || runMutation.isPending ? (
                <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Running…</>
              ) : (
                <><Play className="mr-2 h-4 w-4" />Run All</>
              )}
            </Button>
            {result && result.status !== 'idle' && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => queryClient.invalidateQueries({ queryKey: ['e2e-results'] })}
              >
                <RotateCcw className="mr-1 h-3 w-3" />
                Refresh
              </Button>
            )}
            {runMutation.isError && (
              <p className="flex items-center gap-1 text-sm text-destructive">
                <AlertTriangle className="h-4 w-4" />
                {(runMutation.error as Error).message}
              </p>
            )}
          </div>

          {/* Individual Spec Buttons */}
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {SPECS.map((spec) => (
              <div key={spec.id} className="rounded-md border border-border p-3 flex flex-col gap-2">
                <div>
                  <p className="text-sm font-medium">{spec.label}</p>
                  <p className="text-xs text-muted-foreground">{spec.description}</p>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleRun(spec.id)}
                  disabled={isRunning || runMutation.isPending}
                  className="self-start"
                >
                  <Play className="mr-1 h-3 w-3" />
                  Run {spec.label}
                </Button>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Results Panel */}
      {result && result.status !== 'idle' && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 flex-wrap">
              Last Run
              {statusBadge(result.status)}
              {result.spec && (
                <span className="text-sm font-normal text-muted-foreground">
                  spec: <code className="rounded bg-muted px-1 py-0.5 text-xs">{result.spec}</code>
                </span>
              )}
              {result.durationMs != null && (
                <span className="text-sm font-normal text-muted-foreground">
                  {(result.durationMs / 1000).toFixed(1)}s
                </span>
              )}
            </CardTitle>
            {result.startedAt && (
              <CardDescription>
                Started {new Date(result.startedAt).toLocaleString()}
                {result.completedAt && ` · Completed ${new Date(result.completedAt).toLocaleString()}`}
                {result.exitCode != null && ` · Exit code ${result.exitCode}`}
              </CardDescription>
            )}
          </CardHeader>
          <CardContent>
            {result.output ? (
              <pre className="overflow-x-auto rounded-md bg-black/80 p-4 text-xs text-green-300 font-mono whitespace-pre-wrap max-h-[500px] overflow-y-auto leading-relaxed">
                {result.output}
              </pre>
            ) : (
              <p className="text-sm text-muted-foreground italic">
                {result.status === 'running' ? 'Tests are running — output will appear when complete.' : 'No output captured.'}
              </p>
            )}
            {result.message && (
              <p className="mt-2 text-sm text-destructive flex items-center gap-1">
                <AlertTriangle className="h-4 w-4" />
                {result.message}
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
