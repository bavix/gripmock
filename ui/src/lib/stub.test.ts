import { describe, it, expect } from 'vitest';
import {
  prune, hasContent, compactPreview, matcherTypes, matcherEntries,
  requestMessages, responseMessages, isRequestStream, isResponseStream,
  serviceRefMatches, methodPeers, shadowers, streamKind, outputKind,
  evalMatcherFields, globToRegExp,
} from './stub';
import type { Stub, StubInput } from './types';

const stub = (over: Partial<Stub>): Stub => ({
  id: 'id-1', service: 'pkg.Svc', method: 'M', priority: 0,
  input: {}, output: {}, ...over,
} as Stub);

describe('prune', () => {
  it('strips nulls, empty objects/arrays and empty strings', () => {
    expect(prune({ equals: null, contains: null, matches: null })).toBeUndefined();
    expect(prune({ a: { b: null }, c: [] })).toBeUndefined();
    expect(prune({ a: 1, b: null })).toEqual({ a: 1 });
    expect(prune('')).toBeUndefined();
  });

  it('preserves 0 and false', () => {
    expect(prune({ n: 0, f: false })).toEqual({ n: 0, f: false });
    expect(prune([0, false])).toEqual([0, false]);
  });

  it('prunes nested arrays recursively', () => {
    expect(prune([{ a: null }, { b: 1 }])).toEqual([{ b: 1 }]);
  });
});

describe('hasContent / compactPreview', () => {
  it('hasContent false for all-null matcher', () => {
    expect(hasContent({ equals: null, contains: null })).toBe(false);
    expect(hasContent({ equals: { x: 1 } })).toBe(true);
  });

  it('compactPreview truncates and dashes empties', () => {
    expect(compactPreview(undefined)).toBe('—');
    expect(compactPreview({ equals: null })).toBe('—');
    const long = { key: 'x'.repeat(100) };
    expect(compactPreview(long, 20).endsWith('…')).toBe(true);
  });
});

describe('matcherTypes / matcherEntries', () => {
  it('detects input matcher kinds in order, headers last', () => {
    const s = stub({ input: { equals: { a: 1 }, matches: { b: '.*' } }, headers: { contains: { h: 'v' } } });
    expect(matcherTypes(s)).toEqual(['equals', 'matches', 'headers']);
  });

  it('anyOf via inputs[] and fallback "any"', () => {
    expect(matcherTypes(stub({ inputs: [{ equals: { a: 1 } }] }))).toEqual(['anyOf']);
    expect(matcherTypes(stub({}))).toEqual(['any']);
  });

  it('matcherEntries lists meaningful rules only', () => {
    const e = matcherEntries({ equals: { a: 1 }, contains: null, matches: undefined, anyOf: [{ equals: { b: 2 } }] });
    expect(e.map((x) => x.kind)).toEqual(['equals', 'anyOf']);
  });

  it('matcherEntries empty for nullish', () => {
    expect(matcherEntries(undefined)).toEqual([]);
    expect(matcherEntries({ equals: null })).toEqual([]);
  });
});

describe('requestMessages / responseMessages', () => {
  it('unary: single input, single data', () => {
    const s = stub({ input: { equals: { a: 1 } }, output: { data: { ok: true } } });
    expect(requestMessages(s)).toHaveLength(1);
    expect(responseMessages(s)).toEqual([{ ok: true }]);
  });

  it('client-streaming: inputs[] wins over empty input', () => {
    const s = stub({ input: { equals: null } as unknown as Stub['input'], inputs: [{ equals: { m: 1 } }, { equals: { m: 2 } }] });
    expect(requestMessages(s)).toHaveLength(2);
  });

  it('server-streaming: output.stream wins over data', () => {
    const s = stub({ output: { stream: [{ n: 1 }, { n: 2 }], data: undefined } });
    expect(responseMessages(s)).toEqual([{ n: 1 }, { n: 2 }]);
  });

  it('empty matcher input yields no request messages', () => {
    expect(requestMessages(stub({ input: { equals: null } as unknown as Stub['input'] }))).toHaveLength(0);
  });
});

describe('stream direction helpers', () => {
  it('isRequestStream only for client/bidi', () => {
    expect(isRequestStream('client_streaming')).toBe(true);
    expect(isRequestStream('bidi_streaming')).toBe(true);
    expect(isRequestStream('server_streaming')).toBe(false);
    expect(isRequestStream('unary')).toBe(false);
  });

  it('isResponseStream only for server/bidi', () => {
    expect(isResponseStream('server_streaming')).toBe(true);
    expect(isResponseStream('bidi_streaming')).toBe(true);
    expect(isResponseStream('client_streaming')).toBe(false);
  });

  it('streamKind labels', () => {
    expect(streamKind('unary').label).toBe('U');
    expect(streamKind('bidi_streaming').label).toBe('BD');
    expect(streamKind(undefined).label).toBe('?');
  });
});

describe('serviceRefMatches', () => {
  it('matches FQN and bare name', () => {
    expect(serviceRefMatches('pkg.Svc', 'pkg.Svc', 'Svc')).toBe(true);
    expect(serviceRefMatches('Svc', 'pkg.Svc', 'Svc')).toBe(true);
    expect(serviceRefMatches('Other', 'pkg.Svc', 'Svc')).toBe(false);
  });
});

