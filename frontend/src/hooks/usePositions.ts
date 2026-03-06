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

export interface ClosePositionRequest {
  symbol: string;
  quantity: number;
  orderType: 'MKT' | 'LMT';
  limitPrice?: number;
}

export interface ProtectPositionRequest {
  symbol: string;
  quantity: number;
  stopLoss: number;
  takeProfit?: number;
  replaceExisting?: boolean;
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
    ? (() => {
        const totalValue = positions.reduce((sum, p) => sum + p.marketValue, 0);
        const totalPnl = positions.reduce((sum, p) => sum + p.pnl, 0);
        const totalCostBasis = positions.reduce((sum, p) => sum + p.costBasis, 0);
        const totalPnlPercent = totalCostBasis !== 0 ? (totalPnl / totalCostBasis) * 100 : 0;

        return {
          totalValue,
          totalPnl,
          totalPnlPercent,
          positionCount: positions.length,
        };
      })()
    : null;

  return { ...rest, data: summary, positions };
}

export function useClosePosition() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: ClosePositionRequest) => {
      const payload = {
        quantity: request.quantity,
        order_type: request.orderType,
        limit_price: request.orderType === 'LMT' ? request.limitPrice : undefined,
      };

      const response = await fetch(
        buildUrl('IB_BRIDGE', `/positions/${encodeURIComponent(request.symbol)}/close`),
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        }
      );

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Position close failed (HTTP ${response.status})`);
      }

      return response.json() as Promise<{ success: boolean; order_id: number; message: string }>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['positions'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}

export function useProtectPosition() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: ProtectPositionRequest) => {
      const response = await fetch(
        buildUrl('IB_BRIDGE', `/positions/${encodeURIComponent(request.symbol)}/protect`),
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            quantity: request.quantity,
            stop_loss: request.stopLoss,
            take_profit: request.takeProfit,
            replace_existing: request.replaceExisting ?? true,
          }),
        }
      );

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || `Position protection failed (HTTP ${response.status})`);
      }

      return response.json() as Promise<{
        success: boolean;
        order_ids: number[];
        cancelled_order_ids?: number[];
        message: string;
      }>;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['positions'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}
