import type { Stub, StubInput } from './types';

export const isRequestStream = (t?: string) => t === 'client_streaming' || t === 'bidi_streaming';
export const isResponseStream = (t?: string) => t === 'server_streaming' || t === 'bidi_streaming';

// Ordered request-message matchers. Client/bidi streaming use `inputs[]` (the
// sequence of streamed messages); unary/server-streaming use the single `input`.
export function requestMessages(stub: Stub): StubInput[] {
  if (stub.inputs && stub.inputs.length > 0) return stub.inputs;
  return hasContent(stub.input) ? [stub.input] : [];
}

// Ordered response messages. Server/bidi streaming use `output.stream[]`;
// unary/client-streaming use the single `output.data`.
export function responseMessages(stub: Stub): unknown[] {
  const o = stub.output;
  if (o?.stream && o.stream.length > 0) return o.stream;
  if (o?.data !== undefined && o.data !== null) return [o.data];
  return [];
}

export type MatcherKind = 'equals' | 'contains' | 'matches' | 'glob' | 'anyOf';
export interface MatcherEntry { kind: MatcherKind; value: unknown; }

export function matcherEntries(m?: StubInput | Record<string, unknown>): MatcherEntry[] {
  if (!m) return [];
  const out: MatcherEntry[] = [];
  for (const k of ['equals', 'contains', 'matches', 'glob'] as const) {
    const v = (m as Record<string, unknown>)[k];
    if (hasContent(v)) out.push({ kind: k, value: v });
  }
  const anyOf = (m as StubInput).anyOf;
  if (anyOf && anyOf.length > 0) out.push({ kind: 'anyOf', value: anyOf });
  return out;
}

// Translate a glob pattern (*, ?) to an anchored RegExp. Best-effort — mirrors
// the server's glob semantics closely enough for a field-level diagnosis.
export function globToRegExp(glob: string): RegExp {
  const escaped = glob.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*').replace(/\?/g, '.');
  return new RegExp(`^${escaped}$`);
}

export interface FieldRule {
  field: string;
  kind: MatcherKind; // equals | contains | matches | glob
  expected: unknown;
  actual: unknown;
  ok: boolean;
}

// Per-field eval across matcher kinds (equals/contains/matches/glob + anyOf).
// Client-side best-effort — the server remains authoritative.
export function evalMatcherFields(payload: Record<string, unknown>, matcher?: StubInput | Record<string, unknown>): FieldRule[] {
  if (!matcher) return [];
  const rows: FieldRule[] = [];

  for (const kind of ['equals', 'contains', 'matches', 'glob'] as const) {
    const rules = (matcher as Record<string, unknown>)[kind] as Record<string, unknown> | undefined;
    if (!hasContent(rules)) continue;

    for (const [field, expected] of Object.entries(rules as Record<string, unknown>)) {
      const actual = payload[field];
      rows.push({ field, kind, expected, actual, ok: fieldMatches(kind, expected, actual) });
    }
  }

  // anyOf: the stub matches if ANY alternative matches. Diagnose against the
  // CLOSEST alternative (fewest field mismatches) so the diff isn't misleading.
  const anyOf = (matcher as StubInput).anyOf;
  if (rows.length === 0 && anyOf && anyOf.length > 0) {
    let best: FieldRule[] | null = null;
    let bestFails = Infinity;
    for (const alt of anyOf) {
      const altRows = evalMatcherFields(payload, alt as Record<string, unknown>);
      const fails = altRows.filter((r) => !r.ok).length;
      if (fails < bestFails) { bestFails = fails; best = altRows; }
      if (fails === 0) break; // a matching alternative — no mismatch to show
    }
    return best ?? [];
  }

  return rows;
}

function fieldMatches(kind: MatcherKind, expected: unknown, actual: unknown): boolean {
  if (actual === undefined) return false;

  switch (kind) {
    case 'equals':
      return JSON.stringify(expected) === JSON.stringify(actual);
    case 'contains':
      // String → substring. Array → every expected element present. Otherwise
      // (scalars, objects) fall back to equality — a scalar only "contains"
      // an equal scalar (avoids "15" matching contains 5).
      if (typeof expected === 'string' && typeof actual === 'string') return actual.includes(expected);
      if (Array.isArray(expected) && Array.isArray(actual)) {
        return expected.every((e) => actual.some((a) => JSON.stringify(a) === JSON.stringify(e)));
      }
      return JSON.stringify(expected) === JSON.stringify(actual);
    case 'matches':
      try { return new RegExp(String(expected)).test(String(actual)); } catch { return false; }
    case 'glob':
      try { return globToRegExp(String(expected)).test(String(actual)); } catch { return false; }
    default:
      return false;
  }
}

