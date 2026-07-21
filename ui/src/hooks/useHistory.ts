import { useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { CallRecord } from '../lib/types';

export function useHistory(refetchInterval = 0) {
  return useQuery({
    queryKey: ['history'],
    queryFn: () => api.get<CallRecord[]>('/history'),
    refetchInterval,
  });
}

// History scoped to a single endpoint via the server-side ?service&method
// filter (session scope is applied by the server too). Used by Verify so the
// evidence list matches the endpoint's server-counted calls without pulling
// the whole history.
export function useScopedHistory(service: string, method: string, enabled = true) {
  return useQuery({
    queryKey: ['history', 'scoped', service, method],
    queryFn: () => api.get<CallRecord[]>('/history', { service, method }),
    enabled: enabled && !!service && !!method,
  });
}

// Bounded feed — only the most recent `limit` records (server-side ?limit).
export function useRecentHistory(limit = 20, refetchInterval = 0) {
  return useQuery({
    queryKey: ['history', 'recent', limit],
    queryFn: () => api.get<CallRecord[]>('/history', { limit: String(limit) }),
    refetchInterval,
  });
}

// Paged history (newest first) with load-more. Each page skips `offset` newest
// records; X-Total-Count bounds how far back we can go.
export function useInfiniteHistory(pageSize = 100, refetchInterval = 0) {
  return useInfiniteQuery({
    queryKey: ['history', 'infinite', pageSize],
    queryFn: ({ pageParam }) =>
      api.getWithMeta<CallRecord[]>('/history', { limit: String(pageSize), offset: String(pageParam) }),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => {
      if (last.data.length === 0) return undefined; // stop on empty page (see useInfiniteStubs)
      const loaded = pages.reduce((n, p) => n + p.data.length, 0);
      return loaded < last.total ? loaded : undefined;
    },
    refetchInterval,
  });
}
