import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

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

interface RawPosition {
  contract_id?: string | number;
  symbol?: string;
  quantity?: number;
  avg_cost?: number;
  market_price?: number;
  unrealized_pnl?: number;
  market_value?: number;
}

interface PositionsResponse {
  positions: RawPosition[];
}

async function fetchPositions(): Promise<Position[]> {
  const response = await fetch(buildUrl('IB_BRIDGE', '/positions'));
  if (!response.ok) {
    throw new Error(`IB Bridge positions unavailable (HTTP ${response.status})`);
  }

  const data = (await response.json()) as PositionsResponse;

  return data.positions.map((pos) => ({
    id: `${pos.contract_id || pos.symbol}`,
    symbol: pos.symbol ?? '',
    quantity: pos.quantity ?? 0,
    avgPrice: pos.avg_cost ?? 0,
    marketPrice: pos.market_price ?? 0,
    pnl: pos.unrealized_pnl ?? 0,
    pnlPercent:
      pos.avg_cost && pos.quantity
        ? ((pos.unrealized_pnl ?? 0) / (pos.avg_cost * pos.quantity)) * 100
        : 0,
    marketValue: pos.market_value ?? 0,
    costBasis: (pos.avg_cost ?? 0) * (pos.quantity ?? 0),
  }));
}

export function usePositions() {
  return useQuery({
    queryKey: ['positions'],
    queryFn: fetchPositions,
    refetchInterval: 5000,
  });
}

export function usePositionsSummary() {
  const { data: positions, ...rest } = usePositions();

  const summary = positions
    ? {
        totalValue: positions.reduce((sum, p) => sum + p.marketValue, 0),
        totalPnl: positions.reduce((sum, p) => sum + p.pnl, 0),
        totalPnlPercent:
          (positions.reduce((sum, p) => sum + p.pnl, 0) /
            positions.reduce((sum, p) => sum + p.costBasis, 0)) *
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
      console.log('Closing position:', positionId);
      return { success: true };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['positions'] });
    },
  });
}
