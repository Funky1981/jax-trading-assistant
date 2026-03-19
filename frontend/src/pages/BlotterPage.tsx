import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { TradeBlotterPanel } from '@/components/dashboard';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { HelpHint } from '@/components/ui/help-hint';
import { candidatesService, type CandidateTrade } from '@/data/approvals-service';

function fmtDate(raw?: string | null) {
  if (!raw) return '-';
  return new Date(raw).toLocaleString();
}

function compactId(value?: string) {
  if (!value) return '-';
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function statusVariant(status: string) {
  switch (status) {
    case 'filled':
      return 'success';
    case 'submitted':
      return 'default';
    case 'approved':
      return 'secondary';
    default:
      return 'outline';
  }
}

function SummaryCard({ label, value, hint }: { label: string; value: string | number; hint: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold">{value}</p>
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}

function ChainRow({ item }: { item: CandidateTrade }) {
  return (
    <div className="rounded-md border border-border p-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-semibold">{item.symbol}</span>
        <Badge variant={item.signalType === 'BUY' ? 'default' : 'destructive'} className="font-mono">
          {item.signalType}
        </Badge>
        <Badge variant={statusVariant(item.status)}>{item.status.replace(/_/g, ' ')}</Badge>
        <span className="ml-auto text-xs text-muted-foreground">
          Updated {fmtDate(item.filledAt ?? item.submittedAt ?? item.detectedAt)}
        </span>
      </div>
      <div className="mt-3 grid gap-2 text-sm md:grid-cols-2">
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Candidate</span>
            <code className="rounded bg-muted px-2 py-0.5 text-xs">{compactId(item.id)}</code>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Instruction</span>
            <code className="rounded bg-muted px-2 py-0.5 text-xs">{compactId(item.executionInstructionId)}</code>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Trade</span>
            <code className="rounded bg-muted px-2 py-0.5 text-xs">{compactId(item.tradeId)}</code>
          </div>
        </div>
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Approval</span>
            <span>{item.latestApproval?.decision ?? '-'}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Decided</span>
            <span>{fmtDate(item.latestApproval?.decidedAt)}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-muted-foreground">Provenance</span>
            <span className="truncate">{item.dataProvenance || '-'}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

export function BlotterPage() {
  const { data: executionActivity = [] } = useQuery({
    queryKey: ['blotter-execution-activity'],
    queryFn: async () => {
      const [approved, submitted, filled] = await Promise.all([
        candidatesService.list({ status: 'approved', limit: 6 }),
        candidatesService.list({ status: 'submitted', limit: 6 }),
        candidatesService.list({ status: 'filled', limit: 6 }),
      ]);
      return [...approved, ...submitted, ...filled]
        .sort((a, b) => {
          const left = new Date(a.filledAt ?? a.submittedAt ?? a.detectedAt).getTime();
          const right = new Date(b.filledAt ?? b.submittedAt ?? b.detectedAt).getTime();
          return right - left;
        })
        .slice(0, 8);
    },
    refetchInterval: 30_000,
  });

  const counts = useMemo(
    () => ({
      approved: executionActivity.filter((item) => item.status === 'approved').length,
      submitted: executionActivity.filter((item) => item.status === 'submitted').length,
      filled: executionActivity.filter((item) => item.status === 'filled').length,
    }),
    [executionActivity]
  );

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Blotter
        <HelpHint text="Execution history and order activity log." />
      </h1>
      <p className="text-sm text-muted-foreground">Review recent orders and their status.</p>

      <div className="grid gap-3 md:grid-cols-3">
        <SummaryCard label="Approved" value={counts.approved} hint="Candidates approved and waiting for execution progress." />
        <SummaryCard label="Submitted" value={counts.submitted} hint="Candidates with instructions sent into the execution stack." />
        <SummaryCard label="Filled" value={counts.filled} hint="Candidates with trade linkage and fill confirmation." />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Paper Execution Chain</CardTitle>
          <p className="text-sm text-muted-foreground">
            Links approved candidates to instructions and resulting trades so the blotter is not just broker-order history.
          </p>
        </CardHeader>
        <CardContent className="space-y-3">
          {executionActivity.length === 0 ? (
            <p className="text-sm text-muted-foreground">No recent approved, submitted, or filled candidates.</p>
          ) : (
            executionActivity.map((item) => <ChainRow key={`${item.status}-${item.id}`} item={item} />)
          )}
        </CardContent>
      </Card>

      <TradeBlotterPanel isOpen onToggle={() => undefined} />
    </div>
  );
}
