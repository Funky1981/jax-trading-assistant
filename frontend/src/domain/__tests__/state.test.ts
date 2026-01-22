import { describe, expect, it } from 'vitest';
import { reduceDomainState, defaultState } from '../state';
import { selectOpenOrders, selectRiskBreach } from '../selectors';
import type { DomainEvent } from '../events';

const order = {
  id: 'ord-1',
  symbol: 'AAPL',
  side: 'buy' as const,
  quantity: 10,
  price: 100,
  status: 'open' as const,
  createdAt: 1,
};

const position = {
  symbol: 'AAPL',
  quantity: 10,
  avgPrice: 100,
  marketPrice: 100,
};

describe('domain reducer', () => {
  it('handles order placement and updates', () => {
    const placed: DomainEvent = { type: 'OrderPlaced', order };
    const updated: DomainEvent = { type: 'OrderUpdated', orderId: 'ord-1', status: 'filled' };

    const state1 = reduceDomainState(defaultState, placed);
    const state2 = reduceDomainState(state1, updated);

    expect(selectOpenOrders(state1)).toHaveLength(1);
    expect(selectOpenOrders(state2)).toHaveLength(0);
  });

  it('handles position + price updates and risk checks', () => {
    const withPosition = reduceDomainState(defaultState, {
      type: 'PositionUpdated',
      position,
    });

    const withPrice = reduceDomainState(withPosition, {
      type: 'PriceUpdated',
      symbol: 'AAPL',
      price: 150,
    });

    const breached = selectRiskBreach({
      ...withPrice,
      riskLimits: { maxPositionValue: 100, maxDailyLoss: 10 },
    });

    expect(withPrice.positions.AAPL.marketPrice).toBe(150);
    expect(breached).toBe(true);
  });
});
