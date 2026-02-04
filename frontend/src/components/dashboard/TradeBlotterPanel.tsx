import { useMemo, useState } from 'react';
import { FileText, ArrowUpDown, Filter } from 'lucide-react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  flexRender,
  createColumnHelper,
  SortingState,
} from '@tanstack/react-table';
import { useOrdersSummary, Order, OrderStatus } from '@/hooks/useOrders';
import { CollapsiblePanel } from './CollapsiblePanel';
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
import { cn, formatCurrency, formatTime } from '@/lib/utils';

interface TradeBlotterPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const columnHelper = createColumnHelper<Order>();

const statusVariantMap: Record<OrderStatus, 'default' | 'secondary' | 'success' | 'destructive' | 'warning'> = {
  pending: 'warning',
  filled: 'success',
  partial: 'secondary',
  cancelled: 'default',
  rejected: 'destructive',
};

export function TradeBlotterPanel({ isOpen, onToggle }: TradeBlotterPanelProps) {
  const { data: summary, orders, isLoading } = useOrdersSummary();
  const [sorting, setSorting] = useState<SortingState>([{ id: 'createdAt', desc: true }]);
  const [statusFilter, setStatusFilter] = useState<string>('all');

  const filteredOrders = useMemo(() => {
    if (!orders) return [];
    if (statusFilter === 'all') return orders;
    return orders.filter((o) => o.status === statusFilter);
  }, [orders, statusFilter]);

  const columns = useMemo(
    () => [
      columnHelper.accessor('createdAt', {
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
      columnHelper.accessor('symbol', {
        header: 'Symbol',
        cell: (info) => (
          <span className="font-mono font-semibold">{info.getValue()}</span>
        ),
      }),
      columnHelper.accessor('side', {
        header: 'Side',
        cell: (info) => {
          const value = info.getValue();
          return (
            <span
              className={cn(
                'font-semibold uppercase text-xs',
                value === 'buy' ? 'text-success' : 'text-destructive'
              )}
            >
              {value}
            </span>
          );
        },
      }),
      columnHelper.accessor('type', {
        header: 'Type',
        cell: (info) => (
          <span className="text-xs uppercase">{info.getValue().replace('_', ' ')}</span>
        ),
      }),
      columnHelper.accessor('quantity', {
        header: 'Qty',
        cell: (info) => <span className="font-mono">{info.getValue()}</span>,
      }),
      columnHelper.accessor('price', {
        header: 'Price',
        cell: (info) => {
          const value = info.getValue();
          return value ? (
            <span className="font-mono">{formatCurrency(value)}</span>
          ) : (
            <span className="text-muted-foreground">MKT</span>
          );
        },
      }),
      columnHelper.accessor('avgFillPrice', {
        header: 'Fill',
        cell: (info) => {
          const value = info.getValue();
          return value ? (
            <span className="font-mono">{formatCurrency(value)}</span>
          ) : (
            <span className="text-muted-foreground">-</span>
          );
        },
      }),
      columnHelper.accessor('status', {
        header: 'Status',
        cell: (info) => {
          const status = info.getValue();
          return (
            <Badge variant={statusVariantMap[status]} className="text-xs">
              {status}
            </Badge>
          );
        },
      }),
    ],
    []
  );

  const table = useReactTable({
    data: filteredOrders,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onSortingChange: setSorting,
    state: { sorting },
  });

  const summaryText = summary ? (
    <span>
      {summary.total} orders • {summary.pending} pending
      {summary.lastFill && ` • Last: ${summary.lastFill.symbol}`}
    </span>
  ) : null;

  return (
    <CollapsiblePanel
      title="Trade Blotter"
      icon={<FileText className="h-4 w-4" />}
      summary={summaryText}
      isOpen={isOpen}
      onToggle={onToggle}
      isLoading={isLoading}
    >
      <div className="space-y-4">
        {/* Filter */}
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-32 h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="filled">Filled</SelectItem>
              <SelectItem value="cancelled">Cancelled</SelectItem>
              <SelectItem value="rejected">Rejected</SelectItem>
            </SelectContent>
          </Select>
          <span className="text-xs text-muted-foreground">
            {filteredOrders.length} orders
          </span>
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
                    No orders found.
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
