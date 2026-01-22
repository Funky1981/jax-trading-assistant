import type { Side } from './types';

export type OrderStatus = 'open' | 'filled' | 'cancelled' | 'rejected';
export type AlertSeverity = 'info' | 'warning' | 'critical';

export interface Instrument {
  symbol: string;
  name: string;
  tickSize: number;
  lotSize: number;
}

export interface Order {
  id: string;
  symbol: string;
  side: Side;
  quantity: number;
  price: number;
  status: OrderStatus;
  createdAt: number;
}

export interface Position {
  symbol: string;
  quantity: number;
  avgPrice: number;
  marketPrice: number;
}

export interface RiskLimits {
  maxPositionValue: number;
  maxDailyLoss: number;
}

export interface Alert {
  id: string;
  message: string;
  severity: AlertSeverity;
  createdAt: number;
}
