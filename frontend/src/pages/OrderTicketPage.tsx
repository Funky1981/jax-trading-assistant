import { Stack, Typography } from '@mui/material';
import { OrderTicket } from '../components';
import { useDomain } from '../domain/store';
import { selectTickBySymbol } from '../domain/selectors';

export function OrderTicketPage() {
  const { state, actions } = useDomain();
  const primarySymbol = 'AAPL';
  const tick = selectTickBySymbol(state, primarySymbol);

  return (
    <Stack spacing={2} maxWidth={480}>
      <Typography variant="h4">Order Ticket</Typography>
      <Typography variant="body2" color="text.secondary">
        Place orders quickly with pre-filled defaults.
      </Typography>
      <OrderTicket
        symbol={primarySymbol}
        defaultPrice={tick?.price}
        onSubmit={actions.placeOrder}
      />
    </Stack>
  );
}
