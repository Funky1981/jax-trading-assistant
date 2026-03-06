import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

export type OrderSide = 'buy' | 'sell';
export type OrderType = 'market' | 'limit' | 'stop' | 'stop_limit';
export type OrderStatus = 'pending' | 'filled' | 'partial' | 'cancelled' | 'rejected';

export interface Order {
  id: string;
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
}

export interface CreateOrderRequest {
  symbol: string;
  side: OrderSide;
  type: OrderType;
  quantity: number;
  price?: number;
  stopPrice?: number;
}

function normalizeStatus(raw?: string): OrderStatus {
  switch (raw?.toLowerCase()) {
    case 'filled': return 'filled';
    case 'pending':
    case 'submitted':
    case 'presubmitted':
    case 'new':
    case 'open':
      return 'pending';
    case 'partfilled':
    case 'partial': return 'partial';
    case 'cancelled':
    case 'canceled':
    case 'apicancelled':
      return 'cancelled';
    case 'rejected':
    case 'inactive':
    case 'error':
      return 'rejected';
    default:
      // Unknown broker statuses are treated as pending review, not hard reject.
      return 'pending';
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapApiTrade(raw: any): Order {
  return {
    id: raw.id ?? '',
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
      : (raw.createdAt ?? Date.now()),
    updatedAt: raw.updated_at
      ? new Date(raw.updated_at).getTime()
      : raw.created_at
      ? new Date(raw.created_at).getTime()
      : (raw.updatedAt ?? Date.now()),
  };
}

async function fetchOrders(): Promise<Order[]> {
  const response = await fetch(buildUrl('JAX_API', '/api/v1/trades'));
  if (!response.ok) {
    throw new Error('Orders service unavailable');
  }
  const data = await response.json();
  // API returns { trades: [...] } envelope with snake_case fields
  const raw = Array.isArray(data) ? data : (data.trades ?? []);
  return raw.map(mapApiTrade);
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
        throw new Error('Stop orders require stopPrice or price');
      }
      if (order.type === 'stop_limit') {
        throw new Error('Stop-limit is not supported by the current broker bridge');
      }

      const response = await fetch(buildUrl('IB_BRIDGE', '/orders'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          symbol: order.symbol.toUpperCase(),
          action: order.side.toUpperCase(),
          quantity: order.quantity,
          order_type: orderTypeMap[order.type],
          limit_price: order.type === 'limit' ? order.price : undefined,
          stop_price: order.type === 'stop' ? (order.stopPrice ?? order.price) : undefined,
        }),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Order placement failed (HTTP ${response.status})`);
      }

      return response.json() as Promise<{ success: boolean; order_id: number; message?: string }>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      queryClient.invalidateQueries({ queryKey: ['positions'] });
    },
  });
}

export function useCancelOrder() {
  return useMutation({
    mutationFn: async () => {
      throw new Error('Cancel order is not available via current broker bridge API');
    },
  });
}
