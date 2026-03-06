import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HttpError, apiClient } from '@/data/http-client';

export type OrderSide = 'buy' | 'sell';
export type OrderType = 'market' | 'limit' | 'stop' | 'stop_limit';
export type OrderStatus = 'pending' | 'filled' | 'partial' | 'cancelled' | 'rejected';
export type OrderSource = 'broker' | 'strategy';

export interface Order {
  id: string;
  brokerOrderId?: number;
  symbol: string;
  side: OrderSide;
  type: OrderType;
  quantity: number;
  price?: number;
  stopPrice?: number;
  status: OrderStatus;
  filledQuantity: number;
  avgFillPrice?: number;
  createdAt: number;
  updatedAt: number;
  source: OrderSource;
  canCancel: boolean;
  parentId?: number;
  workflow?: 'entry' | 'close' | 'protect' | 'strategy';
}

export interface CreateOrderRequest {
  symbol: string;
  side: OrderSide;
  type: OrderType;
  quantity: number;
  price?: number;
  stopPrice?: number;
  stopLossPrice?: number;
  takeProfitPrice?: number;
}

interface RawBrokerOrder {
  order_id?: number;
  symbol?: string;
  action?: string;
  order_type?: string;
  quantity?: number;
  limit_price?: number;
  stop_price?: number;
  status?: string;
  filled_qty?: number;
  avg_fill_price?: number;
  created_at?: string;
  updated_at?: string;
  can_cancel?: boolean;
  parent_id?: number;
  order_ref?: string;
}

function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof HttpError) {
    const body = error.body;
    if (typeof body === 'string' && body.trim()) {
      return body;
    }
    if (body && typeof body === 'object' && 'error' in body && typeof body.error === 'string') {
      return body.error;
    }
    if (body && typeof body === 'object' && 'detail' in body && typeof body.detail === 'string') {
      return body.detail;
    }
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}

function normalizeStatus(raw?: string): OrderStatus {
  switch (raw?.toLowerCase()) {
    case 'filled':
      return 'filled';
    case 'pending':
    case 'submitted':
    case 'presubmitted':
    case 'new':
    case 'open':
    case 'pendingsubmit':
    case 'pendingcancel':
      return 'pending';
    case 'partfilled':
    case 'partial':
      return 'partial';
    case 'cancelled':
    case 'canceled':
    case 'apicancelled':
      return 'cancelled';
    case 'rejected':
    case 'inactive':
    case 'error':
      return 'rejected';
    default:
      return 'pending';
  }
}

function normalizeOrderType(raw?: string): OrderType {
  switch ((raw ?? '').toUpperCase()) {
    case 'LMT':
    case 'LIMIT':
      return 'limit';
    case 'STP':
    case 'STOP':
      return 'stop';
    case 'STP LMT':
    case 'STOP_LIMIT':
      return 'stop_limit';
    default:
      return 'market';
  }
}

function normalizeWorkflow(orderRef?: string, parentId?: number): Order['workflow'] {
  switch (orderRef) {
    case 'manual-close':
      return 'close';
    case 'manual-protect':
    case 'manual-bracket-child':
      return 'protect';
    case 'manual-entry':
      return parentId ? 'protect' : 'entry';
    default:
      return 'strategy';
  }
}

