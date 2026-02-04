import { useMemo, useState } from 'react';
import { Activity, ArrowUpDown, Filter } from 'lucide-react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  createColumnHelper,
  SortingState,
} from '@tanstack/react-table';
import { useMetricsSummary, MetricEvent } from '@/hooks/useMetrics';
import { CollapsiblePanel, StatusDot } from './CollapsiblePanel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { formatTime } from '@/lib/utils';

interface MetricsPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const columnHelper = createColumnHelper<MetricEvent>();

export function MetricsPanel({ isOpen, onToggle }: MetricsPanelProps) {
  const { data: summary, metrics, isLoading } = useMetricsSummary();
  const [sorting, setSorting] = useState<SortingState>([{ id: 'timestamp', desc: true }]);
  const [sourceFilter, setSourceFilter] = useState<string>('all');

  const sources = useMemo(() => {
    if (!metrics) return [];
    return [...new Set(metrics.map((m) => m.source))];
  }, [metrics]);

  const filteredMetrics = useMemo(() => {
    if (!metrics) return [];
    if (sourceFilter === 'all') return metrics;
    return metrics.filter((m) => m.source === sourceFilter);
  }, [metrics, sourceFilter]);

  const columns = useMemo(
    () => [
      columnHelper.accessor('timestamp', {
        header: ({ column }) => (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-3 h-8"
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
          >
            Time
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        ),
        cell: (info) => (
          <span className="text-xs text-muted-foreground">
            {formatTime(info.getValue())}
          </span>
        ),
      }),
      columnHelper.accessor('event', {
        header: 'Event',
        cell: (info) => (
          <span className="font-mono text-xs">{info.getValue()}</span>
        ),
      }),
      columnHelper.accessor('source', {
        header: 'Source',
        cell: (info) => (
          <Badge variant="outline" className="text-xs">
            {info.getValue()}
          </Badge>
        ),
      }),
      columnHelper.accessor('duration', {
        header: ({ column }) => (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-3 h-8"
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
          >
            Duration
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        ),
        cell: (info) => (
          <span className="font-mono text-xs">{info.getValue()}ms</span>
        ),
      }),
      columnHelper.accessor('status', {
        header: 'Status',
        cell: (info) => (
          <div className="flex items-center gap-2">
            <StatusDot status={info.getValue()} />
            <span className="text-xs capitalize">{info.getValue()}</span>
          </div>
        ),
      }),
    ],
    []
  );

  const table = useReactTable({
    data: filteredMetrics,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  });

  const summaryText = summary ? (
    <span>
      {summary.lastEvent?.event} • {formatTime(summary.lastEvent?.timestamp || Date.now())}
    </span>
  ) : null;

  return (
    <CollapsiblePanel
      title="System Metrics"
      icon={<Activity className="h-4 w-4" />}
      summary={summaryText}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-4">
        {/* Summary Stats */}
        {summary && (
          <div className="grid grid-cols-4 gap-4 text-center">
            <div>
              <p className="text-xs text-muted-foreground">Total Events</p>
              <p className="font-mono font-semibold">{summary.total}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Success</p>
              <p className="font-mono font-semibold text-success">
                {summary.successCount}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Warnings</p>
              <p className="font-mono font-semibold text-warning">
                {summary.warningCount}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Errors</p>
              <p className="font-mono font-semibold text-destructive">
                {summary.errorCount}
              </p>
            </div>
          </div>
        )}

        {/* Filter */}
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <Select value={sourceFilter} onValueChange={setSourceFilter}>
            <SelectTrigger className="w-36 h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Sources</SelectItem>
              {sources.map((source) => (
                <SelectItem key={source} value={source}>
                  {source}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <span className="text-xs text-muted-foreground">
            {filteredMetrics.length} events
          </span>
        </div>

        {/* Table */}
        <div className="rounded-md border border-border max-h-64 overflow-auto">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext()
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="h-24 text-center"
                  >
                    No metrics found.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </CollapsiblePanel>
  );
}
