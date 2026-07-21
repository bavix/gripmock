import { describe, it, expect } from 'vitest';
import { toYaml } from './toYaml';

describe('toYaml', () => {
  it('renders scalars and plain keys unquoted', () => {
    expect(toYaml({ service: 'pkg.Svc', priority: 5, on: true })).toBe(
      'service: pkg.Svc\npriority: 5\non: true',
    );
  });

  it('quotes strings with spaces/specials and yaml-ambiguous words', () => {
    expect(toYaml({ msg: 'hello world' })).toContain('"hello world"');
    expect(toYaml({ v: 'true' })).toContain('"true"');
    expect(toYaml({ v: '42' })).toContain('"42"');
  });

  it('escapes quotes and backslashes', () => {
    expect(toYaml({ s: 'a"b\\c' })).toContain('"a\\"b\\\\c"');
  });

  it('nests objects with indentation', () => {
    const y = toYaml({ input: { equals: { a: 1 } } });
    expect(y).toBe('input:\n  equals:\n    a: 1');
  });

  it('renders arrays of objects as dash items', () => {
    const y = toYaml({ stream: [{ n: 1 }, { n: 2 }] });
    expect(y).toContain('stream:');
    expect(y.match(/- /g)?.length).toBe(2);
  });

  it('drops null/empty values entirely', () => {
    expect(toYaml({ a: null, b: {}, c: [], d: '' })).toBe('');
    expect(toYaml({ keep: 1, drop: null })).toBe('keep: 1');
  });

  it('keeps zero and false', () => {
    const y = toYaml({ n: 0, f: false });
    expect(y).toContain('n: 0');
    expect(y).toContain('f: false');
  });
});
