import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FlaskConical, ShieldCheck, AlertTriangle, Activity, Play } from 'lucide-react';
import { testingService } from '@/data/testing-service';
import { useHealth } from '@/hooks/useHealth';
import type { TriggerTestResponse } from '@/data/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { HelpHint } from '@/components/ui/help-hint';

type TriggerName = 'data' | 'pnl' | 'failure' | 'flatten';

export function TestingPage() {
  const queryClient = useQueryClient();
  const gatesQuery = useQuery({
    queryKey: ['testing-status'],
    queryFn: () => testingService.getStatus(),
  });
  const testRunsQuery = useQuery({
    queryKey: ['testing-runs'],
    queryFn: () => testingService.getTestRuns(50),
  });
  const healthQuery = useHealth();

  const triggerMutation = useMutation({
    mutationFn: async (trigger: TriggerName): Promise<TriggerTestResponse> => {
      switch (trigger) {
        case 'data':
          return testingService.triggerDataRecon();
        case 'pnl':
          return testingService.triggerPnlRecon();
        case 'failure':
          return testingService.triggerFailureSuite();
        case 'flatten':
          return testingService.triggerFlattenProof();
      }
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['testing-status'] }),
        queryClient.invalidateQueries({ queryKey: ['testing-runs'] }),
      ]);
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">SAFETY CHECKS</p>
        <h1 className="flex items-center gap-2 text-2xl font-bold md:text-3xl">
          Testing
          <HelpHint text="Run reconciliation and safety checks. These are paper-mode only." />
        </h1>
        <p className="text-muted-foreground mt-1">
          Run the safety checks that prove data integrity and paper-trading readiness.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" />
            Trust Gates Checklist
            <HelpHint text="Each gate produces an artifact report and status." />
          </CardTitle>
          <CardDescription>Gate0..Gate7 with latest state and artifact links.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="w-full overflow-x-auto">
            <Table className="min-w-[640px]">
            <TableHeader>
              <TableRow>
                <TableHead>Gate</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Run</TableHead>
                <TableHead>Artifact</TableHead>
                <TableHead>Updated</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(gatesQuery.data ?? []).map((gate) => {
                const artifact = asString(gate.details?.artifactUri) || asString(gate.details?.artifactURI);
                return (
                  <TableRow key={gate.gate}>
                    <TableCell>{gate.gate}</TableCell>
                    <TableCell>{statusBadge(gate.status)}</TableCell>
                    <TableCell>{fmtDate(gate.lastRunAt)}</TableCell>
                    <TableCell>
                      {artifact ? (
                        <a href={artifact} className="text-primary underline" target="_blank" rel="noreferrer">
                          Artifact
                        </a>
                      ) : (
                        '-'
                      )}
                    </TableCell>
                    <TableCell>{fmtDate(gate.updatedAt)}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Quick Diagnostics
            <HelpHint text="Health checks for the running services." />
          </CardTitle>
          <CardDescription>Health checks for trader, research, and IB bridge runtime.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {(healthQuery.data?.services ?? []).map((service) => (
            <div key={service.name} className="rounded-md border border-border p-3">
              <p className="text-sm font-medium">{service.name}</p>
              <p className="text-sm text-muted-foreground">{service.status}</p>
              <p className="text-xs text-muted-foreground">
                {service.latency ? `${service.latency}ms` : '-'} | {fmtDate(new Date(service.lastCheck).toISOString())}
              </p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FlaskConical className="h-5 w-5" />
            Run Tests (Paper Mode)
            <HelpHint text="Triggers runbook jobs that write reports under /reports." />
          </CardTitle>
          <CardDescription>These endpoints are guarded server-side and reject in live mode.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
          <Button onClick={() => triggerMutation.mutate('data')} disabled={triggerMutation.isPending}>
            <Play className="mr-1 h-4 w-4" />
            Run Data Reconciliation
          </Button>
          <Button onClick={() => triggerMutation.mutate('pnl')} disabled={triggerMutation.isPending}>
            <Play className="mr-1 h-4 w-4" />
            Run P/L Reconciliation
          </Button>
          <Button onClick={() => triggerMutation.mutate('failure')} disabled={triggerMutation.isPending}>
            <Play className="mr-1 h-4 w-4" />
            Run Failure Test Suite
          </Button>
          <Button variant="outline" onClick={() => triggerMutation.mutate('flatten')} disabled={triggerMutation.isPending}>
            <Play className="mr-1 h-4 w-4" />
            Run Flatten Proof
          </Button>
          {triggerMutation.isError && (
            <p className="flex items-center gap-1 text-sm text-destructive">
              <AlertTriangle className="h-4 w-4" />
              {(triggerMutation.error as Error).message}
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            Recent Test Runs
            <HelpHint text="Latest test runs with links to generated artifacts." />
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="w-full overflow-x-auto">
            <Table className="min-w-[600px]">
            <TableHeader>
              <TableRow>
                <TableHead>Test</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Started</TableHead>
                <TableHead>Completed</TableHead>
                <TableHead>Artifact</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(testRunsQuery.data ?? []).map((run) => (
                <TableRow key={run.id}>
                  <TableCell>{run.testName}</TableCell>
                  <TableCell>{statusBadge(run.status)}</TableCell>
                  <TableCell>{fmtDate(run.startedAt)}</TableCell>
                  <TableCell>{fmtDate(run.completedAt)}</TableCell>
                  <TableCell>
                    {run.artifactUri ? (
                      <a href={run.artifactUri} className="text-primary underline" target="_blank" rel="noreferrer">
                        Artifact
                      </a>
                    ) : (
                      '-'
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function fmtDate(raw?: string | null): string {
  if (!raw) {
    return '-';
  }
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return raw;
  }
  return d.toLocaleString();
}

function statusBadge(status: string) {
  const normalized = status.toLowerCase();
  if (normalized === 'passed' || normalized === 'healthy' || normalized === 'passing') {
    return <span className="inline-flex rounded bg-emerald-500/15 px-2 py-1 text-xs font-medium text-emerald-400">{status}</span>;
  }
  if (normalized === 'failed' || normalized === 'failing' || normalized === 'unhealthy') {
    return <span className="inline-flex rounded bg-red-500/15 px-2 py-1 text-xs font-medium text-red-400">{status}</span>;
  }
  return <span className="inline-flex rounded bg-muted px-2 py-1 text-xs font-medium">{status}</span>;
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : '';
}
