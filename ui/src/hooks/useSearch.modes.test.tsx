// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import { renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { useSmartSearch } from './useSearch';
import { api } from '../lib/api';

vi.mock('../lib/api', () => ({ api: { get: vi.fn(), post: vi.fn() } }));

const mockApi = api as unknown as { get: Mock; post: Mock };

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

function runSearch() {
  return renderHook(() => useSmartSearch(), { wrapper }).result.current.search;
}

beforeEach(() => { mockApi.get.mockReset(); mockApi.post.mockReset(); });

describe('useSmartSearch.search', () => {
  it('UUID → GET /stubs/:id (encoded), single result', async () => {
    mockApi.get.mockResolvedValueOnce({ id: 'abc12345ff', service: 'S', method: 'M' });

    const r = await runSearch()('abc12345ff');

    expect(mockApi.get).toHaveBeenCalledWith('/stubs/abc12345ff');
    expect(r.results).toHaveLength(1);
    expect(r.error).toBeUndefined();
  });

  it('UUID not found → error surfaced, no results', async () => {
    mockApi.get.mockRejectedValueOnce(new Error('404'));

    const r = await runSearch()('deadbeef00');

    expect(r.results).toEqual([]);
    expect(r.error).toBe('Stub not found by this ID');
  });

  it('endpoint (service/method) → GET /stubs with filters, no inspect', async () => {
    mockApi.get.mockResolvedValueOnce([{ id: '1' }, { id: '2' }]);

    const r = await runSearch()('pkg.Svc/Method');

    expect(mockApi.get).toHaveBeenCalledWith('/stubs', { service: 'pkg.Svc', method: 'Method' });
    expect(mockApi.post).not.toHaveBeenCalled();
    expect(r.results.map((s) => s.id)).toEqual(['1', '2']);
  });

  it('payload + endpoint → inspect filters stubs to matched ids', async () => {
    mockApi.get.mockResolvedValueOnce([{ id: '1' }, { id: '2' }, { id: '3' }]);
    mockApi.post.mockResolvedValueOnce({
      candidates: [{ id: '2', matched: true }, { id: '1', matched: false }],
      matchedStubId: '2',
    });

    const r = await runSearch()('pkg.Svc/Method {"a":1}');

    expect(mockApi.post).toHaveBeenCalledWith('/stubs/inspect', { service: 'pkg.Svc', method: 'Method', input: [{ a: 1 }] });
    expect(r.results.map((s) => s.id)).toEqual(['2']);
  });

  it('payload + endpoint with no match → shows the endpoint candidates', async () => {
    mockApi.get.mockResolvedValueOnce([{ id: '1' }, { id: '2' }]);
    mockApi.post.mockResolvedValueOnce({ candidates: [{ id: '1', matched: false }], matchedStubId: '' });

    const r = await runSearch()('pkg.Svc/Method {"a":1}');

    expect(r.results.map((s) => s.id)).toEqual(['1', '2']);
  });

  it('payload without an endpoint → empty (defers to server filter), no calls', async () => {
    const r = await runSearch()('{"a":1}');

    expect(r.results).toEqual([]);
    expect(r.error).toBeUndefined();
    expect(mockApi.get).not.toHaveBeenCalled();
    expect(mockApi.post).not.toHaveBeenCalled();
  });

  it('JSON payload whose values contain dots is NOT parsed as an endpoint (regression)', async () => {
    // A dotted value (hostname/version) inside a JSON payload must not be split
    // into service.method and fire bogus /stubs + /stubs/inspect calls + a toast.
    const r = await runSearch()('{"host":"example.com","v":"1.2.3"}');

    expect(r.results).toEqual([]);
    expect(r.error).toBeUndefined();
    expect(mockApi.get).not.toHaveBeenCalled();
    expect(mockApi.post).not.toHaveBeenCalled();
  });

  it('endpoint fetch failure → error surfaced', async () => {
    mockApi.get.mockRejectedValueOnce(new Error('boom'));

    const r = await runSearch()('pkg.Svc/Method');

    expect(r.results).toEqual([]);
    expect(r.error).toBe('boom');
  });
});
