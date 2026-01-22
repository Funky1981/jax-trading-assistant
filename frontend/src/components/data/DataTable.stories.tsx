import type { Meta, StoryObj } from '@storybook/react';
import { DataTable } from './DataTable';

interface Row {
  symbol: string;
  quantity: number;
  price: number;
}

const meta: Meta<typeof DataTable<Row>> = {
  title: 'Data/DataTable',
  component: DataTable as unknown as typeof DataTable<Row>,
  args: {
    columns: [
      { key: 'symbol', label: 'Symbol' },
      { key: 'quantity', label: 'Qty', align: 'right' },
      { key: 'price', label: 'Price', align: 'right' },
    ],
    rows: [
      { symbol: 'AAPL', quantity: 120, price: 249.42 },
      { symbol: 'MSFT', quantity: 80, price: 413.1 },
    ],
    getRowId: (row: Row) => row.symbol,
  },
};

export default meta;

type Story = StoryObj<typeof DataTable<Row>>;

export const Default: Story = {};
