import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { OrderTicketPanel } from './OrderTicketPanel';

const mutate = vi.fn();

vi.mock('@/hooks/useOrders', () => ({
  useCreateOrder: () => ({
    mutate,
    isPending: false,
    error: null,
    data: null,
  }),
}));

vi.mock('@/hooks/useMarketDataStatus', () => ({
  useMarketDataStatus: () => ({
    data: {
      marketDataMode: 'delayed',
      paperTrading: true,
    },
    isError: false,
  }),
}));

vi.mock('@/hooks/useTradingPilotStatus', () => ({
  useTradingPilotStatus: () => ({
    data: {
      readOnly: false,
      reasons: ['Quotes on this screen are non-authoritative during the pilot; confirm in IB/TWS.'],
    },
  }),
}));

describe('OrderTicketPanel', () => {
  it('submits a bracket order when stop loss protection is provided', async () => {
    const user = userEvent.setup();

    render(<OrderTicketPanel isOpen onToggle={() => undefined} />);

    await user.type(screen.getByLabelText('Symbol'), 'AAPL');
    await user.type(screen.getByLabelText('Quantity'), '10');
    await user.type(screen.getByLabelText('Stop Loss'), '195');

    expect(screen.getByRole('button', { name: 'Submit BUY Bracket' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Submit BUY Bracket' }));
    await user.click(screen.getByRole('checkbox'));
    await user.click(screen.getByRole('button', { name: 'Submit Broker Order' }));

    expect(mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        symbol: 'AAPL',
        side: 'buy',
        type: 'market',
        quantity: 10,
        stopLossPrice: 195,
      }),
      expect.any(Object)
    );
  });
});
