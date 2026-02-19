import { useState } from 'react';
import { PrimaryButton } from '../primitives/PrimaryButton';
import { SelectInput } from '../primitives/SelectInput';
import { TextInput } from '../primitives/TextInput';
import { tokens } from '../../styles/tokens';
import type { OrderDraft } from '../../domain/models';
import type { Side } from '../../domain/types';

const sideOptions = [
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
];

interface OrderTicketProps {
  symbol: string;
  defaultPrice?: number;
  onSubmit?: (payload: OrderDraft) => void;
}

export function OrderTicket({ symbol, defaultPrice, onSubmit }: OrderTicketProps) {
  const [side, setSide] = useState<Side>('buy');
  const [quantity, setQuantity] = useState(100);
  const [price, setPrice] = useState(() => defaultPrice ?? 0);

  return (
    <div
      className="space-y-4 p-6 rounded-md border border-border bg-card"
    >
      <h3 className="text-sm font-medium">Order Ticket</h3>
      <p className="text-sm text-muted-foreground">
        {symbol}
      </p>
      <SelectInput
        value={side}
        onValueChange={(value) => setSide(value as Side)}
        options={sideOptions}
        placeholder="Select side"
      />
      <TextInput
        label="Quantity"
        type="number"
        value={quantity}
        onChange={(event) => setQuantity(Number(event.target.value))}
      />
      <TextInput
        label="Limit Price"
        type="number"
        value={price}
        onChange={(event) => setPrice(Number(event.target.value))}
      />
      <PrimaryButton
        onClick={() => onSubmit?.({ symbol, side, quantity, price })}
        disabled={!quantity}
      >
        Place Order
      </PrimaryButton>
    </div>
  );
}
