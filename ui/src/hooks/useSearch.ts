import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Stub, InspectReport } from '../lib/types';

export interface SmartSearchResult {
  query: string;
  results: Stub[];
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
  // /stubs/inspect resolves WHICH stub actually matches a payload; the candidate
  // set is scoped by service+method, so payload search needs an endpoint too.
  const inspectMut = useMutation({
    mutationFn: (req: { service: string; method: string; input?: Record<string, unknown>[] }) =>
      api.post<InspectReport>('/stubs/inspect', req),
  });

  return {
    // Returns full Stub[] so the caller can render real stub cards. Empty
    // results with no error means "not resolvable here" — the caller keeps the
    // server-side substring filter (serverQ) as the fallback.
    search: async (query: string): Promise<SmartSearchResult> => {
      const trimmed = query.trim();
      if (!trimmed) return { query, results: [] };

      // Mode 1: by ID — direct stub lookup.
      if (isUuid(trimmed)) {
        try {
          const stub = await api.get<Stub>(`/stubs/${encodeURIComponent(trimmed)}`);
          return { query, results: [stub] };
        } catch {
          return { query, results: [], error: 'Stub not found by this ID' };
        }
      }

      const payload = extractPayload(trimmed);
      const { service, method } = extractServiceMethod(trimmed);

      // Mode 2/3: endpoint (service/method), optionally narrowed by a payload.
      if (service && method) {
        try {
          const stubs = await api.get<Stub[]>('/stubs', { service, method });
          if (!payload) return { query, results: stubs };

          // Ask the matcher which of those stubs the payload actually hits.
          const report = await inspectMut.mutateAsync({ service, method, input: [payload] });
          const matchedIds = new Set((report.candidates ?? []).filter((c) => c.matched).map((c) => c.id));
          if (report.matchedStubId) matchedIds.add(report.matchedStubId);
          const matched = stubs.filter((s) => matchedIds.has(s.id));
          // No match → show the endpoint's stubs so the user sees the candidates.
          return { query, results: matched.length ? matched : stubs };
        } catch (err) {
          return { query, results: [], error: (err as Error).message };
        }
      }

      // Payload without an endpoint can't be inspected (no method to scope
      // candidates) — defer to the server substring filter.
      return { query, results: [] };
    },
  };
}