describe('methodPeers / shadowers', () => {
  const a = stub({ id: 'a', priority: 0 });
  const b = stub({ id: 'b', priority: 5 });
  const c = stub({ id: 'c', method: 'Other' });

  it('peers on same service+method sorted by priority desc, self excluded', () => {
    expect(methodPeers(a, [a, b, c]).map((s) => s.id)).toEqual(['b']);
  });

  it('shadowers are strictly higher-priority peers', () => {
    expect(shadowers(a, [a, b, c]).map((s) => s.id)).toEqual(['b']);
    expect(shadowers(b, [a, b, c])).toHaveLength(0);
  });
});

describe('outputKind', () => {
  it('classifies stream / error / data / empty', () => {
    expect(outputKind(stub({ output: { stream: [{}] } })).label).toBe('Stream 1');
    expect(outputKind(stub({ output: { error: 'x', code: 5 } })).label).toBe('Error');
    expect(outputKind(stub({ output: { data: {} } })).label).toBe('Data');
    expect(outputKind(stub({ output: {} })).label).toBe('Empty');
  });
});

describe('globToRegExp', () => {
  it('translates * and ? and anchors', () => {
    expect(globToRegExp('foo*').test('foobar')).toBe(true);
    expect(globToRegExp('foo*').test('xfoobar')).toBe(false);
    expect(globToRegExp('a?c').test('abc')).toBe(true);
    expect(globToRegExp('a?c').test('ac')).toBe(false);
  });
  it('escapes regex metachars in the literal part', () => {
    expect(globToRegExp('a.b').test('a.b')).toBe(true);
    expect(globToRegExp('a.b').test('axb')).toBe(false);
  });
});

describe('evalMatcherFields', () => {
  const m = (o: Partial<StubInput>) => o as StubInput;

  it('equals: deep compares each field', () => {
    const rows = evalMatcherFields({ a: 1, b: 2 }, m({ equals: { a: 1, b: 3 } }));
    expect(rows.find((r) => r.field === 'a')?.ok).toBe(true);
    expect(rows.find((r) => r.field === 'b')?.ok).toBe(false);
  });

  it('matches: evaluates regex per field', () => {
    const rows = evalMatcherFields({ id: 'PROD_9', bad: 'x' }, m({ matches: { id: 'PROD_\\w+', bad: '^\\d+$' } }));
    expect(rows.find((r) => r.field === 'id')?.ok).toBe(true);
    expect(rows.find((r) => r.field === 'bad')?.ok).toBe(false);
  });

  it('contains: string substring', () => {
    const rows = evalMatcherFields({ name: 'hello world' }, m({ contains: { name: 'world' } }));
    expect(rows[0].ok).toBe(true);
    const rows2 = evalMatcherFields({ name: 'hello' }, m({ contains: { name: 'zzz' } }));
    expect(rows2[0].ok).toBe(false);
  });

  it('contains: numeric field is equality, not substring (15 does not contain 5)', () => {
    expect(evalMatcherFields({ rating: 5 }, m({ contains: { rating: 5 } }))[0].ok).toBe(true);
    expect(evalMatcherFields({ rating: 15 }, m({ contains: { rating: 5 } }))[0].ok).toBe(false);
  });

  it('contains: array subset', () => {
    expect(evalMatcherFields({ tags: ['a', 'b', 'c'] }, m({ contains: { tags: ['a', 'c'] } }))[0].ok).toBe(true);
    expect(evalMatcherFields({ tags: ['a'] }, m({ contains: { tags: ['a', 'z'] } }))[0].ok).toBe(false);
  });

  it('glob: wildcard per field', () => {
    const rows = evalMatcherFields({ path: '/api/v1/x' }, m({ glob: { path: '/api/*' } }));
    expect(rows[0].ok).toBe(true);
  });

  it('absent field is a mismatch, and reports the kind', () => {
    const rows = evalMatcherFields({}, m({ equals: { a: 1 } }));
    expect(rows[0]).toMatchObject({ field: 'a', kind: 'equals', ok: false, actual: undefined });
  });

  it('spans multiple matcher kinds at once', () => {
    const rows = evalMatcherFields({ a: 'PROD_1', b: 'x' }, m({ matches: { a: 'PROD_\\w+' }, equals: { b: 'y' } }));
    expect(rows).toHaveLength(2);
    expect(rows.find((r) => r.kind === 'matches')?.ok).toBe(true);
    expect(rows.find((r) => r.kind === 'equals')?.ok).toBe(false);
  });
});

describe('evalMatcherFields anyOf', () => {
  const m = (o: Partial<StubInput>) => o as StubInput;

  it('picks the closest alternative (fewest mismatches)', () => {
    const matcher = m({ anyOf: [
      { equals: { a: 1, b: 2 } },   // 1 mismatch vs {a:1,b:9}
      { equals: { a: 9, b: 9 } },   // 2 mismatches
    ] });
    const rows = evalMatcherFields({ a: 1, b: 9 }, matcher);
    // Best alt is #1; only b mismatches.
    expect(rows.filter((r) => !r.ok).map((r) => r.field)).toEqual(['b']);
  });

  it('returns no mismatches when one alternative fully matches', () => {
    const matcher = m({ anyOf: [
      { equals: { a: 9 } },
      { matches: { a: '\\d+' } },
    ] });
    const rows = evalMatcherFields({ a: '123' }, matcher);
    expect(rows.filter((r) => !r.ok)).toHaveLength(0);
  });

  it('direct kinds take precedence over anyOf', () => {
    const matcher = m({ equals: { a: 1 }, anyOf: [{ equals: { z: 9 } }] });
    const rows = evalMatcherFields({ a: 2 }, matcher);
    expect(rows).toHaveLength(1);
    expect(rows[0].field).toBe('a');
  });
});
