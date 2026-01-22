import { Stack, Typography } from '@mui/material';
import { DataTable } from '../components';

const blotterRows = [
  { id: '1', symbol: 'AAPL', side: 'Buy', quantity: 150, price: 249.42, status: 'Filled' },
  { id: '2', symbol: 'MSFT', side: 'Sell', quantity: 50, price: 413.1, status: 'Open' },
  { id: '3', symbol: 'SPY', side: 'Buy', quantity: 200, price: 541.22, status: 'Cancelled' },
];

export function BlotterPage() {
  return (
    <Stack spacing={2}>
      <Typography variant="h4">Blotter</Typography>
      <Typography variant="body2" color="text.secondary">
        Execution history and activity log.
      </Typography>
      <DataTable
        columns={[
          { key: 'symbol', label: 'Symbol' },
          { key: 'side', label: 'Side' },
          { key: 'quantity', label: 'Qty', align: 'right' },
          { key: 'price', label: 'Price', align: 'right' },
          { key: 'status', label: 'Status' },
        ]}
        rows={blotterRows}
        getRowId={(row) => row.id}
      />
    </Stack>
  );
}
