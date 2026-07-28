// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest';
import { api } from './api';

afterEach(() => { vi.restoreAllMocks(); });

describe('api.getWithMeta empty-body handling', () => {
  it('does not throw on a 204 response (regression: res.json() would reject)', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })));

    const r = await api.getWithMeta('/stubs');
    expect(r.data).toBeUndefined();
    expect(r.total).toBe(0);
  });

  it('parses body + reads X-Total-Count when present', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: '1' }, { id: '2' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json', 'X-Total-Count': '17' },
      }),
    ));

    const r = await api.getWithMeta<{ id: string }[]>('/stubs');
    expect(r.data).toHaveLength(2);
    expect(r.total).toBe(17);
  });

  it('falls back to loaded length when X-Total-Count is missing', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: '1' }]), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    ));

    const r = await api.getWithMeta<{ id: string }[]>('/stubs');
    expect(r.total).toBe(1);
  });
});
