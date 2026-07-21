import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface SearchMatch {
  id: string;
  service: string;
  method: string;
  priority: number;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  source?: string;
  score?: number;
}

export interface SmartSearchResult {
  mode: 'id' | 'endpoint' | 'payload' | 'unknown';
  query: string;
  results: SearchMatch[];
  matched?: SearchMatch;
  error?: string;
}

// Exported for unit tests — pure intent-detection helpers.
export function isUuid(v: string): boolean {
  return /^[0-9a-f]{8,}$/i.test(v.replace(/-/g, ''));
}

export function extractPayload(query: string): Record<string, unknown> | undefined {
  const jsonMatch = query.match(/\{.*\}/s);
  if (jsonMatch) {
    try { return JSON.parse(jsonMatch[0]); } catch {}
  }
  return undefined;
}

export function extractServiceMethod(query: string): { service?: string; method?: string } {
  const parts = query.split(/\s+/);
  for (const p of parts) {
    if (p.includes('/') || p.includes('.')) {
      const segments = p.split('/');
      if (segments.length === 2) return { service: segments[0], method: segments[1] };
      const dotSegments = p.split('.');
      if (dotSegments.length >= 2) {
        const method = dotSegments.pop()!;
        return { service: dotSegments.join('.'), method };
      }
    }
  }
  return {};
}

export function useSmartSearch() {
  const searchMut = useMutation({
    mutationFn: (req: { service?: string; method?: string; data?: Record<string, unknown> }) =>
      api.post<{ id?: string; service: string; method: string; data?: unknown }>('/stubs/search', req),
  });

  return {
    search: async (query: string): Promise<SmartSearchResult> => {
      const trimmed = query.trim();
      if (!trimmed) return { mode: 'unknown', query, results: [] };

      // Mode 1: by ID
      if (isUuid(trimmed)) {
        try {
          const stub = await api.get<SearchMatch>(`/stubs/${encodeURIComponent(trimmed)}`);
          return { mode: 'id', query, results: [stub], matched: stub };
        } catch {
          return { mode: 'id', query, results: [], error: 'Stub not found by this ID' };
        }
      }

      // Mode 2: by payload (JSON detected)
      const payload = extractPayload(trimmed);
      const { service, method } = extractServiceMethod(trimmed);

      if (payload || (service && method)) {
        try {
          const res = await searchMut.mutateAsync({ service, method, data: payload });
          // The response format depends on the API
          return { mode: 'payload', query, results: res as any, matched: (res as any).id ? (res as any) : undefined };
        } catch (err) {
          return { mode: 'payload', query, results: [], error: (err as Error).message };
        }
      }

      // Mode 3: by endpoint (service/method pattern)
      if (service || method) {
        try {
          const stubs = await api.get<SearchMatch[]>('/stubs', { service, method });
          return { mode: 'endpoint', query, results: stubs };
        } catch (err) {
          return { mode: 'endpoint', query, results: [], error: (err as Error).message };
        }
      }

      return { mode: 'unknown', query, results: [] };
    },
    isSearching: searchMut.isPending,
  };
}
