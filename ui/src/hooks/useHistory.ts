import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import { nextPageOffset } from '../lib/pagination';
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
// records; X-Total-Count bounds how far back we can go. When errorOnly is set the
// server returns only errored calls, so the Errors tab covers ALL errors — not
// just those in already-loaded pages.
export function useInfiniteHistory(pageSize = 100, refetchInterval = 0, errorOnly = false) {
  return useInfiniteQuery({
    queryKey: ['history', 'infinite', pageSize, errorOnly],
    queryFn: ({ pageParam }) =>
      api.getWithMeta<CallRecord[]>('/history', {
        limit: String(pageSize), offset: String(pageParam),
        ...(errorOnly ? { error: 'true' } : {}),
      }),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => {
      const loaded = pages.reduce((n, p) => n + p.data.length, 0);
      return nextPageOffset(last.data.length, loaded, pages[0]?.total ?? last.total);
    },
    refetchInterval,
  });
}

// Accurate server-wide error count via X-Total-Count of an error-only query,
// without loading any records (limit=1). Used for the honest "N errors" badge.
export function useHistoryErrorCount(refetchInterval = 0) {
  return useQuery({
    queryKey: ['history', 'errorCount'],
    queryFn: async () => (await api.getWithMeta<CallRecord[]>('/history', { error: 'true', limit: '1' })).total,
    refetchInterval,
  });
}

// Clears recorded calls. The server scopes the purge to the active session when
// one is set, so it never wipes another session's evidence.
export function usePurgeHistory() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.delete<{ deletedCount: number; session?: string }>('/history'),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['history'] });
      qc.invalidateQueries({ queryKey: ['dashboard'] });
    },
  });
}
