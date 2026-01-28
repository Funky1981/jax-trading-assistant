import { Stack, Typography } from '@mui/material';
import { DataTable } from '../components';
import { useDomain } from '../domain/store';
import { selectOrders } from '../domain/selectors';
import { formatPrice } from '../domain/market';

export function BlotterPage() {
  const { state } = useDomain();
  const orders = selectOrders(state).sort((a, b) => b.createdAt - a.createdAt);

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
          {
            key: 'price',
            label: 'Price',
            align: 'right',
            render: (row) => formatPrice(row.price),
          },
          { key: 'status', label: 'Status' },
        ]}
        rows={orders}
        getRowId={(row) => row.id}
      />
    </Stack>
  );
}
