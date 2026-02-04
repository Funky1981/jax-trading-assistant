import { useState, useMemo } from 'react';
import { Eye, ArrowUpDown, Plus, Trash2 } from 'lucide-react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  createColumnHelper,
  SortingState,
} from '@tanstack/react-table';
import { useWatchlistSummary, WatchlistItem } from '@/hooks/useWatchlist';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn, formatCurrency, formatPercent } from '@/lib/utils';

interface WatchlistPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const columnHelper = createColumnHelper<WatchlistItem>();

export function WatchlistPanel({ isOpen, onToggle }: WatchlistPanelProps) {
  const { data: summary } = useWatchlistSummary();
  const { watchlist, isLoading } = useWatchlistSummary();
  const [sorting, setSorting] = useState<SortingState>([]);
  const [newSymbol, setNewSymbol] = useState('');

  const columns = useMemo(
    () => [
      columnHelper.accessor('symbol', {
        header: 'Symbol',
        cell: (info) => (
          <span className="font-mono font-semibold">{info.getValue()}</span>
        ),
      }),
      columnHelper.accessor('price', {
        header: ({ column }) => (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-3 h-8"
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
          >
            Price
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        ),
        cell: (info) => (
          <span className="font-mono">{formatCurrency(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor('change', {
        header: 'Change',
        cell: (info) => {
          const value = info.getValue();
          return (
            <span
              className={cn(
                'font-mono',
                value > 0 ? 'text-success' : value < 0 ? 'text-destructive' : ''
              )}
            >
              {value >= 0 ? '+' : ''}{value.toFixed(2)}
            </span>
          );
        },
      }),
      columnHelper.accessor('changePercent', {
        header: ({ column }) => (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-3 h-8"
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
          >
            Change %
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        ),
        cell: (info) => {
          const value = info.getValue();
          return (
            <span
              className={cn(
                'font-mono font-semibold',
                value > 0 ? 'text-success' : value < 0 ? 'text-destructive' : ''
              )}
            >
              {formatPercent(value)}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: 'actions',
        cell: () => (
          <Button variant="ghost" size="icon" className="h-8 w-8">
            <Trash2 className="h-4 w-4 text-muted-foreground" />
          </Button>
        ),
      }),
    ],
    []
  );

  const table = useReactTable({
    data: watchlist || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  });

  const summaryText = summary ? (
    <span>
      {summary.count} symbols • Top: {summary.topMover.symbol} ({formatPercent(summary.topMover.changePercent)})
    </span>
  ) : null;

  return (
    <CollapsiblePanel
      title="Watchlist"
      icon={<Eye className="h-4 w-4" />}
      summary={summaryText}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-4">
        {/* Add Symbol */}
        <div className="flex gap-2">
          <Input
            placeholder="Add symbol..."
            value={newSymbol}
            onChange={(e) => setNewSymbol(e.target.value.toUpperCase())}
            className="h-9"
          />
          <Button size="sm" className="h-9">
            <Plus className="h-4 w-4 mr-1" />
            Add
          </Button>
        </div>

        {/* Table */}
        <div className="rounded-md border border-border">
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
                    No symbols in watchlist.
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
