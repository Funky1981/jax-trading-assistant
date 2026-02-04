import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export interface Position {
  id: string;
  symbol: string;
  quantity: number;
  avgPrice: number;
  marketPrice: number;
  pnl: number;
  pnlPercent: number;
  marketValue: number;
  costBasis: number;
}

// Mock positions data
const mockPositions: Position[] = [
  {
    id: '1',
    symbol: 'AAPL',
    quantity: 100,
    avgPrice: 178.50,
    marketPrice: 185.92,
    pnl: 742.00,
    pnlPercent: 4.16,
    marketValue: 18592.00,
    costBasis: 17850.00,
  },
  {
    id: '2',
    symbol: 'MSFT',
    quantity: 50,
    avgPrice: 378.25,
    marketPrice: 412.65,
    pnl: 1720.00,
    pnlPercent: 9.09,
    marketValue: 20632.50,
    costBasis: 18912.50,
  },
  {
    id: '3',
    symbol: 'GOOGL',
    quantity: 25,
    avgPrice: 142.80,
    marketPrice: 138.45,
    pnl: -108.75,
    pnlPercent: -3.05,
    marketValue: 3461.25,
    costBasis: 3570.00,
  },
  {
    id: '4',
    symbol: 'NVDA',
    quantity: 30,
    avgPrice: 485.20,
    marketPrice: 721.33,
    pnl: 7083.90,
    pnlPercent: 48.68,
    marketValue: 21639.90,
    costBasis: 14556.00,
  },
  {
    id: '5',
    symbol: 'TSLA',
    quantity: 40,
    avgPrice: 248.90,
    marketPrice: 231.45,
    pnl: -698.00,
    pnlPercent: -7.01,
    marketValue: 9258.00,
    costBasis: 9956.00,
  },
];

async function fetchPositions(): Promise<Position[]> {
  try {
    const response = await fetch('http://localhost:8080/api/positions');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  // Return mock data with slight price variations
  return mockPositions.map((pos) => {
    const priceChange = pos.marketPrice * (Math.random() * 0.02 - 0.01);
    const newMarketPrice = pos.marketPrice + priceChange;
    const newPnl = (newMarketPrice - pos.avgPrice) * pos.quantity;
    const newPnlPercent = ((newMarketPrice - pos.avgPrice) / pos.avgPrice) * 100;
    
    return {
      ...pos,
      marketPrice: Number(newMarketPrice.toFixed(2)),
      pnl: Number(newPnl.toFixed(2)),
      pnlPercent: Number(newPnlPercent.toFixed(2)),
      marketValue: Number((newMarketPrice * pos.quantity).toFixed(2)),
    };
  });
}

export function usePositions() {
  return useQuery({
    queryKey: ['positions'],
    queryFn: fetchPositions,
    refetchInterval: 5000, // Refresh every 5 seconds
  });
}

export function usePositionsSummary() {
  const { data: positions, ...rest } = usePositions();
  
  const summary = positions
    ? {
        totalValue: positions.reduce((sum, p) => sum + p.marketValue, 0),
        totalPnl: positions.reduce((sum, p) => sum + p.pnl, 0),
        totalPnlPercent:
          positions.reduce((sum, p) => sum + p.pnl, 0) /
          positions.reduce((sum, p) => sum + p.costBasis, 0) *
          100,
        positionCount: positions.length,
      }
    : null;
  
  return { ...rest, data: summary, positions };
}

export function useClosePosition() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (positionId: string) => {
      // In real app, call API
      console.log('Closing position:', positionId);
      return { success: true };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['positions'] });
    },
  });
}
