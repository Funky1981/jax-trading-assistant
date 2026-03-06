import { useMemo, useState } from 'react';
import { ArrowUpDown, Briefcase, Shield, XCircle } from 'lucide-react';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  SortingState,
  useReactTable,
} from '@tanstack/react-table';
import {
  ClosePositionRequest,
  Position,
  ProtectPositionRequest,
  useClosePosition,
  usePositionsSummary,
  useProtectPosition,
} from '@/hooks/usePositions';
import { useMarketDataStatus } from '@/hooks/useMarketDataStatus';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Button } from '@/components/ui/button';
import { DataSourceBadge } from '@/components/ui/DataSourceBadge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
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
import { cn, formatCurrency, formatPercent } from '@/lib/utils';

interface PositionsPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

const columnHelper = createColumnHelper<Position>();

function getDefaultStop(position: Position) {
  const base = position.marketPrice || position.avgPrice || 0;
  if (base <= 0) return '';
  const multiplier = position.quantity > 0 ? 0.98 : 1.02;
  return (base * multiplier).toFixed(2);
}

function getDefaultTarget(position: Position) {
  const base = position.marketPrice || position.avgPrice || 0;
  if (base <= 0) return '';
  const multiplier = position.quantity > 0 ? 1.04 : 0.96;
  return (base * multiplier).toFixed(2);
}