// Recursively strip null/undefined values and empty objects/arrays. Used for
// DISPLAY only (previews, read-only viewers) so noise like
// {"equals":null,"contains":null,"matches":null} never reaches the user.
// Never use on form default values — an intentional empty {} would be dropped.
export function prune<T>(value: T): T | undefined {
  if (value === null || value === undefined) return undefined;
  if (typeof value === 'string') return value === '' ? undefined : value;
  if (Array.isArray(value)) {
    const arr = value.map(prune).filter((v) => v !== undefined);
    return (arr.length ? arr : undefined) as T | undefined;
  }
  if (typeof value === 'object') {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value)) {
      const pv = prune(v);
      if (pv !== undefined) out[k] = pv;
    }
    return (Object.keys(out).length ? out : undefined) as T | undefined;
  }
  return value;
}

export function hasContent(value: unknown): boolean {
  return prune(value) !== undefined;
}

export function prettyJson(value: unknown): string {
  const p = prune(value);
  return p === undefined ? '' : JSON.stringify(p, null, 2);
}

export function compactPreview(value: unknown, len = 48): string {
  const p = prune(value);
  if (p === undefined) return '—';
  const s = JSON.stringify(p).replace(/\s+/g, ' ').trim();
  return s.length > len ? s.slice(0, len) + '…' : s;
}

// Matcher types present on a stub. Input matchers come first (they define the
// stub); headers is a secondary signal listed last. Primary type = types[0].
export function matcherTypes(s: Stub): string[] {
  const t: string[] = [];
  if (hasContent(s.input?.equals)) t.push('equals');
  if (hasContent(s.input?.contains)) t.push('contains');
  if (hasContent(s.input?.matches)) t.push('matches');
  if (hasContent(s.input?.glob)) t.push('glob');
  if (s.inputs && s.inputs.length > 0) t.push('anyOf');
  if (hasContent(s.headers)) t.push('headers');
  return t.length ? t : ['any'];
}

export const MATCHER_COLORS: Record<string, string> = {
  equals: '#3b82f6', contains: '#22c55e', matches: '#f59e0b',
  glob: '#ec4899', anyOf: '#a855f7', any: '#64748b', headers: '#06b6d4',
};

export type OutputKind = { label: string; color: string };

// A stub's `service` may be the fully-qualified name (with package, e.g.
// "ecommerce.EcommerceService") OR the bare service name ("EcommerceService").
// Match either form against a service's id (FQN) and short name.
export function serviceRefMatches(stubService: string, serviceId: string, serviceName?: string): boolean {
  return stubService === serviceId || (!!serviceName && stubService === serviceName);
}

// Build an example request (payload + headers) that this stub would match,
// derived from its matchers. Used to prefill Test/Inspect.
export function stubRequestExample(stub: Stub): { payload: string; headers: string } {
  const i = stub.input as Record<string, unknown> | undefined;
  const payload = i?.equals ?? i?.contains ?? i?.matches ?? i?.glob
    ?? (stub.inputs?.[0] as Record<string, unknown> | undefined)?.equals ?? {};
  const h = stub.headers as Record<string, unknown> | undefined;
  const headers = h?.equals ?? h?.contains ?? h?.matches ?? {};
  return { payload: JSON.stringify(payload, null, 2), headers: JSON.stringify(headers, null, 2) };
}

// Short label + color for a gRPC method's streaming kind.
export function streamKind(methodType?: string): { label: string; full: string; color: string } {
  switch (methodType) {
    case 'server_streaming': return { label: 'SS', full: 'server streaming', color: '#9333ea' };
    case 'client_streaming': return { label: 'CS', full: 'client streaming', color: '#d97706' };
    case 'bidi_streaming': return { label: 'BD', full: 'bidirectional streaming', color: '#e5484d' };
    case 'unary': return { label: 'U', full: 'unary', color: '#5570e6' };
    default: return { label: '?', full: 'unknown', color: '#64748b' };
  }
}

// Other stubs on the same service+method (potential conflicts), sorted by the
// backend selection order (priority desc — higher priority is evaluated first).
export function methodPeers(stub: Stub, all: Stub[]): Stub[] {
  return all
    .filter((s) => s.id !== stub.id && s.service === stub.service && s.method === stub.method)
    .sort((a, b) => b.priority - a.priority);
}

// Peers that would be evaluated before this stub (higher priority) and could
// shadow it if their matcher also accepts the request.
export function shadowers(stub: Stub, all: Stub[]): Stub[] {
  return methodPeers(stub, all).filter((s) => s.priority > stub.priority);
}

// Classify a stub's output into a single badge kind.
export function outputKind(s: Stub): OutputKind {
  const o = s.output;
  if (o?.stream?.length) return { label: `Stream ${o.stream.length}`, color: '#06b6d4' };
  if (o?.error || (o?.code && o.code > 0)) return { label: 'Error', color: '#ef4444' };
  if (o?.data !== undefined) return { label: 'Data', color: '#22c55e' };
  return { label: 'Empty', color: '#64748b' };
}
