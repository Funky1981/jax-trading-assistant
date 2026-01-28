import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { useState } from 'react';
import { DataTable, PrimaryButton } from '../../components';

const columns = [
  { key: 'symbol', label: 'Symbol' },
  { key: 'side', label: 'Side' },
  { key: 'quantity', label: 'Qty', align: 'right' },
];

const initialRows = [
  { id: '1', symbol: 'AAPL', side: 'Buy', quantity: 150 },
  { id: '2', symbol: 'MSFT', side: 'Sell', quantity: 50 },
];

const refreshedRows = [{ id: '3', symbol: 'TSLA', side: 'Buy', quantity: 25 }];

function BlotterHarness() {
  const [rows, setRows] = useState(initialRows);

  return (
    <div>
      <PrimaryButton onClick={() => setRows(refreshedRows)}>Refresh</PrimaryButton>
      <DataTable columns={columns} rows={rows} getRowId={(row) => row.id} />
    </div>
  );
}

describe('blotter updates', () => {
  it('renders refreshed rows after update', async () => {
    const user = userEvent.setup();
    render(<BlotterHarness />);

    expect(screen.getByText('AAPL')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(screen.getByText('TSLA')).toBeInTheDocument();
  });
});
