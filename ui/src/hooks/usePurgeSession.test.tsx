// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { usePurgeSession } from './useSessions';
import { api } from '../lib/api';

vi.mock('../lib/api', () => ({ api: { get: vi.fn(), post: vi.fn(), delete: vi.fn() } }));

const mockApi = api as unknown as { delete: Mock };

let client: QueryClient;

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  mockApi.delete.mockReset().mockResolvedValue(undefined);
  client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
});

describe('usePurgeSession', () => {
  it('sends the session header explicitly so the active session is irrelevant', async () => {
    const { result } = renderHook(() => usePurgeSession(), { wrapper });

    result.current.mutate('team-a');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockApi.delete).toHaveBeenCalledTimes(2);
    expect(mockApi.delete).toHaveBeenNthCalledWith(1, '/stubs', { 'X-Gripmock-Session': 'team-a' });
    expect(mockApi.delete).toHaveBeenNthCalledWith(2, '/history', { 'X-Gripmock-Session': 'team-a' });
  });

  it('surfaces a failure instead of reporting a partial purge as done', async () => {
    mockApi.delete.mockRejectedValueOnce(new Error('boom'));

    const { result } = renderHook(() => usePurgeSession(), { wrapper });

    result.current.mutate('team-a');

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(mockApi.delete).toHaveBeenCalledTimes(1);
  });

  it('refreshes every view a purge can change', async () => {
    const invalidated: unknown[] = [];
    vi.spyOn(client, 'invalidateQueries').mockImplementation((filters) => {
      invalidated.push(filters?.queryKey);

      return Promise.resolve();
    });

    const { result } = renderHook(() => usePurgeSession(), { wrapper });

    result.current.mutate('team-a');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(invalidated).toEqual([['stubs'], ['history'], ['sessions'], ['dashboard']]);
  });
});
