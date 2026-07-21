// Hand a stub to the create form without stuffing it into the URL (which
// breaks on large stubs / URL-length limits). Stored in sessionStorage so it
// survives the client-side navigation and a refresh.
const KEY = 'gripmock.clone';

export function stashClone(stub: unknown): void {
  try { sessionStorage.setItem(KEY, JSON.stringify(stub)); } catch { /* ignore */ }
}

export function takeClone(): Record<string, unknown> | null {
  try {
    const v = sessionStorage.getItem(KEY);
    sessionStorage.removeItem(KEY);
    return v ? JSON.parse(v) : null;
  } catch { return null; }
}
