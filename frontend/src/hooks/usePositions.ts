import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HttpError, apiClient } from '@/data/http-client';

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
  const data = await apiClient.get<PositionsResponse>('/api/v1/broker/positions');

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

      try {
        return await apiClient.post<{ success: boolean; order_id: number; message: string }>(
          `/api/v1/broker/positions/${encodeURIComponent(request.symbol)}/close`,
          payload
        );
      } catch (error) {
        throw new Error(getErrorMessage(error, 'Position close failed'));
      }
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
      try {
        return await apiClient.post<{
          success: boolean;
          order_ids: number[];
          cancelled_order_ids?: number[];
          message: string;
        }>(`/api/v1/broker/positions/${encodeURIComponent(request.symbol)}/protect`, {
          quantity: request.quantity,
          stop_loss: request.stopLoss,
          take_profit: request.takeProfit,
          replace_existing: request.replaceExisting ?? true,
        });
      } catch (error) {
        throw new Error(getErrorMessage(error, 'Position protection failed'));
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['positions'] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
  });
}
