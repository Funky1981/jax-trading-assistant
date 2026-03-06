import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { TradeBlotterPanel } from './TradeBlotterPanel';

const cancelMutate = vi.fn();

vi.mock('@/hooks/useOrders', () => ({
  useOrdersSummary: () => ({
    data: {
      total: 2,
      pending: 1,
      filled: 1,
      cancelled: 0,
      lastFill: {
        id: 'strategy-1',
        symbol: 'MSFT',
        side: 'buy',
        type: 'market',
        quantity: 5,
        status: 'filled',
        filledQuantity: 5,
        createdAt: Date.parse('2026-03-06T14:00:00Z'),
        updatedAt: Date.parse('2026-03-06T14:01:00Z'),
        source: 'strategy',
        canCancel: false,
      },
    },
    orders: [
      {
        id: 'broker-101',
        brokerOrderId: 101,
        symbol: 'AAPL',
        side: 'buy',
        type: 'limit',
        quantity: 10,
        price: 200,
        status: 'pending',
        filledQuantity: 0,
        createdAt: Date.parse('2026-03-06T14:00:00Z'),
        updatedAt: Date.parse('2026-03-06T14:01:00Z'),
        source: 'broker',
        canCancel: true,
        workflow: 'entry',
      },
      {
        id: 'strategy-1',
        symbol: 'MSFT',
        side: 'buy',
        type: 'market',
        quantity: 5,
        status: 'filled',
        filledQuantity: 5,
        createdAt: Date.parse('2026-03-06T14:00:00Z'),
        updatedAt: Date.parse('2026-03-06T14:01:00Z'),
        source: 'strategy',
        canCancel: false,
        workflow: 'strategy',
      },
    ],
    isLoading: false,
  }),
  useCancelOrder: () => ({
    mutate: cancelMutate,
    isPending: false,
    error: null,
  }),
}));

describe('TradeBlotterPanel', () => {
  it('shows cancel only for broker-managed working orders', () => {
    render(<TradeBlotterPanel isOpen onToggle={() => undefined} />);

    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    expect(screen.getAllByText('Read only')).toHaveLength(1);
    expect(screen.getByText('Broker')).toBeInTheDocument();
    expect(screen.getAllByText('Strategy').length).toBeGreaterThan(0);
  });
});