function fallbackId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 100000)}`;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapApiTrade(raw: any): Order {
  return {
    id: `strategy-${raw.id ?? fallbackId('strategy')}`,
    symbol: raw.symbol ?? '',
    side: (raw.direction ?? raw.side ?? 'buy').toLowerCase() as OrderSide,
    type: (raw.type ?? 'market') as OrderType,
    quantity: raw.quantity ?? raw.filled_qty ?? 0,
    price: raw.limit_price ?? raw.price,
    stopPrice: raw.stop_price ?? raw.stop,
    status: normalizeStatus(raw.order_status ?? raw.status),
    filledQuantity: raw.filled_qty ?? raw.filledQuantity ?? 0,
    avgFillPrice: raw.avg_fill_price ?? raw.avgFillPrice,
    createdAt: raw.created_at
      ? new Date(raw.created_at).getTime()
      : raw.createdAt ?? Date.now(),
    updatedAt: raw.updated_at
      ? new Date(raw.updated_at).getTime()
      : raw.created_at
        ? new Date(raw.created_at).getTime()
        : raw.updatedAt ?? Date.now(),
    source: 'strategy',
    canCancel: false,
    workflow: 'strategy',
  };
}

function mapBrokerOrder(raw: RawBrokerOrder): Order {
  return {
    id: `broker-${raw.order_id ?? fallbackId('broker')}`,
    brokerOrderId: raw.order_id,
    symbol: raw.symbol ?? '',
    side: ((raw.action ?? 'BUY').toLowerCase() === 'sell' ? 'sell' : 'buy') as OrderSide,
    type: normalizeOrderType(raw.order_type),
    quantity: raw.quantity ?? 0,
    price: raw.limit_price,
    stopPrice: raw.stop_price,
    status: normalizeStatus(raw.status),
    filledQuantity: raw.filled_qty ?? 0,
    avgFillPrice: raw.avg_fill_price,
    createdAt: raw.created_at ? new Date(raw.created_at).getTime() : Date.now(),
    updatedAt: raw.updated_at ? new Date(raw.updated_at).getTime() : Date.now(),
    source: 'broker',
    canCancel: Boolean(raw.can_cancel),
    parentId: raw.parent_id ?? undefined,
    workflow: normalizeWorkflow(raw.order_ref, raw.parent_id ?? undefined),
  };
}

async function fetchStrategyOrders(): Promise<Order[]> {
  const data = await apiClient.get<{ trades?: unknown[] } | unknown[]>('/api/v1/trades');
  const raw = Array.isArray(data) ? data : (data.trades ?? []);
  return raw.map(mapApiTrade);
}

async function fetchBrokerOrders(): Promise<Order[]> {
  const data = await apiClient.get<{ orders?: RawBrokerOrder[] } | RawBrokerOrder[]>('/api/v1/broker/orders');
  const raw = Array.isArray(data) ? data : (data.orders ?? []);
  return raw.map(mapBrokerOrder);
}

async function fetchOrders(): Promise<Order[]> {
  const [strategyResult, brokerResult] = await Promise.allSettled([
    fetchStrategyOrders(),
    fetchBrokerOrders(),
  ]);

  if (strategyResult.status === 'rejected' && brokerResult.status === 'rejected') {
    throw new Error('Orders service unavailable');
  }

  const merged = [
    ...(strategyResult.status === 'fulfilled' ? strategyResult.value : []),
    ...(brokerResult.status === 'fulfilled' ? brokerResult.value : []),
  ];

  return merged.sort((a, b) => b.updatedAt - a.updatedAt);
}

export function useOrders() {
  return useQuery({
    queryKey: ['orders'],
    queryFn: fetchOrders,
    refetchInterval: (query) => (query.state.error ? false : 5_000),
    retry: false,
  });
}

export function useOrdersSummary() {
  const { data: orders, ...rest } = useOrders();

  const summary = orders
    ? {
        total: orders.length,
        pending: orders.filter((o) => o.status === 'pending').length,
        filled: orders.filter((o) => o.status === 'filled').length,
        cancelled: orders.filter((o) => o.status === 'cancelled').length,
        lastFill: orders
          .filter((o) => o.status === 'filled')
          .sort((a, b) => b.updatedAt - a.updatedAt)[0],
      }
    : null;

  return { ...rest, data: summary, orders };
}

export function useCreateOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (order: CreateOrderRequest) => {
      const orderTypeMap: Record<OrderType, string> = {
        market: 'MKT',
        limit: 'LMT',
        stop: 'STP',
        stop_limit: 'STP LMT',
      };

      if (order.type === 'limit' && !order.price) {
        throw new Error('Limit orders require a price');
      }

      if (order.type === 'stop' && !order.stopPrice && !order.price) {
        throw new Error('Stop orders require an entry stop price');
      }

      if (order.type === 'stop_limit') {
        throw new Error('Stop-limit is not supported by the current broker bridge');
      }

      const hasProtection = Boolean(order.stopLossPrice || order.takeProfitPrice);

      if (hasProtection) {
        if (!['market', 'limit'].includes(order.type)) {
          throw new Error('Attached protection is available only for market or limit entries');
        }

        try {
          return await apiClient.post<{
            success: boolean;
            parent_order_id: number;
            child_order_ids: number[];
            message?: string;
          }>('/api/v1/broker/orders/bracket', {
            symbol: order.symbol.toUpperCase(),
            action: order.side.toUpperCase(),
            quantity: order.quantity,
            entry_order_type: orderTypeMap[order.type],
            entry_limit_price: order.type === 'limit' ? order.price : undefined,
            stop_loss: order.stopLossPrice,
            take_profit: order.takeProfitPrice,
          });
        } catch (error) {
          throw new Error(getErrorMessage(error, 'Bracket order failed'));
        }
      }

      try {
        return await apiClient.post<{ success: boolean; order_id: number; message?: string }>('/api/v1/broker/orders', {
          symbol: order.symbol.toUpperCase(),
          action: order.side.toUpperCase(),
          quantity: order.quantity,
          order_type: orderTypeMap[order.type],
          limit_price: order.type === 'limit' ? order.price : undefined,
          stop_price: order.type === 'stop' ? (order.stopPrice ?? order.price) : undefined,
        });
      } catch (error) {
        throw new Error(getErrorMessage(error, 'Order placement failed'));
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['positions'] });
    },
  });
}

export function useCancelOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (order: Order) => {
      if (order.source !== 'broker' || !order.brokerOrderId) {
        throw new Error('Only broker-managed orders can be cancelled from this blotter');
      }

      try {
        return await apiClient.delete<{ success: boolean; order_id: number; status: string; message?: string }>(
          `/api/v1/broker/orders/${order.brokerOrderId}`
        );
      } catch (error) {
        throw new Error(getErrorMessage(error, 'Order cancel failed'));
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['positions'] });
    },
  });
}
