import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/data/http-client';

export interface IBAccount {
  account_id: string;
  net_liquidation: number;
  total_cash: number;
  buying_power: number;
  equity_with_loan: number;
  currency: string;
}

async function fetchIBAccount(): Promise<IBAccount> {
  return apiClient.get<IBAccount>('/api/v1/broker/account');
}

export function useIBAccount() {
  return useQuery({
    queryKey: ['ib-account'],
    queryFn: fetchIBAccount,
    refetchInterval: 5000, // Update every 5 seconds
    retry: 2,
  });
}
