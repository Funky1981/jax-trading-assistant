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
import {
  useAddToWatchlist,
  useRemoveFromWatchlist,
  useWatchlistSummary,
  WatchlistItem,
} from '@/hooks/useWatchlist';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { DataSourceBadge } from '@/components/ui/DataSourceBadge';
import { Separator } from '@/components/ui/separator';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useMarketDataStatus } from '@/hooks/useMarketDataStatus';
import { useTradingPilotStatus } from '@/hooks/useTradingPilotStatus';
import { PilotStatusBanner } from '@/components/ui/PilotStatusBanner';
import { cn, formatCurrency, formatPercent } from '@/lib/utils';

interface WatchlistPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const columnHelper = createColumnHelper<WatchlistItem>();

export function WatchlistPanel({ isOpen, onToggle }: WatchlistPanelProps) {
  const { data: summary, watchlist, isLoading, isError } = useWatchlistSummary();
  const { data: marketDataStatus } = useMarketDataStatus();
  const { data: pilotStatus } = useTradingPilotStatus();
  const addToWatchlist = useAddToWatchlist();
  const removeFromWatchlist = useRemoveFromWatchlist();
  const [sorting, setSorting] = useState<SortingState>([]);
  const [newSymbol, setNewSymbol] = useState('');
  const tableData = useMemo(() => watchlist ?? [], [watchlist]);

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
          if (value === null) {
            return <span className="text-muted-foreground">--</span>;
          }
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
          if (value === null) {
            return <span className="text-muted-foreground">--</span>;
          }
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
        cell: ({ row }) => (
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            disabled={removeFromWatchlist.isPending}
            onClick={() => removeFromWatchlist.mutate(row.original.symbol)}
          >
            <Trash2 className="h-4 w-4 text-muted-foreground" />
          </Button>
        ),
      }),
    ],
    [removeFromWatchlist]
  );

  const table = useReactTable({
    data: tableData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
    autoResetPageIndex: false,
  });

  const pilotQuoteWarning = pilotStatus && !pilotStatus.brokerConnected ? (
    <PilotStatusBanner
      title="Watchlist prices are non-authoritative during the pilot."
      readOnly={pilotStatus.readOnly}
      reasons={pilotStatus.reasons}
      compact
    />
  ) : null;

  const summaryText = summary ? (
    <div className="flex items-center gap-3 text-xs">
      <DataSourceBadge
        marketDataMode={marketDataStatus?.marketDataMode}
        paperTrading={marketDataStatus?.paperTrading}
        isError={isError}
      />
      <span className="font-mono">{summary.count} symbols</span>
      <Separator orientation="vertical" className="h-4" />
      <span>
        {summary.topMover
          ? `Top: ${summary.topMover.symbol} (${formatPercent(summary.topMover.changePercent)})`
          : 'Change unavailable'}
      </span>
    </div>
  ) : null;

  const handleAddSymbol = () => {
    const symbol = newSymbol.trim().toUpperCase();
    if (!symbol) return;
    addToWatchlist.mutate(symbol, {
      onSuccess: () => {
        setNewSymbol('');
      },
    });
  };

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
        {pilotQuoteWarning}

        {/* Add Symbol */}
        <div className="flex gap-2">
          <Input
            id="watchlist-add-symbol"
            name="watchlistSymbol"
            aria-label="Add symbol to watchlist"
            placeholder="Add symbol..."
            value={newSymbol}
            onChange={(e) => setNewSymbol(e.target.value.toUpperCase())}
            className="h-9"
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                handleAddSymbol();
              }
            }}
          />
          <Button
            size="sm"
            className="h-9"
            onClick={handleAddSymbol}
            disabled={addToWatchlist.isPending || !newSymbol.trim()}
          >
            <Plus className="h-4 w-4 mr-1" />
            {addToWatchlist.isPending ? 'Adding...' : 'Add'}
          </Button>
        </div>

        {addToWatchlist.error ? (
          <p className="text-sm text-destructive">{addToWatchlist.error.message}</p>
        ) : null}

        {removeFromWatchlist.error ? (
          <p className="text-sm text-destructive">{removeFromWatchlist.error.message}</p>
        ) : null}

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
