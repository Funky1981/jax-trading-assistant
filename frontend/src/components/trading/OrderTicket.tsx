import { Stack, Typography } from '@mui/material';
import { useState } from 'react';
import { PrimaryButton } from '../primitives/PrimaryButton';
import { SelectInput } from '../primitives/SelectInput';
import { TextInput } from '../primitives/TextInput';
import { tokens } from '../../styles/tokens';

const sideOptions = [
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
];

interface OrderTicketProps {
  symbol: string;
  onSubmit?: (payload: { side: string; quantity: number; price: number }) => void;
}

export function OrderTicket({ symbol, onSubmit }: OrderTicketProps) {
  const [side, setSide] = useState('buy');
  const [quantity, setQuantity] = useState(100);
  const [price, setPrice] = useState(0);

  return (
    <Stack
      spacing={2}
      sx={{
        padding: tokens.spacing.lg,
        borderRadius: tokens.radius.md,
        border: `1px solid ${tokens.colors.border}`,
        backgroundColor: tokens.colors.surface,
      }}
    >
      <Typography variant="subtitle2">Order Ticket</Typography>
      <Typography variant="body2" color="text.secondary">
        {symbol}
      </Typography>
      <SelectInput
        label="Side"
        value={side}
        onChange={(event) => setSide(event.target.value)}
        options={sideOptions}
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
        onClick={() => onSubmit?.({ side, quantity, price })}
        disabled={!quantity}
      >
        Place Order
      </PrimaryButton>
    </Stack>
  );
}
