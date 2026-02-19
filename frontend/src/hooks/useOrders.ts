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

async function fetchOrders(): Promise<Order[]> {
  const response = await fetch(buildUrl('JAX_API', '/api/v1/trades'));
  if (!response.ok) {
    throw new Error('Orders service unavailable');
  }
  return response.json();
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
