import { useMemo, useState } from 'react';
import { Briefcase, ArrowUpDown } from 'lucide-react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  createColumnHelper,
  SortingState,
} from '@tanstack/react-table';
import { usePositionsSummary, Position } from '@/hooks/usePositions';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { cn, formatCurrency, formatPercent } from '@/lib/utils';

interface PositionsPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const columnHelper = createColumnHelper<Position>();

export function PositionsPanel({ isOpen, onToggle }: PositionsPanelProps) {
  const { data: summary, positions, isLoading } = usePositionsSummary();
  const [sorting, setSorting] = useState<SortingState>([]);

  const columns = useMemo(
    () => [
      columnHelper.accessor('symbol', {
        header: 'Symbol',
        cell: (info) => (
          <span className="font-mono font-semibold">{info.getValue()}</span>
        ),
      }),
      columnHelper.accessor('quantity', {
        header: 'Qty',
        cell: (info) => <span className="font-mono">{info.getValue()}</span>,
      }),
      columnHelper.accessor('avgPrice', {
        header: 'Avg Price',
        cell: (info) => (
          <span className="font-mono">{formatCurrency(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor('marketPrice', {
        header: 'Mkt Price',
        cell: (info) => (
          <span className="font-mono">{formatCurrency(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor('marketValue', {
        header: ({ column }) => (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-3 h-8"
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
          >
            Value
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        ),
        cell: (info) => (
          <span className="font-mono">{formatCurrency(info.getValue())}</span>
        ),
      }),
      columnHelper.accessor('pnl', {
        header: ({ column }) => (
          <Button
            variant="ghost"
            size="sm"
            className="-ml-3 h-8"
            onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
          >
            P&L
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
              {formatCurrency(value)}
            </span>
          );
        },
      }),
      columnHelper.accessor('pnlPercent', {
        header: 'P&L %',
        cell: (info) => {
          const value = info.getValue();
          return (
            <span
              className={cn(
                'font-mono',
                value > 0 ? 'text-success' : value < 0 ? 'text-destructive' : ''
              )}
            >
              {formatPercent(value)}
            </span>
          );
        },
      }),
    ],
    []
  );

  const table = useReactTable({
    data: positions || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  });

  const summaryText = summary ? (
    <div className="flex items-center gap-3">
      <span>{formatCurrency(summary.totalValue)}</span>
      <span
        className={cn(
          'font-mono font-semibold',
          summary.totalPnl > 0
            ? 'text-success'
            : summary.totalPnl < 0
            ? 'text-destructive'
            : ''
        )}
      >
        {formatCurrency(summary.totalPnl)} ({formatPercent(summary.totalPnlPercent)})
      </span>
    </div>
  ) : null;

  return (
    <CollapsiblePanel
      title="Positions"
      icon={<Briefcase className="h-4 w-4" />}
      summary={summaryText}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
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
                  No open positions.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Totals */}
      {summary && (
        <div className="mt-4 flex justify-between text-sm">
          <span className="text-muted-foreground">
            {summary.positionCount} positions
          </span>
          <div className="flex gap-4">
            <span>
              Total Value: <span className="font-mono font-semibold">{formatCurrency(summary.totalValue)}</span>
            </span>
            <span>
              Total P&L:{' '}
              <span
                className={cn(
                  'font-mono font-semibold',
                  summary.totalPnl > 0
                    ? 'text-success'
                    : summary.totalPnl < 0
                    ? 'text-destructive'
                    : ''
                )}
              >
                {formatCurrency(summary.totalPnl)}
              </span>
            </span>
          </div>
        </div>
      )}
    </CollapsiblePanel>
  );
}
