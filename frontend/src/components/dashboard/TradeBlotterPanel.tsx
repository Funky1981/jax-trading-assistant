import { useMemo, useState } from 'react';
import { ArrowUpDown, FileText, Filter } from 'lucide-react';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  SortingState,
  useReactTable,
} from '@tanstack/react-table';
import { Order, OrderStatus, useCancelOrder, useOrdersSummary } from '@/hooks/useOrders';
import { useTradingPilotStatus } from '@/hooks/useTradingPilotStatus';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { PilotStatusBanner } from '@/components/ui/PilotStatusBanner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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

function getWorkflowLabel(order: Order) {
  switch (order.workflow) {
    case 'close':
      return 'Close';
    case 'protect':
      return 'Protect';
    case 'entry':
      return 'Entry';
    default:
      return order.source === 'broker' ? 'Broker' : 'Strategy';
  }
}

export function TradeBlotterPanel({ isOpen, onToggle }: TradeBlotterPanelProps) {
  const { data: summary, orders, isLoading } = useOrdersSummary();
  const { data: pilotStatus } = useTradingPilotStatus();
  const cancelOrder = useCancelOrder();
  const [sorting, setSorting] = useState<SortingState>([{ id: 'createdAt', desc: true }]);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [pendingCancelOrder, setPendingCancelOrder] = useState<Order | null>(null);
  const [cancelConfirmed, setCancelConfirmed] = useState(false);

  const filteredOrders = useMemo(() => {
    if (!orders) return [];
    if (statusFilter === 'all') return orders;
    return orders.filter((order) => order.status === statusFilter);
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
      columnHelper.accessor('source', {
        header: 'Source',
        cell: (info) => {
          const order = info.row.original;
          return (
            <div className="flex flex-col gap-1">
              <Badge variant={order.source === 'broker' ? 'secondary' : 'outline'} className="w-fit text-xs">
                {order.source === 'broker' ? 'Broker' : 'Strategy'}
              </Badge>
              <span className="text-[11px] text-muted-foreground">{getWorkflowLabel(order)}</span>
            </div>
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
      columnHelper.display({
        id: 'actions',
        header: 'Actions',
        cell: ({ row }) => (
          row.original.canCancel ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={cancelOrder.isPending || pilotStatus?.readOnly === true}
              onClick={() => {
                setCancelConfirmed(false);
                setPendingCancelOrder(row.original);
              }}
            >
              Cancel
            </Button>
          ) : (
            <span className="text-xs text-muted-foreground">Read only</span>
          )
        ),
      }),
    ],
    [cancelOrder, pilotStatus?.readOnly]
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
      {summary.lastFill ? ` • Last fill: ${summary.lastFill.symbol}` : ''}
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
        {pilotStatus ? (
          <PilotStatusBanner
            title={
              pilotStatus.readOnly
                ? 'Order cancellation is disabled while the pilot is in read-only mode.'
                : 'Working broker orders require IB/TWS confirmation before cancellation.'
            }
            readOnly={pilotStatus.readOnly}
            reasons={pilotStatus.reasons}
            compact
          />
        ) : null}

        <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
          Broker orders created from this UI can be cancelled here while they are still working. Strategy-sourced history stays visible for context but is read only.
        </div>

        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="h-8 w-32" aria-label="Filter orders by status">
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

        {cancelOrder.error ? (
          <p className="text-sm text-destructive">{cancelOrder.error.message}</p>
        ) : null}

        <div className="overflow-x-auto rounded-md border border-border">
          <Table className="min-w-[980px]">
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id} className="whitespace-nowrap">
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
                      <TableCell key={cell.id} className="whitespace-nowrap">
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

      <Dialog open={Boolean(pendingCancelOrder)} onOpenChange={(open) => {
        if (!open) {
          setPendingCancelOrder(null);
          setCancelConfirmed(false);
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Cancel</DialogTitle>
            <DialogDescription>
              Confirm the working order in IB/TWS before sending a cancel request from the pilot UI.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 text-sm">
            <div className="rounded-md border border-border bg-muted/20 px-3 py-3">
              <p className="font-mono text-foreground">
                {pendingCancelOrder?.symbol} {pendingCancelOrder?.quantity} {pendingCancelOrder?.side.toUpperCase()}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Order #{pendingCancelOrder?.brokerOrderId} • {pendingCancelOrder?.type.toUpperCase()} • {pendingCancelOrder?.status.toUpperCase()}
              </p>
            </div>

            <label className="flex items-start gap-2 text-sm text-foreground">
              <input
                type="checkbox"
                className="mt-1"
                checked={cancelConfirmed}
                onChange={(event) => setCancelConfirmed(event.target.checked)}
              />
              <span>I confirmed in IB/TWS that this working order should be cancelled.</span>
            </label>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setPendingCancelOrder(null)}>
              Keep Order
            </Button>
            <Button
              type="button"
              disabled={!cancelConfirmed || cancelOrder.isPending || !pendingCancelOrder}
              onClick={() => {
                if (!pendingCancelOrder) return;
                cancelOrder.mutate(pendingCancelOrder, {
                  onSuccess: () => {
                    setPendingCancelOrder(null);
                    setCancelConfirmed(false);
                  },
                });
              }}
            >
              {cancelOrder.isPending ? 'Cancelling...' : 'Submit Cancel'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </CollapsiblePanel>
  );
}
