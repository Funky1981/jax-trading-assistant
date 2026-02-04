import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

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

// Mock orders data
const mockOrders: Order[] = [
  {
    id: 'ord-001',
    symbol: 'AAPL',
    side: 'buy',
    type: 'limit',
    quantity: 50,
    price: 184.50,
    status: 'filled',
    filledQuantity: 50,
    avgFillPrice: 184.48,
    createdAt: Date.now() - 3600000,
    updatedAt: Date.now() - 3500000,
  },
  {
    id: 'ord-002',
    symbol: 'MSFT',
    side: 'sell',
    type: 'market',
    quantity: 25,
    status: 'filled',
    filledQuantity: 25,
    avgFillPrice: 412.75,
    createdAt: Date.now() - 7200000,
    updatedAt: Date.now() - 7100000,
  },
  {
    id: 'ord-003',
    symbol: 'NVDA',
    side: 'buy',
    type: 'limit',
    quantity: 10,
    price: 700.00,
    status: 'pending',
    filledQuantity: 0,
    createdAt: Date.now() - 1800000,
    updatedAt: Date.now() - 1800000,
  },
  {
    id: 'ord-004',
    symbol: 'TSLA',
    side: 'sell',
    type: 'stop_limit',
    quantity: 20,
    price: 225.00,
    stopPrice: 228.00,
    status: 'pending',
    filledQuantity: 0,
    createdAt: Date.now() - 900000,
    updatedAt: Date.now() - 900000,
  },
  {
    id: 'ord-005',
    symbol: 'AMD',
    side: 'buy',
    type: 'market',
    quantity: 100,
    status: 'filled',
    filledQuantity: 100,
    avgFillPrice: 155.32,
    createdAt: Date.now() - 14400000,
    updatedAt: Date.now() - 14300000,
  },
  {
    id: 'ord-006',
    symbol: 'GOOGL',
    side: 'buy',
    type: 'limit',
    quantity: 30,
    price: 135.00,
    status: 'cancelled',
    filledQuantity: 0,
    createdAt: Date.now() - 28800000,
    updatedAt: Date.now() - 25200000,
  },
];

async function fetchOrders(): Promise<Order[]> {
  try {
    const response = await fetch('http://localhost:8080/api/orders');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  return mockOrders;
}

export function useOrders() {
  return useQuery({
    queryKey: ['orders'],
    queryFn: fetchOrders,
    refetchInterval: 5000,
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
