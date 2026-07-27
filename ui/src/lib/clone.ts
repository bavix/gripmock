// Hand a stub to the create form without stuffing it into the URL (which
// breaks on large stubs / URL-length limits). Stored in sessionStorage so it
// survives the client-side navigation and a refresh.
const KEY = 'gripmock.clone';

// Keys that can corrupt an object prototype (prototype pollution).
const DANGEROUS_KEYS = new Set(['__proto__', 'constructor', 'prototype']);

// Stubs are user-controlled data (any client can POST arbitrary JSON via the
// API), so a cloned stub is tainted. Recursively drop prototype-pollution keys
// before it touches sessionStorage — and again when read back — so a poisoned
// stub cannot corrupt a prototype once the clone is parsed into the form.
function sanitize(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sanitize);

  if (value && typeof value === 'object') {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      if (DANGEROUS_KEYS.has(k)) continue;
      out[k] = sanitize(v);
    }
    return out;
  }

  return value;
}

export function stashClone(stub: unknown): void {
  try { sessionStorage.setItem(KEY, JSON.stringify(sanitize(stub))); } catch { /* ignore */ }
}

export function takeClone(): Record<string, unknown> | null {
  try {
    const v = sessionStorage.getItem(KEY);
    sessionStorage.removeItem(KEY);
    return v ? (sanitize(JSON.parse(v)) as Record<string, unknown>) : null;
  } catch { return null; }
}
