import { DataTable } from '../components';
import { useDomain } from '../domain/store';
import { selectOrders } from '../domain/selectors';
import { formatPrice } from '../domain/market';
import { HelpHint } from '@/components/ui/help-hint';

export function BlotterPage() {
  const { state } = useDomain();
  const orders = selectOrders(state).sort((a, b) => b.createdAt - a.createdAt);

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-3xl font-semibold">
        Blotter
        <HelpHint text="Execution history and order activity log." />
      </h1>
      <p className="text-sm text-muted-foreground">
        Review recent orders and their status.
      </p>
      <div className="w-full overflow-x-auto [&_table]:min-w-[520px]">
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
      </div>
    </div>
  );
}
