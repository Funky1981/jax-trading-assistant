import { useQuery } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

export interface IBAccount {
  account_id: string;
  net_liquidation: number;
  total_cash: number;
  buying_power: number;
  equity_with_loan: number;
  currency: string;
}

async function fetchIBAccount(): Promise<IBAccount> {
  const response = await fetch(buildUrl('IB_BRIDGE', '/account'));
  if (!response.ok) {
    throw new Error('IB Bridge unavailable');
  }
  return response.json();
}

export function useIBAccount() {
  return useQuery({
    queryKey: ['ib-account'],
    queryFn: fetchIBAccount,
    refetchInterval: 5000, // Update every 5 seconds
    retry: 2,
  });
}