export function PositionsPanel({ isOpen, onToggle }: PositionsPanelProps) {
  const { data: summary, positions, isLoading, isError } = usePositionsSummary();
  const { data: marketDataStatus } = useMarketDataStatus();
  const closePosition = useClosePosition();
  const protectPosition = useProtectPosition();
  const [sorting, setSorting] = useState<SortingState>([]);

  const [closeTarget, setCloseTarget] = useState<Position | null>(null);
  const [closeQuantity, setCloseQuantity] = useState('');
  const [closeOrderType, setCloseOrderType] = useState<'MKT' | 'LMT'>('MKT');
  const [closeLimitPrice, setCloseLimitPrice] = useState('');

  const [protectTarget, setProtectTarget] = useState<Position | null>(null);
  const [protectQuantity, setProtectQuantity] = useState('');
  const [stopLoss, setStopLoss] = useState('');
  const [takeProfit, setTakeProfit] = useState('');

  const handleCloseSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!closeTarget) return;

    const request: ClosePositionRequest = {
      symbol: closeTarget.symbol,
      quantity: parseInt(closeQuantity, 10),
      orderType: closeOrderType,
      limitPrice: closeOrderType === 'LMT' ? parseFloat(closeLimitPrice) : undefined,
    };

    closePosition.mutate(request, {
      onSuccess: () => setCloseTarget(null),
    });
  };

  const handleProtectSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!protectTarget) return;

    const request: ProtectPositionRequest = {
      symbol: protectTarget.symbol,
      quantity: parseInt(protectQuantity, 10),
      stopLoss: parseFloat(stopLoss),
      takeProfit: takeProfit ? parseFloat(takeProfit) : undefined,
      replaceExisting: true,
    };

    protectPosition.mutate(request, {
      onSuccess: () => setProtectTarget(null),
    });
  };

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
            P&amp;L
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
      columnHelper.display({
        id: 'actions',
        header: 'Actions',
        cell: ({ row }) => (
          <div className="flex items-center justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                protectPosition.reset();
                setProtectTarget(row.original);
                setProtectQuantity(String(Math.abs(row.original.quantity)));
                setStopLoss(getDefaultStop(row.original));
                setTakeProfit(getDefaultTarget(row.original));
              }}
            >
              <Shield className="h-3.5 w-3.5" />
              Protect
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => {
                closePosition.reset();
                setCloseTarget(row.original);
                setCloseQuantity(String(Math.abs(row.original.quantity)));
                setCloseOrderType('MKT');
                setCloseLimitPrice(
                  row.original.marketPrice ? row.original.marketPrice.toFixed(2) : ''
                );
              }}
            >
              <XCircle className="h-3.5 w-3.5" />
              Close
            </Button>
          </div>
        ),
      }),
    ],
    [closePosition, protectPosition]
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
    <div className="flex items-center gap-3 text-xs">
      <DataSourceBadge
        marketDataMode={marketDataStatus?.marketDataMode}
        paperTrading={marketDataStatus?.paperTrading}
        isError={isError}
      />
      <span className="font-mono">{summary.positionCount} positions</span>
      <Separator orientation="vertical" className="h-4" />
      <span className="font-mono">{formatCurrency(summary.totalValue)}</span>
      <Separator orientation="vertical" className="h-4" />
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
    <>
      <CollapsiblePanel
        title="Positions"
        icon={<Briefcase className="h-4 w-4" />}
        summary={summaryText}
        isOpen={isOpen}
        onToggle={onToggle}
        isLoading={isLoading}
      >
        <div className="mb-4 rounded-md border border-border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
          Use <span className="font-semibold text-foreground">Protect</span> to replace the current stop / target set by this UI, or <span className="font-semibold text-foreground">Close</span> to flatten all or part of a position with a market or limit exit.
        </div>

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

        {summary && (
          <div className="mt-4 flex justify-between text-sm">
            <span className="text-muted-foreground">
              {summary.positionCount} positions
            </span>
            <div className="flex gap-4">
              <span>
                Total Value:{' '}
                <span className="font-mono font-semibold">{formatCurrency(summary.totalValue)}</span>
              </span>
              <span>
                Total P&amp;L:{' '}
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

      <Dialog open={Boolean(closeTarget)} onOpenChange={(open) => !open && setCloseTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Close Position</DialogTitle>
            <DialogDescription>
              Submit an exit order for {closeTarget?.symbol}. Use market to flatten immediately or limit to work a price.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleCloseSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label htmlFor="close-position-quantity" className="text-sm font-medium">Quantity</label>
                <Input
                  id="close-position-quantity"
                  value={closeQuantity}
                  onChange={(event) => setCloseQuantity(event.target.value)}
                  type="number"
                  min="1"
                  max={closeTarget ? String(Math.abs(closeTarget.quantity)) : undefined}
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="close-position-type" className="text-sm font-medium">Exit Type</label>
                <Select value={closeOrderType} onValueChange={(value) => setCloseOrderType(value as 'MKT' | 'LMT')}>
                  <SelectTrigger id="close-position-type" aria-label="Close position order type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="MKT">Market</SelectItem>
                    <SelectItem value="LMT">Limit</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {closeOrderType === 'LMT' ? (
              <div className="space-y-2">
                <label htmlFor="close-position-limit" className="text-sm font-medium">Limit Price</label>
                <Input
                  id="close-position-limit"
                  value={closeLimitPrice}
                  onChange={(event) => setCloseLimitPrice(event.target.value)}
                  type="number"
                  step="0.01"
                  min="0"
                />
              </div>
            ) : null}

            {closePosition.error ? (
              <p className="text-sm text-destructive">{closePosition.error.message}</p>
            ) : null}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCloseTarget(null)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="destructive"
                disabled={
                  closePosition.isPending ||
                  !closeQuantity ||
                  (closeOrderType === 'LMT' && !closeLimitPrice)
                }
              >
                {closePosition.isPending ? 'Submitting...' : 'Submit Close'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(protectTarget)} onOpenChange={(open) => !open && setProtectTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Protect Position</DialogTitle>
            <DialogDescription>
              Attach or replace broker-side exits for {protectTarget?.symbol}. A stop is required; take profit is optional.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleProtectSubmit} className="space-y-4">
            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <label htmlFor="protect-position-quantity" className="text-sm font-medium">Quantity</label>
                <Input
                  id="protect-position-quantity"
                  value={protectQuantity}
                  onChange={(event) => setProtectQuantity(event.target.value)}
                  type="number"
                  min="1"
                  max={protectTarget ? String(Math.abs(protectTarget.quantity)) : undefined}
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="protect-position-stop" className="text-sm font-medium">Stop Loss</label>
                <Input
                  id="protect-position-stop"
                  value={stopLoss}
                  onChange={(event) => setStopLoss(event.target.value)}
                  type="number"
                  step="0.01"
                  min="0"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="protect-position-target" className="text-sm font-medium">Take Profit</label>
                <Input
                  id="protect-position-target"
                  value={takeProfit}
                  onChange={(event) => setTakeProfit(event.target.value)}
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="Optional"
                />
              </div>
            </div>

            <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
              Existing UI-created stops and targets for this symbol are cancelled before the new protection is submitted.
            </div>

            {protectPosition.error ? (
              <p className="text-sm text-destructive">{protectPosition.error.message}</p>
            ) : null}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setProtectTarget(null)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="success"
                disabled={protectPosition.isPending || !protectQuantity || !stopLoss}
              >
                {protectPosition.isPending ? 'Submitting...' : 'Submit Protection'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
