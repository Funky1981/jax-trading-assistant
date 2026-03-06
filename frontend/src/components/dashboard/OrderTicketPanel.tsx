import { useState } from 'react';
import { Receipt } from 'lucide-react';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Button } from '@/components/ui/button';
import { DataSourceBadge } from '@/components/ui/DataSourceBadge';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { OrderSide, OrderType, useCreateOrder } from '@/hooks/useOrders';
import { useMarketDataStatus } from '@/hooks/useMarketDataStatus';

interface OrderTicketPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

function formatMessage(result: unknown) {
  if (result && typeof result === 'object' && 'message' in result && typeof result.message === 'string') {
    return result.message;
  }
  return 'Order submitted successfully.';
}

export function OrderTicketPanel({ isOpen, onToggle }: OrderTicketPanelProps) {
  const [symbol, setSymbol] = useState('');
  const [side, setSide] = useState<OrderSide>('buy');
  const [orderType, setOrderType] = useState<OrderType>('market');
  const [quantity, setQuantity] = useState('');
  const [price, setPrice] = useState('');
  const [entryStopPrice, setEntryStopPrice] = useState('');
  const [stopLossPrice, setStopLossPrice] = useState('');
  const [takeProfitPrice, setTakeProfitPrice] = useState('');

  const createOrder = useCreateOrder();
  const { data: marketDataStatus, isError: marketStatusError } = useMarketDataStatus();

  const hasProtection = Boolean(stopLossPrice || takeProfitPrice);

  const resetForm = () => {
    setSymbol('');
    setQuantity('');
    setPrice('');
    setEntryStopPrice('');
    setStopLossPrice('');
    setTakeProfitPrice('');
  };

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();

    if (!symbol || !quantity) return;

    createOrder.mutate(
      {
        symbol: symbol.toUpperCase(),
        side,
        type: orderType,
        quantity: parseInt(quantity, 10),
        price: orderType === 'limit' ? (price ? parseFloat(price) : undefined) : undefined,
        stopPrice: orderType === 'stop' ? (entryStopPrice ? parseFloat(entryStopPrice) : undefined) : undefined,
        stopLossPrice: stopLossPrice ? parseFloat(stopLossPrice) : undefined,
        takeProfitPrice: takeProfitPrice ? parseFloat(takeProfitPrice) : undefined,
      },
      {
        onSuccess: () => resetForm(),
      }
    );
  };

  const summary = (
    <div className="flex items-center gap-2 text-xs">
      <DataSourceBadge
        marketDataMode={marketDataStatus?.marketDataMode}
        paperTrading={marketDataStatus?.paperTrading}
        isError={marketStatusError}
      />
      <span>{hasProtection ? 'Entry with protection' : 'Quick order entry'}</span>
    </div>
  );

  return (
    <CollapsiblePanel
      title="Order Ticket"
      icon={<Receipt className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {marketDataStatus ? (
          <div className="rounded-md border border-border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
            Orders are submitted with {marketDataStatus.paperTrading ? 'paper trading' : 'live trading'} enabled while quotes are currently in {marketDataStatus.marketDataMode} mode.
          </div>
        ) : null}

        <div className="rounded-md border border-border bg-card px-3 py-3 text-xs text-muted-foreground">
          <p className="font-semibold uppercase tracking-wide text-foreground">Operator Workflow</p>
          <p className="mt-2">
            1. Enter a market or limit entry. 2. Add a stop loss and optional take profit to submit attached protection. 3. Use the blotter to cancel working orders. 4. Use Positions to close or re-protect live exposure.
          </p>
        </div>

        <div className="space-y-2">
          <label htmlFor="order-ticket-symbol" className="text-sm font-medium">Symbol</label>
          <Input
            id="order-ticket-symbol"
            name="symbol"
            placeholder="AAPL"
            value={symbol}
            onChange={(event) => setSymbol(event.target.value.toUpperCase())}
            className="font-mono"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <Button
            type="button"
            variant={side === 'buy' ? 'default' : 'outline'}
            onClick={() => setSide('buy')}
            className={side === 'buy' ? 'bg-success hover:bg-success/90' : ''}
          >
            Buy
          </Button>
          <Button
            type="button"
            variant={side === 'sell' ? 'default' : 'outline'}
            onClick={() => setSide('sell')}
            className={side === 'sell' ? 'bg-destructive hover:bg-destructive/90' : ''}
          >
            Sell
          </Button>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <label htmlFor="order-ticket-type" className="text-sm font-medium">Entry Type</label>
            <Select name="orderType" value={orderType} onValueChange={(value) => setOrderType(value as OrderType)}>
              <SelectTrigger id="order-ticket-type" aria-label="Order type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="market">Market</SelectItem>
                <SelectItem value="limit">Limit</SelectItem>
                <SelectItem value="stop">Stop Entry</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label htmlFor="order-ticket-quantity" className="text-sm font-medium">Quantity</label>
            <Input
              id="order-ticket-quantity"
              name="quantity"
              type="number"
              placeholder="100"
              value={quantity}
              onChange={(event) => setQuantity(event.target.value)}
              min="1"
            />
          </div>
        </div>

        {orderType === 'limit' ? (
          <div className="space-y-2">
            <label htmlFor="order-ticket-limit-price" className="text-sm font-medium">Limit Price</label>
            <Input
              id="order-ticket-limit-price"
              name="limitPrice"
              type="number"
              placeholder="0.00"
              value={price}
              onChange={(event) => setPrice(event.target.value)}
              step="0.01"
              min="0"
            />
          </div>
        ) : null}

        {orderType === 'stop' ? (
          <div className="space-y-2">
            <label htmlFor="order-ticket-entry-stop-price" className="text-sm font-medium">Entry Stop Price</label>
            <Input
              id="order-ticket-entry-stop-price"
              name="entryStopPrice"
              type="number"
              placeholder="0.00"
              value={entryStopPrice}
              onChange={(event) => setEntryStopPrice(event.target.value)}
              step="0.01"
              min="0"
            />
            <p className="text-xs text-muted-foreground">
              Stop-entry orders are submitted without attached stop loss or take profit. Use Positions after fill if you need protection.
            </p>
          </div>
        ) : null}

        {orderType !== 'stop' ? (
          <div className="space-y-3 rounded-md border border-border bg-muted/20 p-3">
            <div>
              <p className="text-sm font-medium">Attached Protection</p>
              <p className="text-xs text-muted-foreground">
                Add a stop loss and optional take profit to submit a bracket entry.
              </p>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <label htmlFor="order-ticket-stop-loss" className="text-sm font-medium">Stop Loss</label>
                <Input
                  id="order-ticket-stop-loss"
                  name="stopLossPrice"
                  type="number"
                  placeholder="Optional"
                  value={stopLossPrice}
                  onChange={(event) => setStopLossPrice(event.target.value)}
                  step="0.01"
                  min="0"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="order-ticket-take-profit" className="text-sm font-medium">Take Profit</label>
                <Input
                  id="order-ticket-take-profit"
                  name="takeProfitPrice"
                  type="number"
                  placeholder="Optional"
                  value={takeProfitPrice}
                  onChange={(event) => setTakeProfitPrice(event.target.value)}
                  step="0.01"
                  min="0"
                />
              </div>
            </div>
          </div>
        ) : null}

        {createOrder.error ? (
          <p className="text-sm text-destructive">{createOrder.error.message}</p>
        ) : null}

        {createOrder.data ? (
          <p className="text-sm text-success">{formatMessage(createOrder.data)}</p>
        ) : null}

        <Button
          type="submit"
          className="w-full"
          disabled={
            !symbol ||
            !quantity ||
            (orderType === 'limit' && !price) ||
            (orderType === 'stop' && !entryStopPrice) ||
            createOrder.isPending
          }
        >
          {createOrder.isPending
            ? 'Submitting...'
            : hasProtection
              ? `Submit ${side.toUpperCase()} Bracket`
              : `Submit ${side.toUpperCase()} Order`}
        </Button>
      </form>
    </CollapsiblePanel>
  );
}
