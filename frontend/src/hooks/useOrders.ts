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
    case 'pending': return 'pending';
    case 'partfilled':
    case 'partial': return 'partial';
    case 'cancelled': return 'cancelled';
    default: return 'rejected';
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
    price: raw.price ?? raw.entry,
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
      console.log('Creating order:', order);
      
      // Simulate order creation
      const newOrder: Order = {
        id: `ord-${Date.now()}`,
        ...order,
        status: order.type === 'market' ? 'filled' : 'pending',
        filledQuantity: order.type === 'market' ? order.quantity : 0,
        avgFillPrice: order.type === 'market' ? order.price || 100 : undefined,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      };
      
      return newOrder;
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
    mutationFn: async (orderId: string) => {
      console.log('Cancelling order:', orderId);
      return { success: true, orderId };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}
