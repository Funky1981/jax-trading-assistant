import type { DomainEvent } from './events';
import type { MarketTick } from '../data/types';
import type { Alert, Order, Position, RiskLimits } from './models';

export interface DomainState {
  orders: Record<string, Order>;
  positions: Record<string, Position>;
  riskLimits: RiskLimits;
  alerts: Alert[];
  ticks: Record<string, MarketTick>;
}

export const defaultRiskLimits: RiskLimits = {
  maxPositionValue: 5_000_000,
  maxDailyLoss: 100_000,
};

export const defaultState: DomainState = {
  orders: {},
  positions: {},
  riskLimits: defaultRiskLimits,
  alerts: [],
  ticks: {},
};

export function reduceDomainState(state: DomainState, event: DomainEvent): DomainState {
  switch (event.type) {
    case 'OrderPlaced': {
      return {
        ...state,
        orders: {
          ...state.orders,
          [event.order.id]: event.order,
        },
      };
    }
    case 'OrderUpdated': {
      const existing = state.orders[event.orderId];
      if (!existing) return state;
      return {
        ...state,
        orders: {
          ...state.orders,
          [event.orderId]: {
            ...existing,
            status: event.status,
          },
        },
      };
    }
    case 'PriceUpdated': {
      const existing = state.positions[event.symbol];
      const previousTick = state.ticks[event.symbol];
      const nextTick: MarketTick = {
        symbol: event.symbol,
        price: event.price,
        changePct:
          event.changePct ??
          (previousTick ? ((event.price - previousTick.price) / previousTick.price) * 100 : 0),
        timestamp: event.timestamp ?? Date.now(),
      };
      if (!existing) {
        return {
          ...state,
          ticks: {
            ...state.ticks,
            [event.symbol]: nextTick,
          },
        };
      }
      return {
        ...state,
        positions: {
          ...state.positions,
          [event.symbol]: {
            ...existing,
            marketPrice: event.price,
          },
        },
        ticks: {
          ...state.ticks,
          [event.symbol]: nextTick,
        },
      };
    }
    case 'PositionUpdated': {
      return {
        ...state,
        positions: {
          ...state.positions,
          [event.position.symbol]: event.position,
        },
      };
    }
    case 'RiskLimitsUpdated': {
      return {
        ...state,
        riskLimits: event.limits,
      };
    }
    case 'AlertRaised': {
      const nextAlerts = [event.alert, ...state.alerts].slice(0, 100);
      return {
        ...state,
        alerts: nextAlerts,
      };
    }
    default:
      return state;
  }
}
