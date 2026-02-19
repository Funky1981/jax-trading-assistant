import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { signalsService } from '@/data/signals-service';
import type { RecommendationListResponse, SignalListResponse } from '@/data/types';

interface UseSignalsParams {
  status?: string;
  symbol?: string;
  strategy?: string;
  limit?: number;
  offset?: number;
}

export function useSignals(params: UseSignalsParams = {}) {
  return useQuery({
    queryKey: ['signals', params],
    queryFn: () => signalsService.list(params),
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
  });
}

export function useRecommendations(limit = 50, offset = 0) {
  return useQuery({
    queryKey: ['recommendations', limit, offset],
    queryFn: () => signalsService.listRecommendations({ limit, offset }),
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
  });
}

export function useApproveSignal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ signalId, approvedBy, notes }: { signalId: string; approvedBy: string; notes?: string }) =>
      signalsService.approve(signalId, approvedBy, notes),
    onMutate: async ({ signalId }) => {
      await queryClient.cancelQueries({ queryKey: ['signals'] });
      await queryClient.cancelQueries({ queryKey: ['recommendations'] });

      const previousSignals = queryClient.getQueriesData<SignalListResponse>({ queryKey: ['signals'] });
      const previousRecs = queryClient.getQueriesData<RecommendationListResponse>({ queryKey: ['recommendations'] });

      previousSignals.forEach(([key, data]) => {
        if (!data) return;
        const filtered = data.signals.filter((sig) => sig.id !== signalId);
        queryClient.setQueryData<SignalListResponse>(key, {
          ...data,
          signals: filtered,
          total: Math.max(0, data.total - (data.signals.length - filtered.length)),
        });
      });

      previousRecs.forEach(([key, data]) => {
        if (!data) return;
        const filtered = data.recommendations.filter((rec) => rec.signal?.id !== signalId);
        queryClient.setQueryData<RecommendationListResponse>(key, {
          ...data,
          recommendations: filtered,
          total: Math.max(0, data.total - (data.recommendations.length - filtered.length)),
        });
      });

      return { previousSignals, previousRecs };
    },
    onError: (_err, _vars, context) => {
      context?.previousSignals?.forEach(([key, data]) => {
        queryClient.setQueryData(key, data);
      });
      context?.previousRecs?.forEach(([key, data]) => {
        queryClient.setQueryData(key, data);
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signals'] });
      queryClient.invalidateQueries({ queryKey: ['recommendations'] });
    },
  });
}

export function useRejectSignal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ signalId, approvedBy, reason }: { signalId: string; approvedBy: string; reason?: string }) =>
      signalsService.reject(signalId, approvedBy, reason),
    onMutate: async ({ signalId }) => {
      await queryClient.cancelQueries({ queryKey: ['signals'] });
      await queryClient.cancelQueries({ queryKey: ['recommendations'] });

      const previousSignals = queryClient.getQueriesData<SignalListResponse>({ queryKey: ['signals'] });
      const previousRecs = queryClient.getQueriesData<RecommendationListResponse>({ queryKey: ['recommendations'] });

      previousSignals.forEach(([key, data]) => {
        if (!data) return;
        const filtered = data.signals.filter((sig) => sig.id !== signalId);
        queryClient.setQueryData<SignalListResponse>(key, {
          ...data,
          signals: filtered,
          total: Math.max(0, data.total - (data.signals.length - filtered.length)),
        });
      });

      previousRecs.forEach(([key, data]) => {
        if (!data) return;
        const filtered = data.recommendations.filter((rec) => rec.signal?.id !== signalId);
        queryClient.setQueryData<RecommendationListResponse>(key, {
          ...data,
          recommendations: filtered,
          total: Math.max(0, data.total - (data.recommendations.length - filtered.length)),
        });
      });

      return { previousSignals, previousRecs };
    },
    onError: (_err, _vars, context) => {
      context?.previousSignals?.forEach(([key, data]) => {
        queryClient.setQueryData(key, data);
      });
      context?.previousRecs?.forEach(([key, data]) => {
        queryClient.setQueryData(key, data);
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signals'] });
      queryClient.invalidateQueries({ queryKey: ['recommendations'] });
    },
  });
}

export function useAnalyzeSignal() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ signalId, context }: { signalId: string; context?: string }) =>
      signalsService.analyze(signalId, context),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['signals'] });
      queryClient.invalidateQueries({ queryKey: ['recommendations'] });
    },
  });
}
