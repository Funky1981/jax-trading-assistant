import type { Alert, Order, OrderStatus, Position, RiskLimits } from './models';

export type DomainEvent =
  | { type: 'OrderPlaced'; order: Order }
  | { type: 'OrderUpdated'; orderId: string; status: OrderStatus }
  | {
      type: 'PriceUpdated';
      symbol: string;
      price: number;
      changePct?: number;
      timestamp?: number;
    }
  | { type: 'PositionUpdated'; position: Position }
  | { type: 'RiskLimitsUpdated'; limits: RiskLimits }
  | { type: 'AlertRaised'; alert: Alert };
