import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { useDashboard } from './useDashboard';
import type { Dashboard } from '../lib/types';

export interface Readiness {
  dash?: Dashboard;
  ready?: boolean;
  offline: boolean;
  label: string;
}

// The dashboard alone cannot tell "not ready yet" from "server gone": a failed
// refetch just leaves the last payload in place. A dedicated probe on a short
// cadence answers that, and both bars read the same verdict.
export function useReadiness(): Readiness {
  const dashboard = useDashboard();
  const health = useQuery({
    queryKey: ['health', 'readiness'],
    queryFn: () => api.get('/health/readiness'),
    refetchInterval: 10_000,
    retry: false,
  });

  let ready: boolean | undefined;
  if (health.isError) ready = false;
  else if (health.isSuccess) ready = true;
  else ready = dashboard.data?.ready;

  let label: string;
  if (health.isError) label = 'Server unreachable';
  else if (ready) label = 'Ready';
  else if (ready === false) label = 'Not ready';
  else label = 'Checking…';

  return { dash: dashboard.data, ready, offline: health.isError, label };
}
