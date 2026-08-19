// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { usePurgeHistory } from './useHistory';
import { api } from '../lib/api';

vi.mock('../lib/api', () => ({ api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() } }));

const mockApi = api as unknown as { delete: Mock };

let client: QueryClient;

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  mockApi.delete.mockReset();
  client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
});

describe('usePurgeHistory', () => {
  it('deletes /history and reports what the server removed', async () => {
    mockApi.delete.mockResolvedValueOnce({ deletedCount: 7, session: 'team-a' });

    const { result } = renderHook(() => usePurgeHistory(), { wrapper });

    result.current.mutate();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.delete).toHaveBeenCalledWith('/history');
    expect(result.current.data).toEqual({ deletedCount: 7, session: 'team-a' });
  });

  it('refreshes the history and dashboard views once the purge lands', async () => {
    mockApi.delete.mockResolvedValueOnce({ deletedCount: 0 });

    const invalidated: unknown[] = [];
    vi.spyOn(client, 'invalidateQueries').mockImplementation((filters) => {
      invalidated.push(filters?.queryKey);

      return Promise.resolve();
    });

    const { result } = renderHook(() => usePurgeHistory(), { wrapper });

    result.current.mutate();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidated).toEqual([['history'], ['dashboard']]);
  });
});
