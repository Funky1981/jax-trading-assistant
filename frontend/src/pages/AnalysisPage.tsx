import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { Download } from 'lucide-react';
import { backtestService } from '@/data/backtest-service';
import type { BacktestRunBySymbol, BacktestTrade } from '@/data/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

export function AnalysisPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const runId = searchParams.get('runId') ?? '';

  const runsQuery = useQuery({
    queryKey: ['analysis-runs-selector'],
    queryFn: () => backtestService.list({ limit: 200 }),
  });
  const runDetailQuery = useQuery({
    queryKey: ['analysis-run-detail', runId],
    queryFn: () => backtestService.get(runId),
    enabled: runId.length > 0,
  });
  const timelineQuery = useQuery({
    queryKey: ['analysis-run-timeline', runDetailQuery.data?.parentRunId],
    queryFn: () => backtestService.getRunTimeline(runDetailQuery.data?.parentRunId ?? ''),
    enabled: Boolean(runDetailQuery.data?.parentRunId),
  });

  const bySymbol = useMemo(() => {
    const detail = runDetailQuery.data;
    if (!detail) {
      return [] as BacktestRunBySymbol[];
    }
    if (detail.bySymbol && detail.bySymbol.length > 0) {
      return detail.bySymbol;
    }
    return aggregateBySymbol(detail.trades ?? []);
  }, [runDetailQuery.data]);

  const onDownloadCsv = () => {
    const detail = runDetailQuery.data;
    if (!detail || !detail.trades || detail.trades.length === 0) {
      return;
    }
    const header = ['symbol', 'side', 'entryPrice', 'exitPrice', 'quantity', 'pnl', 'pnlPct', 'openedAt', 'closedAt'];
    const rows = detail.trades.map((trade) =>
      [
        trade.symbol,
        trade.side,
        trade.entryPrice ?? '',
        trade.exitPrice ?? '',
        trade.quantity ?? '',
        trade.pnl ?? '',
        trade.pnlPct ?? '',
        trade.openedAt ?? '',
        trade.closedAt ?? '',
      ].join(',')
    );
    const csv = [header.join(','), ...rows].join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    const href = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = href;
    link.download = `backtest-${detail.runId}.csv`;
    link.click();
    URL.revokeObjectURL(href);
  };

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">RUN ANALYSIS</p>
        <h1 className="text-2xl font-bold md:text-3xl">Analysis</h1>
        <p className="text-muted-foreground mt-1">
          Inspect backtest metrics, symbol-level breakdown, trades, and run timeline.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Run Selector</CardTitle>
          <CardDescription>Select a run or open this page with `?runId=...`.</CardDescription>
        </CardHeader>
        <CardContent>
          <Select
            value={runId || 'none'}
            onValueChange={(value) => {
              if (value === 'none') {
                setSearchParams(new URLSearchParams());
                return;
              }
              const next = new URLSearchParams(searchParams);
              next.set('runId', value);
              setSearchParams(next);
            }}
          >
            <SelectTrigger>
              <SelectValue placeholder="Choose run ID" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">Select run</SelectItem>
              {(runsQuery.data ?? []).map((run) => (
                <SelectItem key={run.id} value={run.runId}>
                  {run.runId}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      {runDetailQuery.data && (
        <>
          <div className="grid gap-3 md:grid-cols-3 lg:grid-cols-6">
            <MetricCard title="Trades" value={numOrDash(runDetailQuery.data.stats.trades ?? runDetailQuery.data.stats.totalTrades)} />
            <MetricCard title="Win Rate" value={pctOrDash(runDetailQuery.data.stats.winRate)} />
            <MetricCard title="Avg R" value={numOrDash(runDetailQuery.data.stats.avgR)} />
            <MetricCard title="Max Drawdown" value={pctOrDash(runDetailQuery.data.stats.maxDrawdown)} />
            <MetricCard title="P/L" value={numOrDash(runDetailQuery.data.stats.pnl)} />
            <MetricCard title="Exposure" value={numOrDash(runDetailQuery.data.stats.exposure)} />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>By Symbol</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Symbol</TableHead>
                    <TableHead>Trades</TableHead>
                    <TableHead>Win Rate</TableHead>
                    <TableHead>P/L</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {bySymbol.map((row) => (
                    <TableRow key={row.symbol}>
                      <TableCell>{row.symbol}</TableCell>
                      <TableCell>{row.trades}</TableCell>
                      <TableCell>{pctOrDash(row.winRate)}</TableCell>
                      <TableCell>{numOrDash(row.pnl)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <CardTitle>Trades</CardTitle>
              <Button variant="outline" size="sm" onClick={onDownloadCsv}>
                <Download className="mr-1 h-4 w-4" />
                CSV
              </Button>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Symbol</TableHead>
                    <TableHead>Entry Time</TableHead>
                    <TableHead>Entry</TableHead>
                    <TableHead>Exit Time</TableHead>
                    <TableHead>Exit</TableHead>
                    <TableHead>P/L</TableHead>
                    <TableHead>R</TableHead>
                    <TableHead>Reason</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(runDetailQuery.data.trades ?? []).map((trade, idx) => (
                    <TableRow key={`${trade.symbol}-${trade.openedAt ?? idx}`}>
                      <TableCell>{trade.symbol}</TableCell>
                      <TableCell>{fmtDate(trade.openedAt)}</TableCell>
                      <TableCell>{numOrDash(trade.entryPrice)}</TableCell>
                      <TableCell>{fmtDate(trade.closedAt)}</TableCell>
                      <TableCell>{numOrDash(trade.exitPrice)}</TableCell>
                      <TableCell>{numOrDash(trade.pnl)}</TableCell>
                      <TableCell>{numOrDash(trade.metadata?.rMultiple)}</TableCell>
                      <TableCell>{stringOrDash(trade.metadata?.exitReason)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Timeline</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Timestamp</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Outcome</TableHead>
                    <TableHead>Message</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(timelineQuery.data ?? []).map((event) => (
                    <TableRow key={event.id}>
                      <TableCell>{fmtDate(event.ts)}</TableCell>
                      <TableCell>{event.category ?? '-'}</TableCell>
                      <TableCell>{event.action ?? '-'}</TableCell>
                      <TableCell>{event.outcome ?? '-'}</TableCell>
                      <TableCell>{event.message ?? '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function MetricCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{title}</CardDescription>
      </CardHeader>
      <CardContent>
        <p className="text-xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

function aggregateBySymbol(trades: BacktestTrade[]): BacktestRunBySymbol[] {
  const map = new Map<string, { trades: number; wins: number; pnl: number }>();
  for (const trade of trades) {
    const key = trade.symbol;
    const existing = map.get(key) ?? { trades: 0, wins: 0, pnl: 0 };
    existing.trades += 1;
    if (typeof trade.pnl === 'number' && trade.pnl > 0) {
      existing.wins += 1;
    }
    if (typeof trade.pnl === 'number') {
      existing.pnl += trade.pnl;
    }
    map.set(key, existing);
  }
  return Array.from(map.entries()).map(([symbol, value]) => ({
    symbol,
    trades: value.trades,
    winRate: value.trades > 0 ? value.wins / value.trades : 0,
    pnl: value.pnl,
  }));
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

function numOrDash(value: unknown): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  return value.toFixed(4);
}

function pctOrDash(value: unknown): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  const pct = value <= 1 ? value * 100 : value;
  return `${pct.toFixed(2)}%`;
}

function stringOrDash(value: unknown): string {
  return typeof value === 'string' && value.length > 0 ? value : '-';
}

