import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Dashboard } from '../lib/types';

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<Dashboard>('/dashboard'),
    refetchInterval: 15_000,
  });
}
