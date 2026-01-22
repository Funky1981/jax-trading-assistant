import { Stack, Typography } from '@mui/material';
import { OrderTicket } from '../components';

export function OrderTicketPage() {
  return (
    <Stack spacing={2} maxWidth={480}>
      <Typography variant="h4">Order Ticket</Typography>
      <Typography variant="body2" color="text.secondary">
        Place orders quickly with pre-filled defaults.
      </Typography>
      <OrderTicket symbol="AAPL" />
    </Stack>
  );
}
