// The switcher lists sessions from two places: what this browser has picked before
// and what the server has actually seen. Recents come first (they are what the user
// works with), live ones follow, and the active session is always present even when
// neither source knows it yet.
export function sessionOptions(recent: string[], live: string[], active: string | null, limit = 12): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const candidate of [...(active ? [active] : []), ...recent, ...live]) {
    const id = candidate.trim();
    if (!id || seen.has(id)) continue;

    seen.add(id);
    out.push(id);
  }

  return out.slice(0, limit);
}
