import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Service, Method } from '../lib/types';

export function useServices() {
  return useQuery({
    queryKey: ['services'],
    queryFn: () => api.get<Service[]>('/services'),
    staleTime: 10_000,
    retry: 2,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
  });
}

export function useServiceMethods(serviceId: string | null) {
  return useQuery({
    queryKey: ['services', serviceId, 'methods'],
    queryFn: () => api.get<Method[]>(`/services/${encodeURIComponent(serviceId!)}/methods`),
    enabled: !!serviceId,
    staleTime: 30_000,
  });
}

/** Fetch details for a single method, including full request/response schemas. */
export function useServiceMethod(serviceId: string | null, methodId: string | null) {
  return useQuery({
    queryKey: ['services', serviceId, 'methods', methodId],
    queryFn: () =>
      api.get<Method>(`/services/${encodeURIComponent(serviceId!)}/methods/${encodeURIComponent(methodId!)}`),
    enabled: !!serviceId && !!methodId,
    staleTime: 30_000,
  });
}
