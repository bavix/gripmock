// Guarded so the module is importable outside the browser (vitest node env).
const cfg = (typeof window !== 'undefined' ? window.__GRIPMOCK_CONFIG__ : undefined) || {};
const DEFAULT_BASE = cfg.apiBase || '/api';
const STORAGE_KEY = 'gripmock.ui.apiUrl';

// Error carrying the HTTP status and parsed response body so callers can read
// structured fields (e.g. verify returns {expected, actual} with a 400).
export class ApiError extends Error {
  status?: number;
  body?: Record<string, unknown>;
}

// The base URL may be user-configured (persisted in localStorage), so validate
// it before it ever reaches fetch(): accept only well-formed http(s) URLs
// (absolute, or relative resolved against the current origin); otherwise fall
// back to the default base. Keeps normal behavior for every valid value.
function isValidBase(u: string): boolean {
  try {
    const origin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost';
    const parsed = new URL(u, origin);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch { return false; }
}

function base(): string {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && isValidBase(stored)) return stored;
  } catch {}
  return DEFAULT_BASE;
}

export function getApiUrl() { return base(); }
export function setApiUrl(u: string) { try { localStorage.setItem(STORAGE_KEY, u); } catch {} }
export function resetApiUrl() { try { localStorage.removeItem(STORAGE_KEY); } catch {} }

// Shared transport: builds headers (session, content-type), issues the request,
// and throws ApiError (with status + parsed body) on a non-2xx response.
async function sendRequest(path: string, opts: RequestInit = {}): Promise<Response> {
  const h: Record<string, string> = { Accept: 'application/json', ...(opts.headers as Record<string, string>) };
  if (!(opts.body instanceof Blob)) h['Content-Type'] = 'application/json';
  h['X-GripMock-RequestInternal'] = '92b4d5a9-c74b-4ac0-989c-717f80acba22';
  try {
    const s = localStorage.getItem('gripmock.ui.session');
    if (s) h['X-Gripmock-Session'] = s;
  } catch {}

  const res = await fetch(`${base()}${path.startsWith('/') ? '' : '/'}${path}`, { ...opts, headers: h });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    let body: Record<string, unknown> | undefined;
    try { body = await res.json(); msg = (body?.error as string) || (body?.message as string) || msg; } catch {}
    // Preserve structured fields (e.g. verify's {expected, actual}) + status on the Error.
    throw Object.assign(new ApiError(msg), { status: res.status, body });
  }

  return res;
}

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await sendRequest(path, opts);
  if (res.status === 204 || res.headers.get('content-length') === '0') return undefined as T;
  const text = await res.text();
  return text ? JSON.parse(text) : undefined as T;
}

function qs(params?: Record<string, string | undefined>): string {
  if (!params) return '';
  const e = Object.entries(params).filter(([, v]) => v !== undefined).map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v!)}`);
  return e.length ? '?' + e.join('&') : '';
}

// GET that also returns the X-Total-Count header (server-side pagination).
async function requestWithMeta<T>(path: string): Promise<{ data: T; total: number }> {
  const res = await sendRequest(path);
  const data = (await res.json()) as T;
  // Fall back to the loaded length when the header is missing OR malformed
  // (a NaN/0-from-null total would silently stop infinite pagination).
  const fallback = Array.isArray(data) ? data.length : 0;
  const raw = res.headers.get('X-Total-Count');
  const parsed = raw != null ? Number(raw) : NaN;
  const total = Number.isFinite(parsed) ? parsed : fallback;
  return { data, total };
}

export const api = {
  get: <T>(p: string, params?: Record<string, string | undefined>) => request<T>(`${p}${qs(params)}`),
  getWithMeta: <T>(p: string, params?: Record<string, string | undefined>) => requestWithMeta<T>(`${p}${qs(params)}`),
  post: <T>(p: string, body?: unknown) => request<T>(p, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(p: string) => request<T>(p, { method: 'DELETE' }),
  postBinary: <T>(p: string, data: Blob) => request<T>(p, { method: 'POST', body: data, headers: { 'Content-Type': 'application/octet-stream' } }),
};
