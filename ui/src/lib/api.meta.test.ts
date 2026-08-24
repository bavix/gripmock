// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest';
import { api, setActiveSession } from './api';

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

describe('session scope', () => {
  it('sends the active session as a header and stops once it is cleared', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true, status: 200,
      headers: { get: () => null },
      text: async () => '{}',
    });
    vi.stubGlobal('fetch', fetchMock);

    setActiveSession('team-a');
    await api.get('/stubs');
    expect(fetchMock.mock.calls[0][1].headers['X-Gripmock-Session']).toBe('team-a');

    setActiveSession(null);
    await api.get('/stubs');
    expect(fetchMock.mock.calls[1][1].headers).not.toHaveProperty('X-Gripmock-Session');

    vi.unstubAllGlobals();
  });
});

describe('postBinary', () => {
  it('sends the blob untouched so a descriptor is not JSON-stringified into "{}"', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true, status: 200,
      headers: { get: () => null },
      text: async () => '{}',
    });
    vi.stubGlobal('fetch', fetchMock);

    const blob = new Blob([new Uint8Array([10, 20, 30])]);
    await api.postBinary('/descriptors', blob);

    const [, init] = fetchMock.mock.calls[0];
    expect(init.body).toBe(blob);
    expect(init.headers['Content-Type']).toBe('application/octet-stream');

    vi.unstubAllGlobals();
  });
});
