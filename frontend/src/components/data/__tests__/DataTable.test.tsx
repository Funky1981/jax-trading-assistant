import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DataTable } from '../DataTable';

describe('DataTable', () => {
  it('renders headers and rows', () => {
    render(
      <DataTable
        columns={[
          { key: 'symbol', label: 'Symbol' },
          { key: 'price', label: 'Price', align: 'right' },
        ]}
        rows={[{ symbol: 'AAPL', price: 100 }]}
        getRowId={(row) => row.symbol}
      />
    );

    expect(screen.getByText('Symbol')).toBeInTheDocument();
    expect(screen.getByText('AAPL')).toBeInTheDocument();
  });
});
