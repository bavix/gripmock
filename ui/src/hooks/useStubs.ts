import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Stub } from '../lib/types';

export function useStubs(params?: { service?: string; method?: string; source?: string }) {
  return useQuery({
    queryKey: ['stubs', params],
    queryFn: () => api.get<Stub[]>('/stubs', params as Record<string, string>),
  });
}

export interface StubListFilters {
  service?: string;
  method?: string;
  source?: string;
  q?: string;
  sort?: string;
}

const clean = (f: StubListFilters): Record<string, string | undefined> => ({
  service: f.service || undefined,
  method: f.method || undefined,
  source: f.source || undefined,
  q: f.q || undefined,
  sort: f.sort || undefined,
});

// One page of stubs with the server-side total (X-Total-Count).
export function useStubsPage(f: StubListFilters, limit: number, offset: number) {
  const cf = clean(f);
  return useQuery({
    // Key on the CLEANED filters so logically-equal filters (empty string vs
    // undefined) share one cache entry instead of double-fetching.
    queryKey: ['stubs', 'page', cf, limit, offset],
    queryFn: () => api.getWithMeta<Stub[]>('/stubs', { ...cf, limit: String(limit), offset: String(offset) }),
    placeholderData: (prev) => prev, // keep previous page while the next loads
  });
}

// Infinite scrolling feed of stubs for the cards view.
export function useInfiniteStubs(f: StubListFilters, pageSize = 60) {
  const cf = clean(f);
  return useInfiniteQuery({
    queryKey: ['stubs', 'infinite', cf, pageSize],
    queryFn: ({ pageParam }) =>
      api.getWithMeta<Stub[]>('/stubs', { ...cf, limit: String(pageSize), offset: String(pageParam) }),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => {
      // Stop on an empty page — guards against an offset loop if `total` shrank
      // mid-scroll (concurrent deletes) while loaded never catches up.
      if (last.data.length === 0) return undefined;
      const loaded = pages.reduce((n, p) => n + p.data.length, 0);
      return loaded < last.total ? loaded : undefined;
    },
  });
}

export function useStub(id: string) {
  return useQuery({
    queryKey: ['stubs', id],
    queryFn: () => api.get<Stub>(`/stubs/${id}`),
    enabled: !!id,
  });
}

export function useUsedStubs() {
  return useQuery({ queryKey: ['stubs', 'used'], queryFn: () => api.get<Stub[]>('/stubs/used') });
}

export function useUnusedStubs() {
  return useQuery({ queryKey: ['stubs', 'unused'], queryFn: () => api.get<Stub[]>('/stubs/unused') });
}

export function useCreateStub() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Stub> | Partial<Stub>[]) => api.post<string[]>('/stubs', data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['stubs'] }),
  });
}

export function useUpdateStub() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Stub>) => api.post<string[]>('/stubs', [data]),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['stubs'] }),
  });
}
