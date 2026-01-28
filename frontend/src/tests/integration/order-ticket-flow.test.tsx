import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { OrderTicket } from '../../components';

describe('order ticket flow', () => {
  it('submits the order payload', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();

    render(<OrderTicket symbol="AAPL" onSubmit={onSubmit} />);

    const quantityInput = screen.getByLabelText('Quantity');
    const priceInput = screen.getByLabelText('Limit Price');

    await user.clear(quantityInput);
    await user.type(quantityInput, '250');
    await user.clear(priceInput);
    await user.type(priceInput, '123.45');
    await user.click(screen.getByRole('button', { name: 'Place Order' }));

    expect(onSubmit).toHaveBeenCalledWith({
      symbol: 'AAPL',
      side: 'buy',
      quantity: 250,
      price: 123.45,
    });
  });
});
