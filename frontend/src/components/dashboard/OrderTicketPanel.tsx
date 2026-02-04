import { useState } from 'react';
import { Receipt } from 'lucide-react';
import { CollapsiblePanel } from './CollapsiblePanel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useCreateOrder, OrderSide, OrderType } from '@/hooks/useOrders';

interface OrderTicketPanelProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function OrderTicketPanel({ isOpen, onToggle }: OrderTicketPanelProps) {
  const [symbol, setSymbol] = useState('');
  const [side, setSide] = useState<OrderSide>('buy');
  const [orderType, setOrderType] = useState<OrderType>('market');
  const [quantity, setQuantity] = useState('');
  const [price, setPrice] = useState('');
  
  const createOrder = useCreateOrder();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!symbol || !quantity) return;
    
    createOrder.mutate({
      symbol: symbol.toUpperCase(),
      side,
      type: orderType,
      quantity: parseInt(quantity, 10),
      price: price ? parseFloat(price) : undefined,
    });
    
    // Reset form
    setSymbol('');
    setQuantity('');
    setPrice('');
  };

  const summary = <span>Quick order entry</span>;

  return (
    <CollapsiblePanel
      title="Order Ticket"
      icon={<Receipt className="h-4 w-4" />}
      summary={summary}
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Symbol */}
        <div className="space-y-2">
          <label className="text-sm font-medium">Symbol</label>
          <Input
            placeholder="AAPL"
            value={symbol}
            onChange={(e) => setSymbol(e.target.value.toUpperCase())}
            className="font-mono"
          />
        </div>

        {/* Side */}
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

        {/* Order Type */}
        <div className="space-y-2">
          <label className="text-sm font-medium">Order Type</label>
          <Select value={orderType} onValueChange={(v) => setOrderType(v as OrderType)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="market">Market</SelectItem>
              <SelectItem value="limit">Limit</SelectItem>
              <SelectItem value="stop">Stop</SelectItem>
              <SelectItem value="stop_limit">Stop Limit</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Quantity */}
        <div className="space-y-2">
          <label className="text-sm font-medium">Quantity</label>
          <Input
            type="number"
            placeholder="100"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            min="1"
          />
        </div>

        {/* Price (for limit orders) */}
        {(orderType === 'limit' || orderType === 'stop_limit') && (
          <div className="space-y-2">
            <label className="text-sm font-medium">Limit Price</label>
            <Input
              type="number"
              placeholder="0.00"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              step="0.01"
              min="0"
            />
          </div>
        )}

        {/* Submit */}
        <Button
          type="submit"
          className="w-full"
          disabled={!symbol || !quantity || createOrder.isPending}
        >
          {createOrder.isPending ? 'Submitting...' : `Submit ${side.toUpperCase()} Order`}
        </Button>
      </form>
    </CollapsiblePanel>
  );
}
