// @vitest-environment jsdom
import { describe, it, expect, beforeEach } from 'vitest';
import { stashClone, takeClone } from './clone';

describe('clone stash/take', () => {
  beforeEach(() => sessionStorage.clear());

  it('round-trips a plain stub', () => {
    stashClone({ service: 'S', method: 'M', input: { equals: { a: 1 } } });
    expect(takeClone()).toEqual({ service: 'S', method: 'M', input: { equals: { a: 1 } } });
  });

  it('drops the id so a clone does not overwrite its source', () => {
    stashClone({ id: 'abc', service: 'S', method: 'M' });

    const clone = takeClone();

    expect(clone).not.toBeNull();
    expect(clone).not.toHaveProperty('id');
    expect(clone?.service).toBe('S');
  });

  it('take consumes the stored value (one-shot)', () => {
    stashClone({ x: 1 });
    expect(takeClone()).not.toBeNull();
    expect(takeClone()).toBeNull();
  });

  it('strips prototype-pollution keys before storage (S8475)', () => {
    stashClone(JSON.parse('{"service":"S","__proto__":{"polluted":true},"nested":{"constructor":{"bad":1},"ok":2}}'));

    const stored = JSON.parse(sessionStorage.getItem('gripmock.clone')!);
    expect(Object.prototype.hasOwnProperty.call(stored, '__proto__')).toBe(false);
    expect(Object.prototype.hasOwnProperty.call(stored.nested, 'constructor')).toBe(false);
    expect(stored.nested.ok).toBe(2);
    expect(({} as Record<string, unknown>).polluted).toBeUndefined();
  });

  it('sanitizes again on read', () => {
    sessionStorage.setItem('gripmock.clone', '{"__proto__":{"p":1},"keep":3}');
    const out = takeClone()!;
    expect(Object.prototype.hasOwnProperty.call(out, '__proto__')).toBe(false);
    expect(out.keep).toBe(3);
  });
});
